package gui

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"qzone-history/internal/delivery/app"
	"qzone-history/internal/delivery/bootstrap"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/pkg/applog"
	"qzone-history/pkg/loghub"
	"qzone-history/pkg/offset"
	"qzone-history/pkg/paths"
	"qzone-history/version"
)

//go:embed dashboard.html
var dashboardHTML []byte

const DefaultPort = 17890

type Server struct {
	cfg    *config.Config
	hub    *loghub.Hub
	mu     sync.Mutex
	runCtx context.Context
	cancel context.CancelFunc
	running bool

	sessionStack *bootstrap.Stack
	loggedUser   *entity.User
	qrsig        string
	httpSrv      *http.Server
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		hub: loghub.Default(),
	}
	s.restoreSession()
	return s
}

func (s *Server) restoreSession() {
	if err := s.ensureSession(); err != nil {
		return
	}
	u, ok, err := s.sessionStack.App.Auth().CheckLocalLoginStatus(context.Background())
	if err != nil || !ok {
		return
	}
	s.mu.Lock()
	s.loggedUser = u
	s.mu.Unlock()
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/events", s.handleSSE)
	mux.HandleFunc("/api/recommend", s.handleRecommend)
	mux.HandleFunc("/api/login/qrcode", s.handleLoginQR)
	mux.HandleFunc("/api/login/poll", s.handleLoginPoll)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/exit", s.handleExit)
	mux.HandleFunc("/api/open-viewer", s.handleOpenViewer)

	addr := fmt.Sprintf("127.0.0.1:%d", DefaultPort)
	applog.Redirect(s.hub)
	s.hub.Logf("QQ空间历史恢复工具 %s | 作者: %s | QQ: %s", version.Version, version.Author, version.ContactQQ)
	s.hub.Logf("控制台已启动: http://%s", addr)
	s.hub.Log("程序在后台运行，可关闭浏览器窗口，任务不会中断")
	s.hub.Log("不需要后台运行时，请点击页面底部「关闭并结束进程」")

	go openBrowser("http://" + addr)

	s.httpSrv = &http.Server{Addr: addr, Handler: mux}
	return s.httpSrv.ListenAndServe()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	st := s.hub.GetStatus()
	resp := map[string]interface{}{
		"phase":           st.Phase,
		"running":         st.Running,
		"activityCount":   st.ActivityCount,
		"earliestDate":    st.EarliestDate,
		"targetYear":      st.TargetYear,
		"maxOffset":       st.MaxOffset,
		"userQQ":          st.UserQQ,
		"done":            st.Done,
		"error":           st.Error,
		"viewerPath":      st.ViewerPath,
		"progressPercent": st.ProgressPercent,
		"logs":            s.hub.Logs(),
	}
	s.mu.Lock()
	if s.loggedUser != nil {
		resp["loggedIn"] = true
		resp["qq"] = s.loggedUser.QQ
		resp["nickname"] = s.loggedUser.Nickname
	}
	s.mu.Unlock()
	writeJSON(w, resp)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no sse", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	st := s.hub.GetStatus()
	fmt.Fprintf(w, "event: status\ndata: %s\n\n", mustJSON(st))
	flusher.Flush()

	ch := s.hub.Subscribe()
	defer s.hub.Unsubscribe(ch)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", mustJSON(e))
			flusher.Flush()
		case <-ticker.C:
			st := s.hub.GetStatus()
			fmt.Fprintf(w, "event: status\ndata: %s\n\n", mustJSON(st))
			flusher.Flush()
		}
	}
}

func (s *Server) handleRecommend(w http.ResponseWriter, r *http.Request) {
	year := parseInt(r.URL.Query().Get("year"), 2017)
	maxOff := offset.RecommendMaxOffset(year)
	if q := r.URL.Query().Get("maxOffset"); q != "" {
		if v := parseInt(q, 0); v >= 500 {
			maxOff = v
		}
	}
	lo, hi := offset.EstimateScan(year, maxOff)
	writeJSON(w, map[string]interface{}{
		"maxOffset":       maxOff,
		"hint":            offset.RecommendHint(year),
		"estimate":        offset.EstimateScanText(year, maxOff),
		"estimateMinMins": lo,
		"estimateMaxMins": hi,
	})
}

func (s *Server) ensureSession() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionStack != nil {
		return nil
	}
	sessionDB := filepath.Join(paths.ExeDir(), "session.db")
	stack, err := bootstrap.Build(s.cfg, sessionDB)
	if err != nil {
		return err
	}
	s.sessionStack = stack
	return nil
}

func (s *Server) handleLoginQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	if err := s.ensureSession(); err != nil {
		writeErr(w, err)
		return
	}
	qr, qrsig, err := s.sessionStack.App.Auth().GetLoginQRCode(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	s.mu.Lock()
	s.qrsig = qrsig
	s.mu.Unlock()
	s.hub.Log("已生成登录二维码，请使用手机 QQ 扫描")
	writeJSON(w, map[string]string{
		"qrsig":  qrsig,
		"qrcode": base64.StdEncoding.EncodeToString(qr),
	})
}

func (s *Server) handleLoginPoll(w http.ResponseWriter, r *http.Request) {
	qrsig := r.URL.Query().Get("qrsig")
	if qrsig == "" {
		writeErr(w, fmt.Errorf("缺少 qrsig"))
		return
	}
	if err := s.ensureSession(); err != nil {
		writeErr(w, err)
		return
	}
	auth := s.sessionStack.App.Auth()
	status, res, err := auth.CheckQRCodeLoginStatus(r.Context(), qrsig)
	if err != nil {
		writeErr(w, err)
		return
	}
	switch status {
	case entity.LoginStatusSuccess:
		user, err := auth.CompleteLogin(r.Context(), res)
		if err != nil {
			writeErr(w, err)
			return
		}
		s.mu.Lock()
		s.loggedUser = user
		s.mu.Unlock()
		s.hub.Logf("登录成功: %s (%s)", user.Nickname, user.QQ)
		writeJSON(w, map[string]interface{}{
			"loggedIn": true,
			"qq":       user.QQ,
			"nickname": user.Nickname,
		})
	case entity.LoginStatusExpired:
		writeJSON(w, map[string]interface{}{"expired": true})
	case entity.LoginStatusScanning:
		writeJSON(w, map[string]interface{}{"message": "认证中..."})
	default:
		writeJSON(w, map[string]interface{}{"message": "等待扫描..."})
	}
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		writeErr(w, fmt.Errorf("任务正在运行中"))
		return
	}
	user := s.loggedUser
	s.mu.Unlock()
	if user == nil {
		if err := s.ensureSession(); err != nil {
			writeErr(w, err)
			return
		}
		u, ok, err := s.sessionStack.App.Auth().CheckLocalLoginStatus(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		if !ok {
			writeErr(w, fmt.Errorf("请先扫码登录"))
			return
		}
		user = u
		s.mu.Lock()
		s.loggedUser = user
		s.mu.Unlock()
	}

	var req struct {
		TargetYear int `json:"targetYear"`
		MaxOffset  int `json:"maxOffset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, err)
		return
	}
	if req.MaxOffset < 500 {
		req.MaxOffset = 500
	}
	if req.TargetYear < 2005 {
		req.TargetYear = 2005
	}

	s.hub.Reset()
	s.hub.SetStatus(func(st *loghub.Status) {
		st.Running = true
		st.UserQQ = user.QQ
		st.TargetYear = req.TargetYear
		st.MaxOffset = req.MaxOffset
		st.Phase = "启动中"
	})

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.runCtx = ctx
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	go s.runJob(ctx, user, app.RunOptions{
		TargetYear: req.TargetYear,
		MaxOffset:  req.MaxOffset,
	})

	writeJSON(w, map[string]string{"ok": "started"})
}

func (s *Server) runJob(ctx context.Context, user *entity.User, opts app.RunOptions) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.cancel = nil
		s.mu.Unlock()
	}()

	if _, err := paths.EnsureUserDir(user.QQ); err != nil {
		s.hub.Logf("创建目录失败: %v", err)
		return
	}

	dbPath := paths.UserDBPath(user.QQ)
	stack, err := bootstrap.Build(s.cfg, dbPath)
	if err != nil {
		s.hub.Logf("初始化用户数据库失败: %v", err)
		return
	}
	defer stack.DB.Close()

	if _, err := stack.App.Auth().RefreshLogin(ctx, user); err != nil {
		s.hub.Logf("同步登录状态: %v（继续尝试）", err)
	}

	s.hub.Logf("开始恢复，目标年份 ≥ %d，max offset = %d", opts.TargetYear, opts.MaxOffset)
	s.hub.Log(offset.RecommendHint(opts.TargetYear))
	s.hub.Log(offset.EstimateScanText(opts.TargetYear, opts.MaxOffset))

	if err := stack.App.RunPipeline(ctx, user, opts); err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			s.hub.Log("任务已停止")
			s.hub.SetStatus(func(st *loghub.Status) {
				st.Running = false
				st.Phase = "已停止"
			})
			return
		}
	}
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	running := s.running
	s.mu.Unlock()
	if !running || cancel == nil {
		writeErr(w, fmt.Errorf("当前没有运行中的任务"))
		return
	}
	s.hub.Log("收到停止请求，正在中断当前任务…")
	s.hub.SetStatus(func(st *loghub.Status) {
		st.Phase = "正在停止"
	})
	cancel()
	writeJSON(w, map[string]string{"ok": "stopped"})
}

func (s *Server) handleExit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", 405)
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		s.hub.Log("正在停止任务并退出程序…")
		cancel()
	} else {
		s.hub.Log("正在退出程序…")
	}
	writeJSON(w, map[string]string{"ok": "exiting"})
	go func() {
		time.Sleep(400 * time.Millisecond)
		if s.httpSrv != nil {
			shutdownCtx, release := context.WithTimeout(context.Background(), 3*time.Second)
			_ = s.httpSrv.Shutdown(shutdownCtx)
			release()
		}
		if s.sessionStack != nil && s.sessionStack.DB != nil {
			_ = s.sessionStack.DB.Close()
		}
		os.Exit(0)
	}()
}

func (s *Server) handleOpenViewer(w http.ResponseWriter, r *http.Request) {
	st := s.hub.GetStatus()
	path := st.ViewerPath
	if path == "" && st.UserQQ != "" {
		path = paths.ViewerHTMLPath(st.UserQQ)
	}
	if path == "" {
		writeErr(w, fmt.Errorf("浏览页尚未生成"))
		return
	}
	if _, err := os.Stat(path); err != nil {
		writeErr(w, fmt.Errorf("文件不存在: %s", path))
		return
	}
	abs, _ := filepath.Abs(path)
	url := "file:///" + filepath.ToSlash(abs)
	openBrowser(url)
	writeJSON(w, map[string]string{"url": url, "path": abs})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func parseInt(s string, def int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

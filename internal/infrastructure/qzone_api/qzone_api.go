package qzone_api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/pkg/loghub"
	"qzone-history/pkg/timeparse"
	"qzone-history/pkg/utils"
	"regexp"
	"strings"
	"time"
)

type QzoneAPIClient interface {
	GetLoginQRCode() ([]byte, string, error)

	CheckLoginStatus(qrsig string) (entity.LoginStatus, string, error)

	CompleteLogin(responseText string) (map[string]string, error)

	GetUserInfo(cookies map[string]string) (*entity.User, error)

	GetActivities(cookies map[string]string, offset, count int) ([]*entity.Activity, error)

	GetAllActivities(cookies map[string]string, opts FetchOptions) ([]*entity.Activity, error)

	GetVisibleMoments(cookies map[string]string) ([]entity.Moment, error)

	GetBoardMessages(cookies map[string]string) ([]entity.BoardMessage, error)
}

type qzoneAPIClient struct {
	httpClient *http.Client
	config     *config.Config
}

func NewQzoneAPIClient(config *config.Config) QzoneAPIClient {
	return &qzoneAPIClient{
		httpClient: &http.Client{
			Timeout: time.Second * 60,
		},
		config: config,
	}
}

func (c *qzoneAPIClient) setBrowserHeaders(req *http.Request, uin string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	if uin != "" {
		req.Header.Set("Referer", fmt.Sprintf("https://user.qzone.qq.com/%s", uin))
	}
}

func (c *qzoneAPIClient) setCorsHeaders(req *http.Request, uin string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if uin != "" {
		req.Header.Set("Referer", fmt.Sprintf("https://user.qzone.qq.com/%s/main", uin))
	}
}

func (c *qzoneAPIClient) doGet(ctx context.Context, cookies map[string]string, url, uin string, useCors bool) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	if useCors {
		c.setCorsHeaders(req, uin)
	} else {
		c.setBrowserHeaders(req, uin)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(body), "waf.tencent.com") {
		return nil, fmt.Errorf("请求被腾讯 WAF 拦截")
	}
	return body, nil
}

func mergeCookies(base map[string]string, resp *http.Response) map[string]string {
	merged := make(map[string]string, len(base)+8)
	for k, v := range base {
		merged[k] = v
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Value != "" || merged[cookie.Name] == "" {
			merged[cookie.Name] = cookie.Value
		}
	}
	return merged
}

func (c *qzoneAPIClient) warmUpSession(cookies map[string]string) map[string]string {
	uin := utils.ExtractUin(cookies)
	urls := []string{
		fmt.Sprintf("https://user.qzone.qq.com/%s", uin),
		fmt.Sprintf("https://user.qzone.qq.com/%s/311", uin),
	}
	merged := cookies
	for _, url := range urls {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		for name, value := range merged {
			req.AddCookie(&http.Cookie{Name: name, Value: value})
		}
		c.setBrowserHeaders(req, uin)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}
		merged = mergeCookies(merged, resp)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return merged
}

func (c *qzoneAPIClient) GetLoginQRCode() ([]byte, string, error) {
	resp, err := c.httpClient.Get(c.config.QzoneAPI.QRCodeURL)
	if err != nil {
		return nil, "", fmt.Errorf("获取二维码失败: %w", err)
	}
	defer resp.Body.Close()

	var qrsig string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "qrsig" {
			qrsig = cookie.Value
			break
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("读取二维码数据失败: %w", err)
	}

	return body, qrsig, nil
}

func (c *qzoneAPIClient) CheckLoginStatus(qrsig string) (entity.LoginStatus, string, error) {
	ptqrtoken := utils.GeneratePtqrToken(qrsig)

	loginURL := fmt.Sprintf("%s?u1=https%%3A%%2F%%2Fqzs.qq.com%%2Fqzone%%2Fv5%%2Floginsucc.html%%3Fpara%%3Dizone&ptqrtoken=%s&ptredirect=0&h=1&t=1&g=1&from_ui=1&ptlang=2052&action=0-0-%d&js_ver=20032614&js_type=1&login_sig=&pt_uistyle=40&aid=549000912&daid=5&",
		c.config.QzoneAPI.LoginURL, ptqrtoken, time.Now().Unix())

	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		return entity.LoginStatusWaiting, "", fmt.Errorf("创建登录请求失败: %w", err)
	}

	req.AddCookie(&http.Cookie{Name: "qrsig", Value: qrsig})
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return entity.LoginStatusWaiting, "", fmt.Errorf("发送登录请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return entity.LoginStatusWaiting, "", fmt.Errorf("读取登录响应失败: %w", err)
	}

	responseText := string(body)
	if strings.Contains(responseText, "二维码未失效") {
		return entity.LoginStatusWaiting, "", nil
	} else if strings.Contains(responseText, "二维码认证中") {
		return entity.LoginStatusScanning, "", nil
	} else if strings.Contains(responseText, "二维码已失效") {
		return entity.LoginStatusExpired, "", nil
	} else if strings.Contains(responseText, "登录成功") {
		return entity.LoginStatusSuccess, responseText, nil
	}

	return entity.LoginStatusWaiting, "", nil
}

func (c *qzoneAPIClient) CompleteLogin(responseText string) (map[string]string, error) {
	re := regexp.MustCompile(`ptsigx=(.*?)&`)
	matches := re.FindStringSubmatch(responseText)
	if len(matches) < 2 {
		return nil, fmt.Errorf("无法提取 ptsigx")
	}
	sigx := matches[1]

	re = regexp.MustCompile(`uin=(\d+)`)
	matches = re.FindStringSubmatch(responseText)
	if len(matches) < 2 {
		return nil, fmt.Errorf("无法获取 uin")
	}
	uin := matches[1]

	checkSigURL := fmt.Sprintf("https://ptlogin2.qzone.qq.com/check_sig?pttype=1&uin=%s&service=ptqrlogin&nodirect=0&ptsigx=%s&s_url=https%%3A%%2F%%2Fqzs.qq.com%%2Fqzone%%2Fv5%%2Floginsucc.html%%3Fpara%%3Dizone&f_url=&ptlang=2052&ptredirect=100&aid=549000912&daid=5&j_later=0&low_login_hour=0&regmaster=0&pt_login_type=3&pt_aid=0&pt_aaid=16&pt_light=0&pt_3rd_aid=0",
		uin, sigx)

	c.httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	defer func() {
		c.httpClient.CheckRedirect = nil
	}()

	cookies := make(map[string]string)
	currentURL := checkSigURL
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建 check_sig 请求失败: %w", err)
		}
		for name, value := range cookies {
			req.AddCookie(&http.Cookie{Name: name, Value: value})
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("发送 check_sig 请求失败: %w", err)
		}

		cookies = mergeCookies(cookies, resp)
		location := resp.Header.Get("Location")
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if location == "" || resp.StatusCode < 300 || resp.StatusCode >= 400 {
			break
		}
		if strings.HasPrefix(location, "//") {
			location = "https:" + location
		} else if strings.HasPrefix(location, "/") {
			location = "https://user.qzone.qq.com" + location
		}
		currentURL = location
	}

	return c.warmUpSession(cookies), nil
}

func (c *qzoneAPIClient) GetUserInfo(cookies map[string]string) (*entity.User, error) {
	uin := utils.ExtractUin(cookies)
	skey := cookies["p_skey"]
	g_tk := utils.GenerateGTK(skey)

	url := fmt.Sprintf("https://r.qzone.qq.com/fcg-bin/cgi_get_portrait.fcg?g_tk=%s&uins=%s", g_tk, uin)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建获取用户信息请求失败: %w", err)
	}

	for name, value := range cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	c.setBrowserHeaders(req, uin)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送获取用户信息请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	info := string(body)
	info = strings.TrimSpace(info)

	if strings.HasPrefix(info, "portraitCallBack(") {
		info = strings.TrimPrefix(info, "portraitCallBack(")
		info = strings.TrimSuffix(info, ")")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(info), &result); err != nil {
		return nil, fmt.Errorf("解析用户信息响应失败: %w", err)
	}

	userData, ok := result[uin].([]interface{})
	if !ok || len(userData) < 7 {
		return nil, fmt.Errorf("用户信息格式不正确")
	}

	nickname, ok := userData[6].(string)
	if !ok {
		return nil, fmt.Errorf("无法获取昵称")
	}

	return &entity.User{
		QQ:       uin,
		Nickname: nickname,
		Cookies:  cookies,
	}, nil
}

func (c *qzoneAPIClient) getActivityCount(cookies map[string]string) (int, error) {
	uin := utils.ExtractUin(cookies)
	total := 0
	offset := 0
	pageSize := 100

	for {
		body, err := c.fetchFeedBody(context.Background(), cookies, uin, offset, pageSize)
		if err != nil {
			return total, err
		}
		processed := utils.ProcessFeedResponse(string(body))
		if !strings.Contains(processed, "li") {
			break
		}
		activities, err := c.parseActivitiesFromHTML(processed, uin)
		if err != nil {
			return total, err
		}
		if len(activities) == 0 {
			break
		}
		total += len(activities)
		if !utils.HasMoreFeeds(string(body)) {
			break
		}
		offset += len(activities)
		time.Sleep(200 * time.Millisecond)
	}

	if total > 0 {
		return total, nil
	}

	lowerBound := 0
	upperBound := 10000000
	mid := upperBound / 2
	for lowerBound <= upperBound {
		body, err := c.fetchFeedBody(context.Background(), cookies, uin, mid, 100)
		if err != nil {
			return 0, err
		}
		processed := utils.ProcessFeedResponse(string(body))
		if strings.Contains(processed, "li") {
			lowerBound = mid + 1
		} else {
			upperBound = mid - 1
		}
		mid = (lowerBound + upperBound) / 2
	}
	return mid, nil
}

func (c *qzoneAPIClient) fetchFeedBody(ctx context.Context, cookies map[string]string, uin string, offset, count int) ([]byte, error) {
	return c.fetchFeedBodyWithRange(ctx, cookies, uin, 0, 0, offset, count)
}

func (c *qzoneAPIClient) fetchFeedBodyWithRange(ctx context.Context, cookies map[string]string, uin string, beginTime, endTime int64, offset, count int) ([]byte, error) {
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds2_html_pav_all?uin=%s&begin_time=%d&end_time=%d&getappnotification=1&getnotifi=1&has_get_key=0&offset=%d&set=0&count=%d&useutf8=1&outputhtmlfeed=1&scope=1&format=jsonp&g_tk=%s",
		uin, beginTime, endTime, offset, count, gTk,
	)
	body, err := c.doGet(ctx, cookies, url, uin, true)
	if err == nil {
		return body, nil
	}

	legacyURL := fmt.Sprintf(
		"https://user.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds2_html_pav_all?uin=%s&begin_time=%d&end_time=%d&getappnotification=1&getnotifi=1&has_get_key=0&offset=%d&set=0&count=%d&useutf8=1&outputhtmlfeed=1&scope=1&format=jsonp&g_tk=%s",
		uin, beginTime, endTime, offset, count, gTk,
	)
	return c.doGet(ctx, cookies, legacyURL, uin, true)
}

func (c *qzoneAPIClient) GetAllActivities(cookies map[string]string, opts FetchOptions) ([]*entity.Activity, error) {
	if opts.MaxOffset <= 0 {
		opts.MaxOffset = 25000
	}
	return c.getAllActivitiesWithOpts(fetchCtx(opts), cookies, opts, defaultReporter())
}

func (c *qzoneAPIClient) getAllActivitiesWithOpts(ctx context.Context, cookies map[string]string, opts FetchOptions, rep ProgressReporter) ([]*entity.Activity, error) {
	cookies = c.warmUpSession(cookies)
	uin := utils.ExtractUin(cookies)
	seen := make(map[string]struct{})
	var allActivities []*entity.Activity
	var earliestUnix int64
	pageSize := 100
	maxOff := opts.MaxOffset
	scanLog := newScanProgressLog()

	report := func(phase string) {
		rep.OnActivities(len(allActivities), earliestUnix, phase)
	}

	appendUnique := func(batch []*entity.Activity) int {
		added := 0
		for _, a := range batch {
			key := activityDedupKey(a)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			allActivities = append(allActivities, a)
			added++
		}
		return added
	}

	trackEarliest := func(raw string) {
		if ts := utils.ExtractMinAbstime(raw); ts > 0 && (earliestUnix == 0 || ts < earliestUnix) {
			earliestUnix = ts
		}
	}

	trackBatch := func(batch []*entity.Activity, raw string) {
		trackEarliest(raw)
		refYear := timeparse.RefYearFromEarliest(earliestUnix, opts.TargetYear)
		for _, a := range batch {
			if a == nil {
				continue
			}
			if a.Timestamp.IsZero() && a.TimeText != "" {
				a.Timestamp = timeparse.ParseCN(a.TimeText, refYear)
			}
			if !a.Timestamp.IsZero() {
				ts := a.Timestamp.Unix()
				if earliestUnix == 0 || ts < earliestUnix {
					earliestUnix = ts
				}
			}
		}
	}

	fetchAtOffset := func(startOffset int, maxPages int, phase string) error {
		offset := startOffset
		for page := 0; page < maxPages; page++ {
			if err := checkScanCtx(ctx); err != nil {
				return err
			}
			batch, hasMore, raw, err := c.fetchActivitiesPageRaw(ctx, cookies, uin, offset, pageSize)
			if err != nil {
				if strings.Contains(err.Error(), "need login") {
					return nil
				}
				if err == context.Canceled {
					return err
				}
				break
			}
			if len(batch) == 0 {
				break
			}
			trackBatch(batch, raw)
			appendUnique(batch)
			report(phase)
			scanLog.tick(phase, fmt.Sprintf("offset %d", offset), len(allActivities), earliestUnix)
			offset += len(batch)
			if !hasMore && startOffset == 0 {
				break
			}
			if !hasMore && page >= 2 {
				break
			}
			if err := sleepCtx(ctx, 80*time.Millisecond); err != nil {
				return err
			}
		}
		return nil
	}

	scanLog.phaseStart("顺序分页", "从最新动态开始拉取")
	report("顺序分页")
	if err := fetchAtOffset(0, 100, "顺序分页"); err != nil {
		loghub.Default().Log("活动抓取已停止")
		return allActivities, err
	}
	if err := checkScanCtx(ctx); err != nil {
		loghub.Default().Log("活动抓取已停止")
		return allActivities, err
	}

	scanLog.phaseStart("稀疏深扫", fmt.Sprintf("offset 700~%d，步长 100（较耗时，请耐心等待）", clampMax(6000, maxOff)))
	report("稀疏深扫")
	for off := 700; off <= clampMax(6000, maxOff); off += 100 {
		if err := checkScanCtx(ctx); err != nil {
			return allActivities, err
		}
		if err := fetchAtOffset(off, 10, "稀疏深扫"); err != nil {
			return allActivities, err
		}
		scanLog.tick("稀疏深扫", fmt.Sprintf("进度 offset %d/%d", off, clampMax(6000, maxOff)), len(allActivities), earliestUnix)
		if err := sleepCtx(ctx, 60*time.Millisecond); err != nil {
			return allActivities, err
		}
	}

	if maxOff >= 2500 {
		scanLog.phaseStart("细扫 2500-3200", "加密 offset 扫描")
		for off := 2500; off <= clampMax(3200, maxOff); off += 25 {
			if err := checkScanCtx(ctx); err != nil {
				return allActivities, err
			}
			if err := fetchAtOffset(off, 5, "细扫 2500-3200"); err != nil {
				return allActivities, err
			}
			if off%200 == 0 {
				scanLog.tick("细扫 2500-3200", fmt.Sprintf("offset %d", off), len(allActivities), earliestUnix)
			}
		}
	}
	if maxOff >= 4500 {
		scanLog.phaseStart("细扫 4500-5500", "加密 offset 扫描")
		for off := 4500; off <= clampMax(5500, maxOff); off += 50 {
			if err := checkScanCtx(ctx); err != nil {
				return allActivities, err
			}
			if err := fetchAtOffset(off, 5, "细扫 4500-5500"); err != nil {
				return allActivities, err
			}
			if off%250 == 0 {
				scanLog.tick("细扫 4500-5500", fmt.Sprintf("offset %d", off), len(allActivities), earliestUnix)
			}
		}
	}

	if maxOff > 6000 {
		scanLog.phaseStart("极限深扫", fmt.Sprintf("offset 6000~%d", maxOff))
		report("极限深扫")
		for off := 6000; off <= maxOff; off += 100 {
			if err := checkScanCtx(ctx); err != nil {
				return allActivities, err
			}
			if err := fetchAtOffset(off, 10, "极限深扫"); err != nil {
				return allActivities, err
			}
			scanLog.tick("极限深扫", fmt.Sprintf("offset %d/%d", off, maxOff), len(allActivities), earliestUnix)
			if err := sleepCtx(ctx, 50*time.Millisecond); err != nil {
				return allActivities, err
			}
		}
	}
	if maxOff >= 5500 {
		scanLog.phaseStart("超细扫", fmt.Sprintf("offset 5500~%d，步长 20", clampMax(15000, maxOff)))
		for off := 5500; off <= clampMax(15000, maxOff); off += 20 {
			if err := checkScanCtx(ctx); err != nil {
				return allActivities, err
			}
			if err := fetchAtOffset(off, 4, "超细扫"); err != nil {
				return allActivities, err
			}
			if off%500 == 0 {
				scanLog.tick("超细扫", fmt.Sprintf("offset %d", off), len(allActivities), earliestUnix)
			}
		}
	}
	if maxOff > 15000 {
		scanLog.phaseStart("wildcard", fmt.Sprintf("offset 15000~%d", maxOff))
		for off := 15000; off <= maxOff; off += 250 {
			if err := checkScanCtx(ctx); err != nil {
				return allActivities, err
			}
			if err := fetchAtOffset(off, 3, "wildcard"); err != nil {
				return allActivities, err
			}
			scanLog.tick("wildcard", fmt.Sprintf("offset %d", off), len(allActivities), earliestUnix)
			if err := sleepCtx(ctx, 40*time.Millisecond); err != nil {
				return allActivities, err
			}
		}
	}

	scanLog.phaseStart("历史时间窗", fmt.Sprintf("按年/半年切片扫描 %d 年及更早", scanTargetYear(opts.TargetYear)))
	report("历史时间窗")
	for i, win := range buildYearlyWindows(opts.TargetYear) {
		if err := checkScanCtx(ctx); err != nil {
			return allActivities, err
		}
		y := time.Unix(win[0], 0).Year()
		_ = c.fetchActivitiesInTimeWindow(ctx, cookies, uin, win[0], win[1], pageSize, 15, appendUnique, trackBatch, report, "历史时间窗")
		if i%4 == 0 {
			scanLog.tick("历史时间窗", fmt.Sprintf("%d 年片段", y), len(allActivities), earliestUnix)
		}
		if err := sleepCtx(ctx, 40*time.Millisecond); err != nil {
			return allActivities, err
		}
	}

	scanLog.phaseStart("set/scope 变体", "尝试不同 API 参数组合")
	report("set/scope 变体")
	barAdd := func(batch []*entity.Activity) int {
		trackBatch(batch, "")
		return appendUnique(batch)
	}
	for set := 0; set <= 3; set++ {
		if err := checkScanCtx(ctx); err != nil {
			return allActivities, err
		}
		c.fetchActivitiesWithSet(ctx, cookies, uin, set, pageSize, maxOff, barAdd, report)
		scanLog.tick("set/scope", fmt.Sprintf("set=%d 完成", set), len(allActivities), earliestUnix)
	}
	if err := checkScanCtx(ctx); err != nil {
		return allActivities, err
	}
	c.fetchActivitiesWithScope(ctx, cookies, uin, 0, pageSize, maxOff, barAdd, report)
	if err := checkScanCtx(ctx); err != nil {
		return allActivities, err
	}
	c.fetchActivitiesWithScope(ctx, cookies, uin, 1, pageSize, maxOff, barAdd, report)

	scanLog.phaseStart("feeds3 游标", "按时间游标深度回溯历史")
	report("feeds3 游标")
	checkpoints := buildFeeds3Checkpoints(opts.TargetYear)
	for i, bt := range checkpoints {
		if err := checkScanCtx(ctx); err != nil {
			return allActivities, err
		}
		c.fetchActivitiesFromFeeds3Starting(ctx, cookies, uin, bt, appendUnique, trackBatch, report)
		if i%5 == 0 || i == len(checkpoints)-1 {
			scanLog.tick("feeds3", fmt.Sprintf("游标 %d/%d", i+1, len(checkpoints)), len(allActivities), earliestUnix)
		}
	}

	report("完成")
	refYear := timeparse.RefYearFromEarliest(earliestUnix, opts.TargetYear)
	for _, a := range allActivities {
		if a == nil {
			continue
		}
		if a.Timestamp.IsZero() && a.TimeText != "" {
			a.Timestamp = timeparse.ParseCN(a.TimeText, refYear)
		}
		if !a.Timestamp.IsZero() {
			ts := a.Timestamp.Unix()
			if earliestUnix == 0 || ts < earliestUnix {
				earliestUnix = ts
			}
		}
	}
	loghub.Default().Logf("活动抓取完成，共 %d 条，最早约 %s", len(allActivities), time.Unix(earliestUnix, 0).Format("2006-01-02"))
	if opts.TargetYear > 0 && earliestUnix > 0 {
		y := time.Unix(earliestUnix, 0).Year()
		if y > opts.TargetYear {
			loghub.Default().Logf("提示：最早仅到 %d 年，未达到目标 %d 年，可调大 max offset 后重试", y, opts.TargetYear)
		}
	}
	return allActivities, nil
}

func (c *qzoneAPIClient) fetchActivitiesPageRaw(ctx context.Context, cookies map[string]string, uin string, offset, count int) ([]*entity.Activity, bool, string, error) {
	body, err := c.fetchFeedBody(ctx, cookies, uin, offset, count)
	if err != nil {
		return nil, false, "", err
	}
	raw := string(body)
	if strings.Contains(raw, "need login") {
		return nil, false, raw, fmt.Errorf("need login")
	}
	processedHTML := utils.ProcessFeedResponse(raw)
	if !strings.Contains(processedHTML, "li") {
		return nil, false, raw, nil
	}
	activities, err := c.parseActivitiesFromHTML(processedHTML, uin)
	if err != nil {
		return nil, false, raw, err
	}
	return activities, utils.HasMoreFeeds(raw), raw, nil
}

func (c *qzoneAPIClient) fetchActivitiesWithSet(
	ctx context.Context,
	cookies map[string]string, uin string, set, pageSize, maxOff int,
	appendUnique func([]*entity.Activity) int, report func(string),
) {
	starts := []int{0, 800, 1500, 2200, 3000, 4000, 5000, 6000, 8000, 10000, 12000, 15000, 20000}
	for _, start := range starts {
		if err := checkScanCtx(ctx); err != nil {
			return
		}
		if start > maxOff {
			continue
		}
		offset := start
		for page := 0; page < 5; page++ {
			if err := checkScanCtx(ctx); err != nil {
				return
			}
			body, err := c.fetchFeedBodyWithSet(ctx, cookies, uin, set, offset, pageSize)
			if err != nil || strings.Contains(string(body), "need login") {
				return
			}
			processed := utils.ProcessFeedResponse(string(body))
			if !strings.Contains(processed, "li") {
				break
			}
			batch, err := c.parseActivitiesFromHTML(processed, uin)
			if err != nil || len(batch) == 0 {
				break
			}
			appendUnique(batch)
			report("set 变体")
			if !utils.HasMoreFeeds(string(body)) {
				break
			}
			offset += len(batch)
			if err := sleepCtx(ctx, 80*time.Millisecond); err != nil {
				return
			}
		}
	}
}

func (c *qzoneAPIClient) fetchFeedBodyWithSet(ctx context.Context, cookies map[string]string, uin string, set, offset, count int) ([]byte, error) {
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds2_html_pav_all?uin=%s&begin_time=0&end_time=0&getappnotification=1&getnotifi=1&has_get_key=0&offset=%d&set=%d&count=%d&useutf8=1&outputhtmlfeed=1&scope=1&format=jsonp&g_tk=%s",
		uin, offset, set, count, gTk,
	)
	return c.doGet(ctx, cookies, url, uin, true)
}

func (c *qzoneAPIClient) fetchActivitiesWithScope(
	ctx context.Context,
	cookies map[string]string, uin string, scope, pageSize, maxOff int,
	appendUnique func([]*entity.Activity) int, report func(string),
) {
	starts := []int{0, 3000, 6000, 9000, 12000, 15000, 20000}
	for _, start := range starts {
		if err := checkScanCtx(ctx); err != nil {
			return
		}
		if start > maxOff {
			continue
		}
		offset := start
		for page := 0; page < 6; page++ {
			if err := checkScanCtx(ctx); err != nil {
				return
			}
			body, err := c.fetchFeedBodyWithScope(ctx, cookies, uin, scope, offset, pageSize)
			if err != nil || strings.Contains(string(body), "need login") {
				return
			}
			processed := utils.ProcessFeedResponse(string(body))
			if !strings.Contains(processed, "li") {
				break
			}
			batch, err := c.parseActivitiesFromHTML(processed, uin)
			if err != nil || len(batch) == 0 {
				break
			}
			appendUnique(batch)
			report("scope 变体")
			if !utils.HasMoreFeeds(string(body)) {
				break
			}
			offset += len(batch)
			if err := sleepCtx(ctx, 80*time.Millisecond); err != nil {
				return
			}
		}
	}
}

func (c *qzoneAPIClient) fetchFeedBodyWithScope(ctx context.Context, cookies map[string]string, uin string, scope, offset, count int) ([]byte, error) {
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds2_html_pav_all?uin=%s&begin_time=0&end_time=0&getappnotification=1&getnotifi=1&has_get_key=0&offset=%d&set=0&count=%d&useutf8=1&outputhtmlfeed=1&scope=%d&format=jsonp&g_tk=%s",
		uin, offset, count, scope, gTk,
	)
	return c.doGet(ctx, cookies, url, uin, true)
}

func (c *qzoneAPIClient) fetchActivitiesFromFeeds3Starting(
	ctx context.Context,
	cookies map[string]string, uin string, startTime int64,
	appendUnique func([]*entity.Activity) int,
	trackBatch func([]*entity.Activity, string),
	report func(string),
) {
	begintime := startTime
	if begintime == 0 {
		begintime = time.Now().Unix()
	}
	for round := 0; round < 120; round++ {
		if err := checkScanCtx(ctx); err != nil {
			return
		}
		body, err := c.fetchLegacyFeeds3(ctx, cookies, uin, begintime, 50)
		if err != nil || strings.Contains(string(body), "need login") {
			return
		}
		raw := string(body)
		processed := utils.ProcessFeedResponse(raw)
		if !strings.Contains(processed, "li") {
			break
		}
		batch, err := c.parseActivitiesFromHTML(processed, uin)
		if err != nil || len(batch) == 0 {
			break
		}
		trackBatch(batch, raw)
		appendUnique(batch)
		report("feeds3")

		minTs := utils.ExtractMinAbstime(raw)
		for _, a := range batch {
			if a != nil && !a.Timestamp.IsZero() {
				ts := a.Timestamp.Unix()
				if minTs <= 0 || ts < minTs {
					minTs = ts
				}
			}
		}
		if minTs <= 0 || minTs >= begintime {
			break
		}
		begintime = minTs
		if err := sleepCtx(ctx, 100*time.Millisecond); err != nil {
			return
		}
	}
}

func (c *qzoneAPIClient) fetchActivitiesFromFeeds3(
	ctx context.Context,
	cookies map[string]string, uin string,
	appendUnique func([]*entity.Activity) int, report func(string),
) {
	trackBatch := func(batch []*entity.Activity, raw string) {}
	c.fetchActivitiesFromFeeds3Starting(ctx, cookies, uin, 1514736000, appendUnique, trackBatch, report)
}

func (c *qzoneAPIClient) fetchLegacyFeeds3(ctx context.Context, cookies map[string]string, uin string, begintime int64, count int) ([]byte, error) {
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds3_html_more?uin=%s&scope=0&view=1&flag=1&filter=all&begintime=%d&count=%d&useutf8=1&outputhtmlfeed=1&format=jsonp&g_tk=%s",
		uin, begintime, count, gTk,
	)
	body, err := c.doGet(ctx, cookies, url, uin, true)
	if err == nil {
		return body, nil
	}
	legacy := fmt.Sprintf(
		"https://user.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds3_html_more?uin=%s&scope=0&view=1&flag=1&filter=all&begintime=%d&count=%d&useutf8=1&outputhtmlfeed=1&format=jsonp&g_tk=%s",
		uin, begintime, count, gTk,
	)
	return c.doGet(ctx, cookies, legacy, uin, true)
}

func (c *qzoneAPIClient) fetchActivitiesPage(cookies map[string]string, uin string, offset, count int) ([]*entity.Activity, bool, error) {
	body, err := c.fetchFeedBody(context.Background(), cookies, uin, offset, count)
	if err != nil {
		return nil, false, err
	}
	if strings.Contains(string(body), "need login") {
		return nil, false, fmt.Errorf("need login")
	}
	processedHTML := utils.ProcessFeedResponse(string(body))
	if !strings.Contains(processedHTML, "li") {
		return nil, false, nil
	}
	activities, err := c.parseActivitiesFromHTML(processedHTML, uin)
	if err != nil {
		return nil, false, err
	}
	return activities, utils.HasMoreFeeds(string(body)), nil
}

func activityDedupKey(a *entity.Activity) string {
	return fmt.Sprintf("%s|%s|%s|%d", a.SenderQQ, a.Content, a.TimeText, a.Type)
}

func (c *qzoneAPIClient) GetActivities(cookies map[string]string, offset, count int) ([]*entity.Activity, error) {
	uin := utils.ExtractUin(cookies)
	body, err := c.fetchFeedBody(context.Background(), cookies, uin, offset, count)
	if err != nil {
		return nil, err
	}
	processedHTML := utils.ProcessFeedResponse(string(body))
	if !strings.Contains(processedHTML, "li") {
		return nil, nil
	}
	return c.parseActivitiesFromHTML(processedHTML, uin)
}

func (c *qzoneAPIClient) parseActivitiesFromHTML(processedHTML, uin string) ([]*entity.Activity, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(processedHTML))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %w", err)
	}

	var activities []*entity.Activity
	doc.Find("li.f-single.f-s-s").Each(func(i int, s *goquery.Selection) {
		activity := &entity.Activity{}
		activity.ReceiverQQ = uin

		senderElement := s.Find("a.f-name.q_namecard")
		if senderElement.Length() > 0 {
			activity.SenderName = strings.TrimSpace(senderElement.First().Text())
			activity.SenderQQ = strings.TrimPrefix(senderElement.First().AttrOr("link", ""), "nameCard_")
			activity.SenderLink = senderElement.First().AttrOr("href", "")
		}

		timeElement := s.Find("div.info-detail")
		if timeElement.Length() > 0 {
			activity.TimeText = strings.TrimSpace(timeElement.First().Text())
			activity.Timestamp = parseTime(activity.TimeText)
		}

		contentElement := s.Find("p.txt-box-title.ellipsis-one")
		if contentElement.Length() > 0 {
			activity.Content = strings.TrimSpace(contentElement.First().Text())
			activity.Content = strings.ReplaceAll(activity.Content, "\u00a0", " ")
		}

		imgElements := s.Find("a.img-item img")
		imgElements.Each(func(i int, img *goquery.Selection) {
			if src, exists := img.Attr("src"); exists {
				activity.ImageURLs = append(activity.ImageURLs, src)
			}
		})

		stateElement := s.Find("span.state")
		stateText := strings.Join(stateElement.Map(func(_ int, sel *goquery.Selection) string {
			return strings.TrimSpace(sel.Text())
		}), " ")

		switch {
		case strings.Contains(stateText, "留言") && strings.Contains(stateText, "回复"):
			activity.Type = entity.TypeBoardReply
		case strings.Contains(stateText, "留言"):
			activity.Type = entity.TypeBoardMessage
		case strings.Contains(stateText, "赞了我的说说"):
			activity.Type = entity.TypeLike
		case strings.Contains(stateText, "查看了我的说说"):
			activity.Type = entity.TypeView
		case strings.Contains(stateText, "访问了我的主页"):
			activity.Type = entity.TypeView
		case strings.Contains(stateText, "评论"):
			activity.Type = entity.TypeComment
		case strings.Contains(stateText, "回复"):
			activity.Type = entity.TypeComment
		case strings.Contains(stateText, "发表了说说"), strings.Contains(stateText, "发表说说"):
			activity.Type = entity.TypeMoment
		case strings.Contains(stateText, "说说") && activity.SenderQQ == uin:
			activity.Type = entity.TypeMoment
		case s.Find("div.f-reprint").Length() > 0:
			activity.Type = entity.TypeForward
			forwardContent := s.Find("div.f-reprint div.f-info").Text()
			activity.Content = strings.TrimSpace(forwardContent)
		default:
			activity.Type = entity.TypeOther
		}

		activities = append(activities, activity)
	})

	return activities, nil
}

func parseTime(timeStr string) time.Time {
	return timeparse.ParseCN(timeStr, time.Now().Year())
}

func (c *qzoneAPIClient) GetVisibleMoments(cookies map[string]string) ([]entity.Moment, error) {
	cookies = c.warmUpSession(cookies)
	uin := utils.ExtractUin(cookies)
	gTk := utils.GenerateGTK(cookies["p_skey"])

	total, err := c.fetchVisibleMomentsPage(cookies, uin, gTk, 0, 1)
	if err != nil {
		return nil, err
	}
	if total.Total > 1 {
		all, err := c.fetchVisibleMomentsPage(cookies, uin, gTk, 0, total.Total)
		if err == nil && len(all.MsgList) > 0 {
			return c.momentsFromMsgList(all.MsgList, uin), nil
		}
	}

	pageSize := 30
	offset := 0
	var allMoments []entity.Moment

	for {
		url := fmt.Sprintf(
			"https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/emotion_cgi_msglist_v6?uin=%s&ftype=0&sort=0&pos=%d&num=%d&replynum=100&g_tk=%s&callback=_preloadCallback&code_version=1&format=jsonp&need_private_comment=1",
			uin, offset, pageSize, gTk,
		)
		body, err := c.doGet(context.Background(), cookies, url, uin, true)
		if err != nil {
			return nil, err
		}

		raw := strings.TrimSpace(string(body))
		raw = strings.TrimPrefix(raw, "_preloadCallback(")
		raw = strings.TrimSuffix(raw, ");")
		raw = strings.TrimSuffix(raw, ")")

		var result struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Total   int    `json:"total"`
			MsgList []struct {
				Content     string `json:"content"`
				CreatedTime int64  `json:"created_time"`
				TID         string `json:"tid"`
				Pic         []struct {
					URL1 string `json:"url1"`
				} `json:"pic"`
				CommentList []struct {
					Content     string      `json:"content"`
					CreateTime2 string      `json:"createTime2"`
					Uin         interface{} `json:"uin"`
				} `json:"commentlist"`
			} `json:"msglist"`
		}
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return allMoments, fmt.Errorf("解析说说列表失败: %w", err)
		}
		if result.Code != 0 {
			return allMoments, fmt.Errorf("获取说说失败: %s", result.Message)
		}
		if len(result.MsgList) == 0 {
			break
		}

		for _, item := range result.MsgList {
			moment := entity.Moment{
				ID:              item.TID,
				UserQQ:          uin,
				SenderQQ:        uin,
				Content:         item.Content,
				Timestamp:       time.Unix(item.CreatedTime, 0),
				IsReconstructed: false,
			}
			for _, pic := range item.Pic {
				moment.ImageURLs = append(moment.ImageURLs, pic.URL1)
			}
			for _, cmt := range item.CommentList {
				moment.Comments = append(moment.Comments, entity.Comment{
					UserQQ:   fmt.Sprintf("%v", cmt.Uin),
					Content:  cmt.Content,
					TimeText: cmt.CreateTime2,
				})
			}
			if moment.ID == "" {
				_ = moment.BeforeCreate(nil)
			}
			allMoments = append(allMoments, moment)
		}

		offset += len(result.MsgList)
		if offset >= result.Total {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	return allMoments, nil
}

type visibleMsgItem struct {
	Content     string `json:"content"`
	CreatedTime int64  `json:"created_time"`
	TID         string `json:"tid"`
	Pic         []struct {
		URL1 string `json:"url1"`
	} `json:"pic"`
	CommentList []struct {
		Content     string      `json:"content"`
		CreateTime2 string      `json:"createTime2"`
		Uin         interface{} `json:"uin"`
	} `json:"commentlist"`
}

type visibleMsgResult struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Total   int              `json:"total"`
	MsgList []visibleMsgItem `json:"msglist"`
}

func (c *qzoneAPIClient) fetchVisibleMomentsPage(cookies map[string]string, uin, gTk string, pos, num int) (*visibleMsgResult, error) {
	url := fmt.Sprintf(
		"https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/emotion_cgi_msglist_v6?uin=%s&ftype=0&sort=0&pos=%d&num=%d&replynum=100&g_tk=%s&callback=_preloadCallback&code_version=1&format=jsonp&need_private_comment=1",
		uin, pos, num, gTk,
	)
	body, err := c.doGet(context.Background(), cookies, url, uin, true)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(string(body))
	raw = strings.TrimPrefix(raw, "_preloadCallback(")
	raw = strings.TrimSuffix(raw, ");")
	raw = strings.TrimSuffix(raw, ")")
	var result visibleMsgResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("解析说说列表失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("获取说说失败: %s", result.Message)
	}
	return &result, nil
}

func (c *qzoneAPIClient) momentsFromMsgList(items []visibleMsgItem, uin string) []entity.Moment {
	var allMoments []entity.Moment
	for _, item := range items {
		moment := entity.Moment{
			ID:              item.TID,
			UserQQ:          uin,
			SenderQQ:        uin,
			Content:         item.Content,
			Timestamp:       time.Unix(item.CreatedTime, 0),
			IsReconstructed: false,
		}
		for _, pic := range item.Pic {
			moment.ImageURLs = append(moment.ImageURLs, pic.URL1)
		}
		for _, cmt := range item.CommentList {
			moment.Comments = append(moment.Comments, entity.Comment{
				UserQQ:   fmt.Sprintf("%v", cmt.Uin),
				Content:  cmt.Content,
				TimeText: cmt.CreateTime2,
			})
		}
		if moment.ID == "" {
			_ = moment.BeforeCreate(nil)
		}
		allMoments = append(allMoments, moment)
	}
	return allMoments
}

func (c *qzoneAPIClient) GetBoardMessages(cookies map[string]string) ([]entity.BoardMessage, error) {
	cookies = c.warmUpSession(cookies)
	uin := utils.ExtractUin(cookies)
	gTk := utils.GenerateGTK(cookies["p_skey"])

	start := 0
	pageSize := 50
	var allMessages []entity.BoardMessage

	for {
		url := fmt.Sprintf(
			"https://user.qzone.qq.com/proxy/domain/m.qzone.qq.com/cgi-bin/new/get_msgb?uin=%s&hostUin=%s&num=%d&start=%d&hostword=0&essence=1&iNotice=0&inCharset=utf-8&outCharset=utf-8&format=jsonp&ref=qzone&g_tk=%s",
			uin, uin, pageSize, start, gTk,
		)
		body, err := c.doGet(context.Background(), cookies, url, uin, true)
		if err != nil {
			return allMessages, err
		}

		raw := strings.TrimSpace(string(body))
		raw = strings.TrimPrefix(raw, "_Callback(")
		raw = strings.TrimSuffix(raw, ");")
		raw = strings.TrimSuffix(raw, ")")

		var result struct {
			Code int `json:"code"`
			Data struct {
				Total       int `json:"total"`
				CommentList []struct {
					ID          string      `json:"id"`
					Uin         interface{} `json:"uin"`
					HTMLContent string      `json:"htmlContent"`
					UbbContent  string      `json:"ubbContent"`
					Pubtime     string      `json:"pubtime"`
					Nickname    string      `json:"nickname"`
				} `json:"commentList"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return allMessages, fmt.Errorf("解析留言板响应失败: %w", err)
		}
		if result.Code != 0 {
			return allMessages, fmt.Errorf("获取留言板失败: code=%d", result.Code)
		}
		if len(result.Data.CommentList) == 0 {
			break
		}

		for _, item := range result.Data.CommentList {
			content := item.UbbContent
			if content == "" {
				content = stripHTML(item.HTMLContent)
			}
			msg := entity.BoardMessage{
				ID:        item.ID,
				UserQQ:    uin,
				SenderQQ:  formatQQ(item.Uin),
				Content:   content,
				TimeText:  item.Pubtime,
				Timestamp: parseBoardTime(item.Pubtime),
			}
			if msg.ID == "" {
				_ = msg.BeforeCreate(nil)
			}
			allMessages = append(allMessages, msg)
		}

		start += len(result.Data.CommentList)
		if start >= result.Data.Total {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	return allMessages, nil
}

func parseBoardTime(timeStr string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, timeStr, time.Local); err == nil {
			return t
		}
	}
	return parseTime(timeStr)
}

func stripHTML(input string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(input, "")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	return strings.TrimSpace(text)
}

func formatQQ(value interface{}) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

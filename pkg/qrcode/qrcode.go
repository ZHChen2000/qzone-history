package qrcode

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mdp/qrterminal/v3"
)

//go:embed templates/login.html
var loginTemplate string

// SaveQRCode 用于将二维码保存为图片文件
func SaveQRCode(qrData []byte) (string, error) {
	// 获取当前工作目录
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 创建文件名
	filename := filepath.Join(dir, "qrcode.png")

	// 写入文件
	err = os.WriteFile(filename, qrData, 0644)
	if err != nil {
		return "", err
	}

	return filename, nil
}

// Display 用于在终端上显示二维码
func Display(qrData []byte) {
	qrterminal.GenerateHalfBlock(string(qrData), qrterminal.M, os.Stdout)
	fmt.Println("请使用手机QQ扫描上方二维码登录")
}

// SaveAndDisplayQRCode 整合保存和显示二维码的功能
func SaveAndDisplayQRCode(qrData []byte) error {
	// 保存二维码
	qrPath, err := SaveQRCode(qrData)
	if err != nil {
		return fmt.Errorf("保存二维码失败: %w", err)
	}

	// 打印二维码路径
	fmt.Printf("二维码已保存至 %s，请使用手机QQ扫描登录\n", qrPath)

	// 在终端显示二维码
	Display(qrData)

	return nil
}

// OpenInBrowser 在浏览器中打开二维码，返回一个关闭服务器的函数和通知登录成功的函数
func OpenInBrowser(qrData []byte) (shutdown func(), notifySuccess func(), err error) {
	// 找一个可用端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("无法监听端口: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	// 登录状态
	loginSuccess := false

	// 解析模板
	tmpl, err := template.New("login").Parse(loginTemplate)
	if err != nil {
		return nil, nil, fmt.Errorf("解析模板失败: %w", err)
	}

	// 渲染 HTML
	base64Data := base64.StdEncoding.EncodeToString(qrData)
	var htmlBuf bytes.Buffer
	err = tmpl.Execute(&htmlBuf, map[string]string{
		"QRCodeBase64": base64Data,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("渲染模板失败: %w", err)
	}
	html := htmlBuf.String()

	// 创建 HTTP 服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if loginSuccess {
			w.Write([]byte(`{"success": true}`))
		} else {
			w.Write([]byte(`{"success": false}`))
		}
	})

	server := &http.Server{Handler: mux}

	// 在后台启动服务器
	go func() {
		server.Serve(listener)
	}()

	// 打开浏览器
	if err := openBrowser(url); err != nil {
		fmt.Printf("无法自动打开浏览器，请手动访问: %s\n", url)
	} else {
		fmt.Printf("已在浏览器中打开二维码页面: %s\n", url)
	}

	// 返回关闭函数和通知成功函数
	shutdown = func() {
		server.Shutdown(context.Background())
	}
	notifySuccess = func() {
		loginSuccess = true
	}

	return shutdown, notifySuccess, nil
}

// openBrowser 在默认浏览器中打开 URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	return cmd.Start()
}

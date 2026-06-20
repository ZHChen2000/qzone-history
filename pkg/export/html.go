package export

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"qzone-history/internal/domain/entity"
	"time"
)

//go:embed templates/viewer.html
var viewerTemplateFS embed.FS

type ViewerPayload struct {
	UserQQ        string                `json:"userQQ"`
	GeneratedAt   string                `json:"generatedAt"`
	Moments       []entity.Moment       `json:"moments"`
	BoardMessages []entity.BoardMessage `json:"boardMessages"`
	Activities    []entity.Activity     `json:"activities"`
}

func WriteViewerHTML(filename string, payload ViewerPayload) error {
	if payload.GeneratedAt == "" {
		payload.GeneratedAt = time.Now().Format("2006-01-02 15:04:05")
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化浏览数据失败: %w", err)
	}

	tmpl, err := template.ParseFS(viewerTemplateFS, "templates/viewer.html")
	if err != nil {
		return fmt.Errorf("解析 HTML 模板失败: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]string{
		"UserQQ":      payload.UserQQ,
		"GeneratedAt": payload.GeneratedAt,
		"PayloadB64":  base64.StdEncoding.EncodeToString(raw),
	}); err != nil {
		return fmt.Errorf("渲染 HTML 失败: %w", err)
	}

	return os.WriteFile(filename, buf.Bytes(), 0644)
}

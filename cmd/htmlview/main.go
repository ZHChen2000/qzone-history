package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"qzone-history/internal/domain/entity"
	"qzone-history/pkg/export"
	"time"
)

func main() {
	qq := flag.String("qq", "", "QQ 号码（必填）")
	flag.Parse()
	if *qq == "" {
		log.Fatal("请指定 -qq 参数")
	}

	exportFile := fmt.Sprintf("%s_export.json", *qq)
	activitiesFile := fmt.Sprintf("%s_activities.json", *qq)
	outFile := fmt.Sprintf("%s_view.html", *qq)

	exportRaw, err := os.ReadFile(exportFile)
	if err != nil {
		log.Fatalf("读取 %s 失败: %v", exportFile, err)
	}

	var exportData struct {
		UserQQ        string                `json:"userQQ"`
		Moments       []entity.Moment       `json:"moments"`
		BoardMessages []entity.BoardMessage `json:"boardMessages"`
	}
	if err := json.Unmarshal(exportRaw, &exportData); err != nil {
		log.Fatalf("解析 export JSON 失败: %v", err)
	}

	var activities []entity.Activity
	if activitiesRaw, err := os.ReadFile(activitiesFile); err == nil {
		if err := json.Unmarshal(activitiesRaw, &activities); err != nil {
			log.Printf("解析 activities JSON 失败（将跳过活动记录）: %v", err)
		}
	}

	payload := export.ViewerPayload{
		UserQQ:        exportData.UserQQ,
		GeneratedAt:   time.Now().Format("2006-01-02 15:04:05"),
		Moments:       exportData.Moments,
		BoardMessages: exportData.BoardMessages,
		Activities:    activities,
	}
	if payload.UserQQ == "" {
		payload.UserQQ = *qq
	}

	if err := export.WriteViewerHTML(outFile, payload); err != nil {
		log.Fatalf("生成 HTML 失败: %v", err)
	}
	log.Printf("已生成浏览页: %s", outFile)
}

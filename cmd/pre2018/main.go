package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/internal/infrastructure/persistence"
	"qzone-history/internal/infrastructure/qzone_api"
	"qzone-history/pkg/database"
	"qzone-history/pkg/database/sqlite"
	"qzone-history/pkg/utils"
	"strings"
	"time"
)

func main() {
	cfg, _ := config.LoadConfig()
	db := sqlite.NewSQLiteDB()
	db.Connect(&database.Config{DBName: cfg.Database.DBName})
	defer db.Close()

	user, err := persistence.NewUserRepository(db).GetLastLoginUser(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	uin := user.QQ
	cookies := user.Cookies

	fmt.Printf("=== 2018年前数据全力探测 QQ %s ===\n\n", uin)

	fmt.Println("--- feeds2 超大 offset ---")
	bestMin := int64(0)
	for _, off := range []int{3000, 3200, 3400, 3600, 3800, 4000, 4500, 5000, 6000, 7000, 8000, 10000, 12000, 15000} {
		body, err := qzone_api.ProbeFeedOffset(cookies, uin, off, 100)
		if err != nil {
			fmt.Printf("offset=%-5d ERR %v\n", off, err)
			continue
		}
		raw := string(body)
		minTs, n := qzone_api.ExtractMinAbstime(raw)
		if n == 0 {
			fmt.Printf("offset=%-5d 空\n", off)
			continue
		}
		fmt.Printf("offset=%-5d count~%-3d min=%s\n", off, n, qzone_api.FormatUnix(minTs))
		if bestMin == 0 || (minTs > 0 && minTs < bestMin) {
			bestMin = minTs
		}
		time.Sleep(150 * time.Millisecond)
	}

	fmt.Println("\n--- feeds2 细粒度 offset 2500-4500 ---")
	for off := 2500; off <= 4500; off += 50 {
		body, err := qzone_api.ProbeFeedOffset(cookies, uin, off, 100)
		if err != nil {
			continue
		}
		minTs, n := qzone_api.ExtractMinAbstime(string(body))
		if n == 0 {
			continue
		}
		if minTs > 0 && minTs < 1514764800 { // before 2018-01-01
			fmt.Printf("*** offset=%d min=%s (2018前!) ***\n", off, qzone_api.FormatUnix(minTs))
			bestMin = minTs
		} else if minTs > 0 && minTs < 1546300800 { // before 2019
			fmt.Printf("offset=%d min=%s\n", off, qzone_api.FormatUnix(minTs))
			if bestMin == 0 || minTs < bestMin {
				bestMin = minTs
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n--- feeds3 basetime 历史 ---")
	timestamps := []int64{
		0,
		1483228800, // 2017-01-01
		1451606400, // 2016-01-01
		1420041600, // 2015-01-01
		1388534400, // 2014-01-01
		1356998400, // 2013-01-01
	}
	for _, bt := range timestamps {
		body, err := qzone_api.ProbeLegacyFeeds3(cookies, uin, bt, 50)
		if err != nil {
			fmt.Printf("feeds3 bt=%s ERR %v\n", qzone_api.FormatUnix(bt), err)
			continue
		}
		raw := string(body)
		processed := utils.ProcessFeedResponse(raw)
		acts, _ := qzone_api.ParseActivitiesFromHTML(processed, uin)
		minTs, _ := qzone_api.ExtractMinAbstime(raw)
		fmt.Printf("feeds3 bt=%-10s html_li=%v acts=%-3d min=%s\n",
			qzone_api.FormatUnix(bt), strings.Contains(processed, "li"), len(acts), qzone_api.FormatUnix(minTs))
		if minTs > 0 && (bestMin == 0 || minTs < bestMin) {
			bestMin = minTs
		}
		time.Sleep(150 * time.Millisecond)
	}

	fmt.Println("\n--- emotion_cgi_msglist 变体 ---")
	for _, tc := range []struct {
		sort, ftype, pos, num int
		label                string
	}{
		{0, 0, 0, 100, "sort0 ftype0 pos0"},
		{1, 0, 0, 100, "sort1(正序?) pos0"},
		{0, 0, 100, 100, "pos100"},
		{0, 0, 500, 100, "pos500"},
		{0, 1, 0, 100, "ftype1"},
	} {
		body, err := qzone_api.ProbeEmotionMsgList(cookies, uin, tc.sort, tc.ftype, tc.pos, tc.num)
		if err != nil {
			fmt.Printf("%s ERR %v\n", tc.label, err)
			continue
		}
		raw := strings.TrimSpace(string(body))
		raw = strings.TrimPrefix(raw, "_preloadCallback(")
		raw = strings.TrimSuffix(raw, ");")
		var result struct {
			Code  int `json:"code"`
			Total int `json:"total"`
			MsgList []struct {
				Content     string `json:"content"`
				CreatedTime int64  `json:"created_time"`
			} `json:"msglist"`
		}
		if json.Unmarshal([]byte(raw), &result) != nil {
			fmt.Printf("%s 解析失败\n", tc.label)
			continue
		}
		var minT int64
		for _, m := range result.MsgList {
			if minT == 0 || m.CreatedTime < minT {
				minT = m.CreatedTime
			}
		}
		fmt.Printf("%-20s total=%-4d batch=%-3d min=%s\n", tc.label, result.Total, len(result.MsgList), qzone_api.FormatUnix(minT))
		if minT > 0 && minT < 1514764800 {
			fmt.Printf("  *** 发现2018前说说! ***\n")
		}
		if minT > 0 && (bestMin == 0 || minT < bestMin) {
			bestMin = minT
		}
		time.Sleep(150 * time.Millisecond)
	}

	fmt.Printf("\n=== 全局最早时间戳: %s ===\n", qzone_api.FormatUnix(bestMin))
	if bestMin >= 1514764800 || bestMin == 0 {
		fmt.Println("结论: 本次探测未发现 2018-01-01 之前的可访问数据")
	}
}

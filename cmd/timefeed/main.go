package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/internal/infrastructure/persistence"
	"qzone-history/internal/infrastructure/qzone_api"
	"qzone-history/pkg/database"
	"qzone-history/pkg/database/sqlite"
	"qzone-history/pkg/utils"
	"regexp"
	"sort"
	"strings"
	"time"
)

type timeRange struct {
	label     string
	beginTime int64
	endTime   int64
}

func main() {
	deep := flag.Bool("deep", false, "深分页探测")
	fetch := flag.Bool("fetch", false, "测试稀疏深分页拉取并统计")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	db := sqlite.NewSQLiteDB()
	if err := db.Connect(&database.Config{DBName: cfg.Database.DBName}); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	userRepo := persistence.NewUserRepository(db)
	user, err := userRepo.GetLastLoginUser(context.Background())
	if err != nil {
		log.Fatalf("无有效登录，请先运行 qzone-history.exe 扫码登录: %v", err)
	}
	if user.LoginExpireTime.Before(time.Now()) {
		log.Fatalf("登录已过期 (%s)，请重新扫码", user.LoginExpireTime.Format(time.RFC3339))
	}

	client := qzone_api.NewQzoneAPIClient(cfg)
	uin := user.QQ

	if *fetch {
		runFetchTest(client, user.Cookies, uin)
		return
	}

	if *deep {
		runDeepProbe(user.Cookies, uin)
		return
	}

	ranges := []timeRange{
		{"默认(0,0)", 0, 0},
		{"2015-2016", ts(2015, 1, 1), ts(2016, 12, 31, 23, 59, 59)},
		{"2018-2019", ts(2018, 1, 1), ts(2019, 12, 31, 23, 59, 59)},
		{"2019-2020", ts(2019, 1, 1), ts(2020, 12, 31, 23, 59, 59)},
	}

	extraProbes := []struct {
		label string
		fn    func() (int, []string)
	}{
		{"feeds3 begintime=0", func() (int, []string) { return probeFeeds3(client, user.Cookies, uin, 0) }},
		{"feeds3 begintime=1577836800(2020)", func() (int, []string) { return probeFeeds3(client, user.Cookies, uin, 1577836800) }},
		{"feeds3 begintime=1546300800(2019)", func() (int, []string) { return probeFeeds3(client, user.Cookies, uin, 1546300800) }},
		{"feeds3 begintime=1514736000(2018)", func() (int, []string) { return probeFeeds3(client, user.Cookies, uin, 1514736000) }},
		{"set=1 offset0", func() (int, []string) { return probeSetParam(client, user.Cookies, uin, 1) }},
		{"set=2 offset0", func() (int, []string) { return probeSetParam(client, user.Cookies, uin, 2) }},
		{"offset=600", func() (int, []string) { return probeOffset(client, user.Cookies, uin, 600) }},
		{"offset=1000", func() (int, []string) { return probeOffset(client, user.Cookies, uin, 1000) }},
	}

	fmt.Printf("账号: %s (%s)\n\n", user.QQ, user.Nickname)
	fmt.Printf("%-22s %8s %8s %12s %s\n", "时间段", "条数", "total", "hasMore", "时间样例")
	fmt.Println(strings.Repeat("-", 90))

	for _, r := range ranges {
		total, count, hasMore, samples := probeRange(client, user.Cookies, uin, r)
		sampleStr := strings.Join(samples, " | ")
		if sampleStr == "" {
			sampleStr = "(无)"
		}
		fmt.Printf("%-22s %8d %8d %12v %s\n", r.label, count, total, hasMore, sampleStr)
	}

	fmt.Println()
	fmt.Println("=== 其他探测 ===")
	for _, p := range extraProbes {
		count, samples := p.fn()
		sampleStr := strings.Join(samples, " | ")
		if sampleStr == "" {
			sampleStr = "(无)"
		}
		fmt.Printf("%-30s %8d  %s\n", p.label, count, sampleStr)
	}
}

func ts(y int, m int, d int, hms ...int) int64 {
	h, mi, s := 0, 0, 0
	if len(hms) > 0 {
		h = hms[0]
	}
	if len(hms) > 1 {
		mi = hms[1]
	}
	if len(hms) > 2 {
		s = hms[2]
	}
	return time.Date(y, time.Month(m), d, h, mi, s, 0, time.Local).Unix()
}

func probeRange(client qzone_api.QzoneAPIClient, cookies map[string]string, uin string, r timeRange) (total, count int, hasMore bool, samples []string) {
	body, err := qzone_api.ProbeFeedRange(cookies, uin, r.beginTime, r.endTime, 0, 100)
	if err != nil {
		samples = []string{fmt.Sprintf("ERR: %v", err)}
		return
	}

	raw := string(body)
	total = utils.ExtractFeedTotalNumber(raw)
	hasMore = utils.HasMoreFeeds(raw)
	processed := utils.ProcessFeedResponse(raw)
	if !strings.Contains(processed, "li") {
		return total, 0, hasMore, nil
	}

	activities, err := qzone_api.ParseActivitiesFromHTML(processed, uin)
	if err != nil {
		samples = []string{fmt.Sprintf("parse ERR: %v", err)}
		return
	}
	count = len(activities)
	for i, a := range activities {
		if i >= 3 {
			break
		}
		tt := strings.TrimSpace(strings.ReplaceAll(a.TimeText, "\t", ""))
		if tt == "" {
			tt = "(无时间)"
		}
		samples = append(samples, fmt.Sprintf("%s: %s", a.SenderName, tt))
	}

	if hasMore && count > 0 {
		body2, err := qzone_api.ProbeFeedRange(cookies, uin, r.beginTime, r.endTime, count, 100)
		if err == nil {
			p2 := utils.ProcessFeedResponse(string(body2))
			if a2, err := qzone_api.ParseActivitiesFromHTML(p2, uin); err == nil {
				count += len(a2)
			}
		}
	}

	abstimeRe := regexp.MustCompile(`abstime:'(\d+)'`)
	matches := abstimeRe.FindAllStringSubmatch(raw, 3)
	for _, m := range matches {
		if len(m) == 2 {
			var ts int64
			fmt.Sscanf(m[1], "%d", &ts)
			samples = append(samples, fmt.Sprintf("abstime=%s", time.Unix(ts, 0).Format("2006-01-02")))
		}
	}
	return
}

func probeFeeds3(client qzone_api.QzoneAPIClient, cookies map[string]string, uin string, begintime int64) (int, []string) {
	body, err := qzone_api.ProbeFeeds3(cookies, uin, begintime)
	if err != nil {
		return 0, []string{fmt.Sprintf("ERR: %v", err)}
	}
	return summarizeBody(body, uin)
}

func probeSetParam(client qzone_api.QzoneAPIClient, cookies map[string]string, uin string, set int) (int, []string) {
	body, err := qzone_api.ProbeFeedSet(cookies, uin, set, 0, 100)
	if err != nil {
		return 0, []string{fmt.Sprintf("ERR: %v", err)}
	}
	return summarizeBody(body, uin)
}

func probeOffset(client qzone_api.QzoneAPIClient, cookies map[string]string, uin string, offset int) (int, []string) {
	body, err := qzone_api.ProbeFeedRange(cookies, uin, 0, 0, offset, 100)
	if err != nil {
		return 0, []string{fmt.Sprintf("ERR: %v", err)}
	}
	return summarizeBody(body, uin)
}

func summarizeBody(body []byte, uin string) (int, []string) {
	raw := string(body)
	if strings.Contains(raw, "waf.tencent.com") {
		return 0, []string{"WAF blocked"}
	}
	processed := utils.ProcessFeedResponse(raw)
	if !strings.Contains(processed, "li") {
		if len(raw) < 300 {
			return 0, []string{strings.TrimSpace(raw)}
		}
		return 0, []string{"(无 li)"}
	}
	activities, err := qzone_api.ParseActivitiesFromHTML(processed, uin)
	if err != nil {
		return 0, []string{fmt.Sprintf("parse: %v", err)}
	}
	var samples []string
	for i, a := range activities {
		if i >= 2 {
			break
		}
		samples = append(samples, fmt.Sprintf("%s %s", a.SenderName, strings.TrimSpace(strings.ReplaceAll(a.TimeText, "\t", ""))))
	}
	abstimeRe := regexp.MustCompile(`abstime:'(\d+)'`)
	for _, m := range abstimeRe.FindAllStringSubmatch(raw, 3) {
		if len(m) == 2 {
			var ts int64
			fmt.Sscanf(m[1], "%d", &ts)
			samples = append(samples, time.Unix(ts, 0).Format("2006-01-02"))
		}
	}
	return len(activities), samples
}

func runFetchTest(client qzone_api.QzoneAPIClient, cookies map[string]string, uin string) {
	fmt.Println("=== 稀疏深分页拉取测试 ===")
	activities, err := client.GetAllActivities(cookies, qzone_api.DefaultFetchOptions())
	if err != nil {
		log.Fatal(err)
	}
	yearCount := map[string]int{}
	abstimeRe := regexp.MustCompile(`(\d{4})年`)
	var minYear string
	for _, a := range activities {
		tt := strings.ReplaceAll(a.TimeText, "\t", "")
		y := ""
		if m := abstimeRe.FindStringSubmatch(tt); len(m) == 2 {
			y = m[1]
		} else if !a.Timestamp.IsZero() && a.Timestamp.Year() > 1 {
			y = fmt.Sprintf("%d", a.Timestamp.Year())
		}
		if y == "" {
			y = "unknown"
		}
		yearCount[y]++
		if minYear == "" || y < minYear && y != "unknown" {
			minYear = y
		}
	}
	fmt.Printf("合计 %d 条活动\n", len(activities))
	fmt.Printf("年份分布: %s\n", sortedYears(yearCount))
	fmt.Printf("最早年份: %s\n", minYear)
}

func runDeepProbe(cookies map[string]string, uin string) {
	fmt.Println("=== feeds2 offset 深分页探测 ===")
	offsets := []int{0, 500, 690, 800, 1000, 1200, 1500, 1800, 2000, 2200, 2400, 2600, 3000}
	for _, off := range offsets {
		body, err := qzone_api.ProbeFeedRange(cookies, uin, 0, 0, off, 100)
		if err != nil {
			fmt.Printf("offset=%-5d ERR: %v\n", off, err)
			continue
		}
		count, earliest, latest := summarizeTimes(body, uin)
		hasMore := utils.HasMoreFeeds(string(body))
		fmt.Printf("offset=%-5d count=%-3d hasMore=%-5v earliest=%s latest=%s\n", off, count, hasMore, earliest, latest)
	}

	fmt.Println("\n=== feeds2 连续深分页 ===")
	offset := 0
	pageSize := 100
	yearCount := map[string]int{}
	var minTs int64
	total := 0
	abstimeRe := regexp.MustCompile(`abstime:'(\d+)'`)

	for page := 0; page < 50; page++ {
		body, err := qzone_api.ProbeFeedRange(cookies, uin, 0, 0, offset, pageSize)
		if err != nil {
			fmt.Printf("page %d ERR: %v\n", page, err)
			break
		}
		raw := string(body)
		processed := utils.ProcessFeedResponse(raw)
		if !strings.Contains(processed, "li") {
			fmt.Printf("page %d offset %d: 无数据，停止\n", page, offset)
			break
		}
		activities, _ := qzone_api.ParseActivitiesFromHTML(processed, uin)
		if len(activities) == 0 {
			break
		}
		total += len(activities)
		for _, m := range abstimeRe.FindAllStringSubmatch(raw, -1) {
			var ts int64
			fmt.Sscanf(m[1], "%d", &ts)
			yearCount[time.Unix(ts, 0).Format("2006")]++
			if minTs == 0 || ts < minTs {
				minTs = ts
			}
		}
		hasMore := utils.HasMoreFeeds(raw)
		fmt.Printf("page %2d offset=%-5d batch=%-3d hasMore=%-5v min=%s\n",
			page, offset, len(activities), hasMore, time.Unix(minTs, 0).Format("2006-01-02"))
		offset += len(activities)
		if !hasMore && page >= 8 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Printf("\n合计 %d 条, 年份: %s, 最早: %s\n", total, sortedYears(yearCount),
		time.Unix(minTs, 0).Format("2006-01-02"))

	fmt.Println("\n=== feeds3 游标向历史翻页 ===")
	begintime := int64(0)
	for i := 0; i < 20; i++ {
		body, err := qzone_api.ProbeFeeds3(cookies, uin, begintime)
		if err != nil {
			fmt.Printf("feeds3 %d ERR: %v\n", i, err)
			break
		}
		matches := abstimeRe.FindAllStringSubmatch(string(body), -1)
		if len(matches) == 0 {
			break
		}
		var minInPage int64
		for _, m := range matches {
			var ts int64
			fmt.Sscanf(m[1], "%d", &ts)
			if minInPage == 0 || ts < minInPage {
				minInPage = ts
			}
		}
		count, earliest, _ := summarizeTimes(body, uin)
		fmt.Printf("feeds3 %2d begintime=%d count=%-3d min=%s earliest=%s\n",
			i, begintime, count, time.Unix(minInPage, 0).Format("2006-01-02"), earliest)
		if begintime > 0 && minInPage >= begintime {
			break
		}
		begintime = minInPage
		time.Sleep(200 * time.Millisecond)
	}
}

func summarizeTimes(body []byte, uin string) (count int, earliest, latest string) {
	raw := string(body)
	processed := utils.ProcessFeedResponse(raw)
	if !strings.Contains(processed, "li") {
		return 0, "(空)", "(空)"
	}
	activities, _ := qzone_api.ParseActivitiesFromHTML(processed, uin)
	count = len(activities)
	abstimeRe := regexp.MustCompile(`abstime:'(\d+)'`)
	var minTs, maxTs int64
	for _, m := range abstimeRe.FindAllStringSubmatch(raw, -1) {
		var ts int64
		fmt.Sscanf(m[1], "%d", &ts)
		if minTs == 0 || ts < minTs {
			minTs = ts
		}
		if ts > maxTs {
			maxTs = ts
		}
	}
	if minTs > 0 {
		return count, time.Unix(minTs, 0).Format("2006-01-02"), time.Unix(maxTs, 0).Format("2006-01-02")
	}
	return count, "(无)", "(无)"
}

func sortedYears(m map[string]int) string {
	years := make([]string, 0, len(m))
	for y := range m {
		years = append(years, y)
	}
	sort.Strings(years)
	parts := make([]string, 0, len(years))
	for _, y := range years {
		parts = append(parts, fmt.Sprintf("%s:%d", y, m[y]))
	}
	return strings.Join(parts, ", ")
}

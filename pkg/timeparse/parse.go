package timeparse

import (
	"strings"
	"time"
)

// ParseCN 解析 QQ 空间常见中文时间；仅月日时 defaultYear 补全年份。
func ParseCN(timeStr string, defaultYear int) time.Time {
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		return time.Time{}
	}
	if defaultYear <= 0 {
		defaultYear = time.Now().Year()
	}
	now := time.Now()

	layouts := []struct {
		layout string
		kind   int // 0=full, 1=month-day, 2=yesterday, 3=time-only
	}{
		{"2006年1月2日 15:04:05", 0},
		{"2006年01月02日 15:04:05", 0},
		{"2006年1月2日 15:04", 0},
		{"2006年01月02日 15:04", 0},
		{"2006-01-02 15:04:05", 0},
		{"2006-01-02 15:04", 0},
		{"1月2日 15:04", 1},
		{"01月02日 15:04", 1},
		{"昨天 15:04", 2},
		{"15:04", 3},
	}

	for _, item := range layouts {
		t, err := time.ParseInLocation(item.layout, timeStr, time.Local)
		if err != nil {
			continue
		}
		switch item.kind {
		case 0:
			return t
		case 1:
			return time.Date(defaultYear, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
		case 2:
			yesterday := now.AddDate(0, 0, -1)
			return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		case 3:
			return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
		}
	}
	return time.Time{}
}

// RefYearFromEarliest 根据已知的最早时间戳与目标年份推断缺省年份。
func RefYearFromEarliest(earliestUnix int64, targetYear int) int {
	if earliestUnix > 0 {
		return time.Unix(earliestUnix, 0).Year()
	}
	if targetYear > 0 {
		return targetYear
	}
	return time.Now().Year()
}

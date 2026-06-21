package timeparse

import (
	"strings"
	"time"
)

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

func RefYearFromEarliest(earliestUnix int64, targetYear int) int {
	if earliestUnix > 0 {
		return time.Unix(earliestUnix, 0).Year()
	}
	if targetYear > 0 {
		return targetYear
	}
	return time.Now().Year()
}

package timeparse

import (
	"regexp"
	"strings"
	"time"
)

func ParseCN(timeStr string, defaultYear int) time.Time {
	timeStr = normalize(timeStr)
	if absolute := ParseAbsolute(timeStr); !absolute.IsZero() {
		return absolute
	}
	if timeStr == "" {
		return time.Time{}
	}
	if defaultYear <= 0 {
		defaultYear = time.Now().Year()
	}
	now := time.Now()

	layouts := []struct {
		layout string
		kind   int
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

// QQ dates are expressed in Beijing time, independently of the machine zone.
var qzoneLocation = time.FixedZone("Asia/Shanghai", 8*60*60)
var absoluteDate = regexp.MustCompile("[0-9]{4}(?:年[0-9]{1,2}月[0-9]{1,2}日|-[0-9]{1,2}-[0-9]{1,2})(?:[ T][0-9]{1,2}:[0-9]{2}(?::[0-9]{2})?)?")

func normalize(text string) string {
	text = strings.NewReplacer("\\t", " ", "\\r", " ", "\\n", " ").Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

// ParseAbsolute parses only dates with an explicit year. Export sorting must
// not invent a year for an old month/day or relative activity record.
func ParseAbsolute(text string) time.Time {
	text = normalize(text)
	if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil && parsed.Year() > 1 {
		return parsed
	}
	text = absoluteDate.FindString(text)
	if text == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"2006年1月2日 15:04:05", "2006年1月2日 15:04", "2006年1月2日",
		"2006-1-2 15:04:05", "2006-1-2 15:04", "2006-1-2",
		"2006-1-2T15:04:05", "2006-1-2T15:04",
	} {
		if parsed, err := time.ParseInLocation(layout, text, qzoneLocation); err == nil && parsed.Year() > 1 {
			return parsed
		}
	}
	return time.Time{}
}

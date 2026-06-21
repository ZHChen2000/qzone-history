package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func extractStringBetween(sourceString, startString, endString string) string {
	startIndex := strings.Index(sourceString, startString) + len(startString)
	endIndex := strings.Index(sourceString, endString)
	if startIndex < 0 || endIndex < 0 || startIndex >= endIndex {
		return ""
	}
	return sourceString[startIndex:endIndex]
}

func replaceMultipleSpaces(s string) string {
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

func unescapeFeedText(message string) string {
	re := regexp.MustCompile(`\\x[0-9a-fA-F]{2}`)
	message = re.ReplaceAllStringFunc(message, func(hex string) string {
		byteValue, err := strconv.ParseUint(hex[2:], 16, 8)
		if err != nil {
			return hex
		}
		return string(rune(byteValue))
	})
	message = strings.ReplaceAll(message, "\\/", "/")
	message = strings.ReplaceAll(message, "\\'", "'")
	message = strings.ReplaceAll(message, "\\\"", "\"")
	return message
}

func ExtractH5FeedsHTML(message string) string {
	message = unescapeFeedText(message)
	var parts []string
	searchFrom := 0
	for {
		idx := strings.Index(message[searchFrom:], "html:'")
		if idx < 0 {
			break
		}
		start := searchFrom + idx + len("html:'")
		rest := message[start:]
		endMarkers := []string{"',is_public_pav", "',opuin"}
		end := -1
		for _, marker := range endMarkers {
			if pos := strings.Index(rest, marker); pos >= 0 && (end < 0 || pos < end) {
				end = pos
			}
		}
		if end < 0 {
			break
		}
		parts = append(parts, rest[:end])
		searchFrom = start + end + 1
	}
	return strings.Join(parts, "")
}

func ExtractFeedTotalNumber(message string) int {
	re := regexp.MustCompile(`total_number:(\d+)`)
	matches := re.FindStringSubmatch(message)
	if len(matches) == 2 {
		total, err := strconv.Atoi(matches[1])
		if err == nil {
			return total
		}
	}
	return -1
}

func HasMoreFeeds(message string) bool {
	return strings.Contains(message, "hasMoreFeeds:true")
}

func AbstimeRegex() *regexp.Regexp {
	return regexp.MustCompile(`abstime:'(\d+)'`)
}

func ExtractMinAbstime(message string) int64 {
	re := AbstimeRegex()
	matches := re.FindAllStringSubmatch(message, -1)
	var minTs int64
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		var ts int64
		fmt.Sscanf(m[1], "%d", &ts)
		if minTs == 0 || ts < minTs {
			minTs = ts
		}
	}
	return minTs
}

func ProcessFeedResponse(message string) string {
	if strings.Contains(message, "waf.tencent.com") {
		return ""
	}
	message = unescapeFeedText(message)
	if strings.Contains(message, "_Callback(") || strings.Contains(message, "data:[") {
		if extracted := ExtractH5FeedsHTML(message); extracted != "" {
			return replaceMultipleSpaces(extracted)
		}
	}
	return ProcessOldHTML(message)
}

func ProcessOldHTML(message string) string {
	newText := unescapeFeedText(message)

	patterns := []struct{ start, end string }{
		{"html:'", "',opuin"},
		{"html:\"", "\",opuin"},
		{"\"html\":\"", "\",\"opuin"},
	}
	for _, p := range patterns {
		if extracted := extractStringBetween(newText, p.start, p.end); extracted != "" {
			newText = extracted
			break
		}
	}

	newText = replaceMultipleSpaces(newText)
	newText = strings.ReplaceAll(newText, "\\", "")
	return newText
}

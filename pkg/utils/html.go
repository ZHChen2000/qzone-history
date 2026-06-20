package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// extractStringBetween 提取两个字符串之间的内容
func extractStringBetween(sourceString, startString, endString string) string {
	startIndex := strings.Index(sourceString, startString) + len(startString)
	endIndex := strings.Index(sourceString, endString)
	if startIndex < 0 || endIndex < 0 || startIndex >= endIndex {
		return ""
	}
	return sourceString[startIndex:endIndex]
}

// replaceMultipleSpaces 去除多余的空格
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

// ExtractH5FeedsHTML 从 h5 活动接口响应中提取所有 HTML 片段
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

// ExtractFeedTotalNumber 从 h5 响应中解析活动总数
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

// HasMoreFeeds 判断 h5 活动接口是否还有更多数据
func HasMoreFeeds(message string) bool {
	return strings.Contains(message, "hasMoreFeeds:true")
}

// AbstimeRegex 匹配 feed 中的 abstime 字段
func AbstimeRegex() *regexp.Regexp {
	return regexp.MustCompile(`abstime:'(\d+)'`)
}

// ExtractMinAbstime 从 feed 响应提取最早 abstime 时间戳
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

// ProcessFeedResponse 处理活动接口响应（兼容旧版与 h5 格式）
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

// ProcessOldHTML 替换十六进制编码并处理HTML
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

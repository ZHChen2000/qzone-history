package offset

// EstimateScan 根据目标年份与 max offset 估算全流程耗时（分钟）。
func EstimateScan(targetYear, maxOffset int) (minMinutes, maxMinutes int) {
	if targetYear < 2005 {
		targetYear = 2005
	}
	if maxOffset < 500 {
		maxOffset = 500
	}

	// 活动深扫请求量粗算（offset 深扫 + 时间窗 + feeds3）
	offsetSteps := maxOffset / 100
	if offsetSteps > 800 {
		offsetSteps = 800
	}
	ultraSteps := 0
	if maxOffset >= 5500 {
		ultraSteps = (minInt(15000, maxOffset) - 5500) / 20
	}
	yearSpan := 2026 - targetYear
	if yearSpan < 1 {
		yearSpan = 1
	}

	requests := 500 + offsetSteps*12 + ultraSteps*4 + yearSpan*40 + yearSpan*24
	// 每条请求约 0.25~0.45 分钟（含网络与限速），外加留言/重建/导出
	scanMin := requests * 15 / 100
	scanMax := requests * 27 / 100
	overhead := 15 + yearSpan*2

	minMinutes = scanMin + overhead
	maxMinutes = scanMax + overhead + 30
	if minMinutes < 20 {
		minMinutes = 20
	}
	if maxMinutes < minMinutes+30 {
		maxMinutes = minMinutes + 30
	}
	if maxMinutes > 480 {
		maxMinutes = 480
	}
	return minMinutes, maxMinutes
}

// EstimateScanText 返回给用户看的预计耗时文案。
func EstimateScanText(targetYear, maxOffset int) string {
	lo, hi := EstimateScan(targetYear, maxOffset)
	return "预计本次完整恢复约 " + itoa(lo) + "–" + itoa(hi) + " 分钟（目标 " + itoa(targetYear) +
		" 年及更早，max offset=" + itoa(maxOffset) + "）。深度扫描可能较长，请保持网络畅通。"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

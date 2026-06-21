package offset

func activityScanMinutes(targetYear int) (lo, hi int) {
	switch {
	case targetYear >= 2024:
		return 8, 15
	case targetYear == 2023:
		return 10, 18
	case targetYear == 2022:
		return 12, 22
	case targetYear == 2021:
		return 15, 25
	case targetYear == 2020:
		return 25, 40
	case targetYear == 2019:
		return 35, 55
	case targetYear == 2018:
		return 50, 75
	case targetYear == 2017:
		return 60, 90
	case targetYear == 2016:
		return 80, 110
	case targetYear == 2015:
		return 90, 120
	default:
		return 120, 180
	}
}

func EstimateScan(targetYear, maxOffset int) (minMinutes, maxMinutes int) {
	if targetYear < 2005 {
		targetYear = 2005
	}
	if maxOffset < 500 {
		maxOffset = 500
	}

	lo, hi := activityScanMinutes(targetYear)
	lo += 8
	hi += 15

	rec := RecommendMaxOffset(targetYear)
	if rec > 0 && maxOffset > rec {
		ratio := float64(maxOffset) / float64(rec)
		if ratio > 2.5 {
			ratio = 2.5
		}
		if ratio < 1 {
			ratio = 1
		}
		lo = int(float64(lo)*ratio + 0.5)
		hi = int(float64(hi)*ratio + 0.5)
	}

	if lo > hi {
		lo, hi = hi, lo
	}
	if lo < 15 {
		lo = 15
	}
	if hi < lo+10 {
		hi = lo + 10
	}
	if hi > 300 {
		hi = 300
	}
	if lo > hi-10 {
		lo = hi - 10
		if lo < 15 {
			lo = 15
		}
	}
	return lo, hi
}

func EstimateScanText(targetYear, maxOffset int) string {
	lo, hi := EstimateScan(targetYear, maxOffset)
	return "预计本次完整恢复约 " + itoa(lo) + "–" + itoa(hi) + " 分钟"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

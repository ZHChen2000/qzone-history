package offset

func RecommendMaxOffset(targetYear int) int {
	switch {
	case targetYear >= 2024:
		return 1500
	case targetYear == 2023:
		return 2500
	case targetYear == 2022:
		return 3500
	case targetYear == 2021:
		return 5000
	case targetYear == 2020:
		return 8000
	case targetYear == 2019:
		return 12000
	case targetYear == 2018:
		return 18000
	case targetYear == 2017:
		return 25000
	case targetYear == 2016:
		return 35000
	case targetYear == 2015:
		return 50000
	case targetYear == 2014:
		return 80000
	case targetYear == 2013:
		return 90000
	case targetYear == 2012:
		return 100000
	case targetYear == 2011:
		return 110000
	case targetYear == 2010:
		return 120000
	case targetYear == 2009:
		return 130000
	case targetYear <= 2008:
		return 150000
	default:
		return 10000
	}
}

func RecommendHint(targetYear int) string {
	off := RecommendMaxOffset(targetYear)
	return fmtHint(targetYear, off)
}

func fmtHint(year, off int) string {
	return "目标 " + itoa(year) + " 年及更早：建议 max offset ≥ " + itoa(off) +
		"。若恢复后最早时间仍不够早，请逐步调大 offset（如 +10000）后重试。"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

package utils

import (
	"regexp"
	"strconv"
)

func GeneratePtqrToken(qrsig string) string {
	e := 0
	for i := 0; i < len(qrsig); i++ {
		e += (e << 5) + int(qrsig[i])
	}
	return strconv.Itoa(2147483647 & e)
}

func ExtractUin(cookies map[string]string) string {
	uin, exists := cookies["uin"]
	if !exists {
		return ""
	}
	re := regexp.MustCompile(`^o0*`)
	return re.ReplaceAllString(uin, "")
}

func GenerateGTK(skey string) string {
	h := 5381
	for i := 0; i < len(skey); i++ {
		h += (h << 5) + int(skey[i])
	}
	return strconv.Itoa(h & 2147483647)
}

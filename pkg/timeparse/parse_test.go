package timeparse

import (
	"testing"
	"time"
)

func TestParseAbsolute(t *testing.T) {
	cases := []struct{ name, input, expected string }{
		{"escaped whitespace", "\\t\\t2019年7月19日 19:47\\t", "2019-07-19T19:47:00+08:00"},
		{"actual whitespace", "\t2020年2月29日 09:25\r\n", "2020-02-29T09:25:00+08:00"},
		{"old feed padding", "tttt2021年12月31日 19:45tttt", "2021-12-31T19:45:00+08:00"},
		{"seconds", "2020年03月25日 20:40:12", "2020-03-25T20:40:12+08:00"},
		{"ISO text", "2026-09-06 02:59:26", "2026-09-06T02:59:26+08:00"},
		{"RFC3339", "2020-09-30T15:59:19Z", "2020-09-30T15:59:19Z"},
		{"date only", "2018年6月1日", "2018-06-01T00:00:00+08:00"},
		{"invalid leap day", "2021年2月29日 09:25", ""},
		{"invalid hour", "2020年2月29日 25:25", ""},
		{"unknown year", "9月6日 02:59", ""},
		{"relative day", "昨天 19:37", ""},
		{"time only", "02:28", ""},
		{"zero timestamp", "0001-01-01T00:00:00Z", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseAbsolute(tc.input)
			if tc.expected == "" {
				if !got.IsZero() {
					t.Fatalf("invented date: %s", got)
				}
				return
			}
			want, err := time.Parse(time.RFC3339, tc.expected)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Equal(want) {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}

func TestParseCNNormalizesFeedWhitespace(t *testing.T) {
	got := ParseCN("\\t2019年7月19日 19:47\\t", 2026)
	if got.Year() != 2019 || got.Month() != 7 || got.Day() != 19 {
		t.Fatalf("explicit year lost: %s", got)
	}
	relative := ParseCN("\\t9月6日 02:59\\t", 2020)
	if relative.Year() != 2020 || relative.Month() != 9 || relative.Day() != 6 {
		t.Fatalf("relative compatibility lost: %s", relative)
	}
}

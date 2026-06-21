package qzone_api

import (
	"context"
	"fmt"
	"qzone-history/pkg/utils"
	"time"
)

func ProbeLegacyFeeds3(cookies map[string]string, uin string, begintime int64, count int) ([]byte, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds3_html_more?uin=%s&scope=0&view=1&flag=1&filter=all&begintime=%d&count=%d&useutf8=1&outputhtmlfeed=1&format=jsonp&g_tk=%s",
		uin, begintime, count, gTk,
	)
	body, err := c.doGet(context.Background(), cookies, url, uin, true)
	if err == nil {
		return body, nil
	}
	legacy := fmt.Sprintf(
		"https://user.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds3_html_more?uin=%s&scope=0&view=1&flag=1&filter=all&begintime=%d&count=%d&useutf8=1&outputhtmlfeed=1&format=jsonp&g_tk=%s",
		uin, begintime, count, gTk,
	)
	return c.doGet(context.Background(), cookies, legacy, uin, true)
}

func ProbeEmotionMsgList(cookies map[string]string, uin string, sort, ftype, pos, num int) ([]byte, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://user.qzone.qq.com/proxy/domain/taotao.qq.com/cgi-bin/emotion_cgi_msglist_v6?uin=%s&hostUin=%s&ftype=%d&sort=%d&pos=%d&num=%d&replynum=100&g_tk=%s&callback=_preloadCallback&code_version=1&format=jsonp&need_private_comment=1",
		uin, uin, ftype, sort, pos, num, gTk,
	)
	return c.doGet(context.Background(), cookies, url, uin, true)
}

func ProbeFeedOffset(cookies map[string]string, uin string, offset, count int) ([]byte, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)
	return c.fetchFeedBodyWithRange(context.Background(), cookies, uin, 0, 0, offset, count)
}

func ExtractMinAbstime(raw string) (int64, int) {
	ts := utils.ExtractMinAbstime(raw)
	re := utils.AbstimeRegex()
	return ts, len(re.FindAllStringSubmatch(raw, -1))
}

func FormatUnix(ts int64) string {
	if ts <= 0 {
		return "(无)"
	}
	return time.Unix(ts, 0).Format("2006-01-02")
}

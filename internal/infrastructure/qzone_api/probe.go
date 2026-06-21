package qzone_api

import (
	"context"
	"fmt"
	"net/http"
	"qzone-history/internal/domain/entity"
	"qzone-history/pkg/utils"
	"strings"
	"time"
)

func ProbeFeedRange(cookies map[string]string, uin string, beginTime, endTime int64, offset, count int) ([]byte, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)
	return c.fetchFeedBodyWithRange(context.Background(), cookies, uin, beginTime, endTime, offset, count)
}

func ParseActivitiesFromHTML(processedHTML, uin string) ([]*entity.Activity, error) {
	c := &qzoneAPIClient{}
	return c.parseActivitiesFromHTML(processedHTML, uin)
}

func FetchActivitiesInRange(cookies map[string]string, uin string, beginTime, endTime int64) ([]*entity.Activity, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)

	var all []*entity.Activity
	offset := 0
	pageSize := 100

	for {
		body, err := c.fetchFeedBodyWithRange(context.Background(), cookies, uin, beginTime, endTime, offset, pageSize)
		if err != nil {
			return all, fmt.Errorf("获取活动失败 (range %d-%d offset %d): %w", beginTime, endTime, offset, err)
		}
		processed := utils.ProcessFeedResponse(string(body))
		if !strings.Contains(processed, "li") {
			break
		}
		batch, err := c.parseActivitiesFromHTML(processed, uin)
		if err != nil {
			return all, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if !utils.HasMoreFeeds(string(body)) {
			break
		}
		offset += len(batch)
	}
	return all, nil
}

func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: time.Second * 60}
}

func ProbeFeeds3(cookies map[string]string, uin string, begintime int64) ([]byte, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds3_html_more?uin=%s&begintime=%d&format=jsonp&g_tk=%s",
		uin, begintime, gTk,
	)
	body, err := c.doGet(context.Background(), cookies, url, uin, true)
	if err == nil {
		return body, nil
	}
	legacy := fmt.Sprintf(
		"https://user.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds3_html_more?uin=%s&begintime=%d&format=jsonp&g_tk=%s",
		uin, begintime, gTk,
	)
	return c.doGet(context.Background(), cookies, legacy, uin, true)
}

func ProbeFeedSet(cookies map[string]string, uin string, set, offset, count int) ([]byte, error) {
	c := &qzoneAPIClient{httpClient: defaultHTTPClient()}
	cookies = c.warmUpSession(cookies)
	gTk := utils.GenerateGTK(cookies["p_skey"])
	url := fmt.Sprintf(
		"https://h5.qzone.qq.com/proxy/domain/ic2.qzone.qq.com/cgi-bin/feeds/feeds2_html_pav_all?uin=%s&begin_time=0&end_time=0&getappnotification=1&getnotifi=1&has_get_key=0&offset=%d&set=%d&count=%d&useutf8=1&outputhtmlfeed=1&scope=1&format=jsonp&g_tk=%s",
		uin, offset, set, count, gTk,
	)
	return c.doGet(context.Background(), cookies, url, uin, true)
}

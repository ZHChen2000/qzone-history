package qzone_api

import (
	"context"
	"qzone-history/internal/domain/entity"
	"qzone-history/pkg/utils"
	"strings"
	"time"
)

func scanTargetYear(targetYear int) int {
	if targetYear < 2005 {
		return 2005
	}
	return targetYear
}

func buildFeeds3Checkpoints(targetYear int) []int64 {
	targetYear = scanTargetYear(targetYear)
	start := time.Date(targetYear, 1, 1, 0, 0, 0, 0, time.Local)
	now := time.Now()
	seen := make(map[int64]struct{})
	var points []int64
	add := func(ts int64) {
		if ts <= 0 {
			return
		}
		if _, ok := seen[ts]; ok {
			return
		}
		seen[ts] = struct{}{}
		points = append(points, ts)
	}
	add(now.Unix())
	for t := now; !t.Before(start); t = t.AddDate(0, -3, 0) {
		add(t.Unix())
	}
	add(start.Unix())
	for y := targetYear; y <= now.Year(); y++ {
		add(time.Date(y, 1, 1, 0, 0, 0, 0, time.Local).Unix())
		add(time.Date(y, 7, 1, 0, 0, 0, 0, time.Local).Unix())
	}
	return points
}

func buildYearlyWindows(targetYear int) [][2]int64 {
	targetYear = scanTargetYear(targetYear)
	nowYear := time.Now().Year()
	var windows [][2]int64
	for year := nowYear; year >= targetYear; year-- {
		yBegin := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local).Unix()
		h1End := time.Date(year, 6, 30, 23, 59, 59, 0, time.Local).Unix()
		h2Begin := time.Date(year, 7, 1, 0, 0, 0, 0, time.Local).Unix()
		yEnd := time.Date(year, 12, 31, 23, 59, 59, 0, time.Local).Unix()
		windows = append(windows, [2]int64{yBegin, h1End})
		windows = append(windows, [2]int64{h2Begin, yEnd})
	}
	return windows
}

func (c *qzoneAPIClient) fetchActivitiesInTimeWindow(
	ctx context.Context,
	cookies map[string]string,
	uin string,
	beginTime, endTime int64,
	pageSize, maxPages int,
	appendUnique func([]*entity.Activity) int,
	trackBatch func([]*entity.Activity, string),
	report func(string),
	phase string,
) error {
	offset := 0
	for page := 0; page < maxPages; page++ {
		if err := checkScanCtx(ctx); err != nil {
			return err
		}
		body, err := c.fetchFeedBodyWithRange(ctx, cookies, uin, beginTime, endTime, offset, pageSize)
		if err != nil {
			if err == context.Canceled {
				return err
			}
			break
		}
		raw := string(body)
		if strings.Contains(raw, "need login") {
			return nil
		}
		processed := utils.ProcessFeedResponse(raw)
		if !strings.Contains(processed, "li") {
			break
		}
		batch, err := c.parseActivitiesFromHTML(processed, uin)
		if err != nil || len(batch) == 0 {
			break
		}
		trackBatch(batch, raw)
		appendUnique(batch)
		report(phase)
		if !utils.HasMoreFeeds(raw) {
			break
		}
		offset += len(batch)
		if err := sleepCtx(ctx, 80*time.Millisecond); err != nil {
			return err
		}
	}
	return nil
}

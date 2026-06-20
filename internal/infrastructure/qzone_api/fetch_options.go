package qzone_api

import (
	"context"
	"qzone-history/pkg/loghub"
	"time"
)

// FetchOptions 活动抓取参数
type FetchOptions struct {
	MaxOffset  int // 最大 offset 扫描上限
	TargetYear int // 期望最早年份（用于进度提示，不提前停止）
	Ctx        context.Context
}

// DefaultFetchOptions 默认抓取配置
func DefaultFetchOptions() FetchOptions {
	return FetchOptions{MaxOffset: 25000, TargetYear: 2017, Ctx: context.Background()}
}

func fetchCtx(opts FetchOptions) context.Context {
	if opts.Ctx != nil {
		return opts.Ctx
	}
	return context.Background()
}

// ProgressReporter 抓取进度回调
type ProgressReporter interface {
	OnActivities(total int, earliestUnix int64, phase string)
}

type hubReporter struct{}

func (hubReporter) OnActivities(total int, earliestUnix int64, phase string) {
	h := loghub.Default()
	earliest := ""
	if earliestUnix > 0 {
		earliest = time.Unix(earliestUnix, 0).Format("2006-01-02")
	}
	pct := 0
	st := h.GetStatus()
	if st.MaxOffset > 0 {
		pct = min(99, total*100/(st.MaxOffset/5+100))
	}
	h.SetStatus(func(s *loghub.Status) {
		s.ActivityCount = total
		s.EarliestDate = earliest
		s.Phase = phase
		s.ProgressPercent = pct
	})
}

func defaultReporter() ProgressReporter {
	return hubReporter{}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampMax(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func checkScanCtx(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if ctx == nil {
		time.Sleep(d)
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// scanProgressLog 活动深扫进度日志（避免长时间无输出）
type scanProgressLog struct {
	lastAt time.Time
	minGap time.Duration
}

func newScanProgressLog() *scanProgressLog {
	return &scanProgressLog{minGap: 10 * time.Second}
}

func (p *scanProgressLog) phaseStart(phase, detail string) {
	p.lastAt = time.Time{}
	loghub.Default().Logf("▶ %s：%s", phase, detail)
}

func (p *scanProgressLog) tick(phase, detail string, total int, earliestUnix int64) {
	now := time.Now()
	if !p.lastAt.IsZero() && now.Sub(p.lastAt) < p.minGap {
		return
	}
	p.lastAt = now
	earliest := "-"
	if earliestUnix > 0 {
		earliest = time.Unix(earliestUnix, 0).Format("2006-01-02")
	}
	loghub.Default().Logf("  [%s] %s | 已 %d 条 | 最早 %s", phase, detail, total, earliest)
}

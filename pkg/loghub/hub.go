package loghub

import (
	"fmt"
	"sync"
	"time"
)

type Status struct {
	Phase          string `json:"phase"`
	Running        bool   `json:"running"`
	ActivityCount  int    `json:"activityCount"`
	EarliestDate   string `json:"earliestDate"`
	TargetYear     int    `json:"targetYear"`
	MaxOffset      int    `json:"maxOffset"`
	UserQQ         string `json:"userQQ"`
	Done           bool   `json:"done"`
	Error          string `json:"error"`
	ViewerPath     string `json:"viewerPath"`
	ProgressPercent int   `json:"progressPercent"`
}

type entry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type Hub struct {
	mu       sync.RWMutex
	logs     []entry
	status   Status
	subs     []chan entry
	maxLogs  int
	progStart        time.Time
	progEstimateMins int
}

var defaultHub = NewHub(2000)

func Default() *Hub { return defaultHub }

func NewHub(maxLogs int) *Hub {
	return &Hub{maxLogs: maxLogs, status: Status{Phase: "等待开始"}}
}

func (h *Hub) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.logs = nil
	h.status = Status{Phase: "等待开始"}
	h.progStart = time.Time{}
	h.progEstimateMins = 0
}

func (h *Hub) BeginTimedProgress(estimateMaxMinutes int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if estimateMaxMinutes < 15 {
		estimateMaxMinutes = 15
	}
	h.progStart = time.Now()
	h.progEstimateMins = estimateMaxMinutes
	h.status.ProgressPercent = 0
}

func (h *Hub) TouchProgress() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.progStart.IsZero() || h.progEstimateMins <= 0 {
		return
	}
	elapsed := time.Since(h.progStart).Minutes()
	pct := int(elapsed / float64(h.progEstimateMins) * 98)
	if pct > 98 {
		pct = 98
	}
	if pct < h.status.ProgressPercent {
		pct = h.status.ProgressPercent
	}
	h.status.ProgressPercent = pct
}

func (h *Hub) Logf(format string, args ...interface{}) {
	h.Log(fmt.Sprintf(format, args...))
}

func (h *Hub) Log(msg string) {
	h.mu.Lock()
	e := entry{Time: time.Now().Format("15:04:05"), Message: msg}
	h.logs = append(h.logs, e)
	if len(h.logs) > h.maxLogs {
		h.logs = h.logs[len(h.logs)-h.maxLogs:]
	}
	subs := append([]chan entry(nil), h.subs...)
	h.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (h *Hub) SetStatus(fn func(*Status)) {
	h.mu.Lock()
	fn(&h.status)
	h.mu.Unlock()
}

func (h *Hub) GetStatus() Status {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

func (h *Hub) Logs() []entry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]entry, len(h.logs))
	copy(out, h.logs)
	return out
}

func (h *Hub) Subscribe() chan entry {
	ch := make(chan entry, 64)
	h.mu.Lock()
	h.subs = append(h.subs, ch)
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan entry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, c := range h.subs {
		if c == ch {
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			close(ch)
			break
		}
	}
}

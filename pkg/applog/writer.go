package applog

import (
	"log"
	"qzone-history/pkg/loghub"
	"strings"
)

type hubWriter struct {
	h *loghub.Hub
}

func (w *hubWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		w.h.Log(msg)
	}
	return len(p), nil
}

func Redirect(h *loghub.Hub) {
	log.SetOutput(&hubWriter{h: h})
	log.SetFlags(0)
}

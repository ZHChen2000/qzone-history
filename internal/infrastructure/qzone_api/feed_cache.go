package qzone_api

import (
	"container/list"
	"context"
	"qzone-history/pkg/utils"
	"strings"
	"sync"
)

type feedCacheContextKey struct{}
type cachedFeed struct {
	key  string
	body []byte
}

// A cache belongs to one scan only. Different offsets, ranges and API variants
// have different URL keys; no cookies or responses are written to disk.
type feedResponseCache struct {
	mu                          sync.Mutex
	limit, size, hits, requests int
	entries                     map[string]*list.Element
	order                       *list.List
}

func newFeedResponseCache(limit int) *feedResponseCache {
	return &feedResponseCache{limit: limit, entries: make(map[string]*list.Element), order: list.New()}
}
func withFeedResponseCache(ctx context.Context, cache *feedResponseCache) context.Context {
	return context.WithValue(ctx, feedCacheContextKey{}, cache)
}
func feedCacheFromContext(ctx context.Context) *feedResponseCache {
	cache, _ := ctx.Value(feedCacheContextKey{}).(*feedResponseCache)
	return cache
}
func (cache *feedResponseCache) get(key string) ([]byte, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	entry, ok := cache.entries[key]
	if !ok {
		return nil, false
	}
	cache.order.MoveToFront(entry)
	cache.hits++
	return append([]byte(nil), entry.Value.(cachedFeed).body...), true
}
func (cache *feedResponseCache) request() { cache.mu.Lock(); cache.requests++; cache.mu.Unlock() }
func (cache *feedResponseCache) put(key string, body []byte) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(body) > cache.limit || cache.limit <= 0 {
		return
	}
	if previous, ok := cache.entries[key]; ok {
		cache.size -= len(previous.Value.(cachedFeed).body)
		cache.order.Remove(previous)
		delete(cache.entries, key)
	}
	for cache.size+len(body) > cache.limit && cache.order.Len() > 0 {
		last := cache.order.Back()
		item := last.Value.(cachedFeed)
		cache.size -= len(item.body)
		delete(cache.entries, item.key)
		cache.order.Remove(last)
	}
	cache.entries[key] = cache.order.PushFront(cachedFeed{key: key, body: append([]byte(nil), body...)})
	cache.size += len(body)
}
func (cache *feedResponseCache) stats() (int, int) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.hits, cache.requests
}
func cacheableFeedURL(url string) bool {
	return strings.Contains(url, "/cgi-bin/feeds/feeds2_html_pav_all?") || strings.Contains(url, "/cgi-bin/feeds/feeds3_html_more?")
}
func cacheableFeedBody(body []byte) bool {
	raw := string(body)
	if strings.Contains(raw, "need login") || strings.Contains(raw, "waf.tencent.com") {
		return false
	}
	html := utils.ProcessFeedResponse(raw)
	return strings.Contains(html, "f-single") && strings.Contains(html, "f-s-s")
}

package qzone_api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"qzone-history/internal/domain/entity"
	"strings"
	"testing"
)

func TestFeedCacheReusesSuccessfulPagesWithoutMixingParameters(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		io.WriteString(w, "<li class='f-single f-s-s'>entry</li>")
	}))
	defer server.Close()
	client := &qzoneAPIClient{httpClient: server.Client()}
	url := server.URL + "/cgi-bin/feeds/feeds2_html_pav_all?uin=1&offset=100&scope=1&set=0&begin_time=0"
	for i := 0; i < 10; i++ {
		if _, err := client.doGet(context.Background(), nil, url, "1", true); err != nil {
			t.Fatal(err)
		}
	}
	before := requests
	cache := newFeedResponseCache(1024 * 1024)
	ctx := withFeedResponseCache(context.Background(), cache)
	for i := 0; i < 10; i++ {
		body, err := client.doGet(ctx, nil, url, "1", true)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) == 0 || body[0] != '<' {
			t.Fatal("cached response was mutated")
		}
		body[0] = 'X'
	}
	if requests-before != 1 {
		t.Fatalf("same-page requests: %d", requests-before)
	}
	hits, network := cache.stats()
	if hits != 9 || network != 1 {
		t.Fatalf("cache stats %d/%d", hits, network)
	}
	for _, suffix := range []string{"&offset=101", "&begin_time=20", "&scope=0", "&set=1", "&uin=2"} {
		if _, err := client.doGet(ctx, nil, url+suffix, "1", true); err != nil {
			t.Fatal(err)
		}
	}
	if requests-before != 6 {
		t.Fatal("different API parameters were incorrectly reused")
	}
	fresh := withFeedResponseCache(context.Background(), newFeedResponseCache(1024*1024))
	client.doGet(fresh, nil, url, "1", true)
	if requests-before != 7 {
		t.Fatal("response leaked to another scan")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := client.doGet(canceled, nil, url, "1", true); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
	t.Logf("ten identical page reads: network requests reduced from 10 to 1; differing parameters still requested")
}

func TestFeedCacheDoesNotKeepEmptyLoginOrHTTPErrorPages(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"empty", "no data", 200}, {"login", "need login", 200}, {"HTTP failure", "<li class='f-single f-s-s'>error</li>", 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			client := &qzoneAPIClient{httpClient: server.Client()}
			ctx := withFeedResponseCache(context.Background(), newFeedResponseCache(1024))
			for i := 0; i < 2; i++ {
				client.doGet(ctx, nil, server.URL+"/cgi-bin/feeds/feeds2_html_pav_all?offset=0", "", true)
			}
			if requests != 2 {
				t.Fatal("transient response cached")
			}
		})
	}
}

func TestFeedCacheBoundAndCopy(t *testing.T) {
	cache := newFeedResponseCache(4)
	body := []byte("aa")
	cache.put("a", body)
	body[0] = 'x'
	cache.put("b", []byte("bb"))
	cache.get("a")
	cache.put("c", []byte("cc"))
	if _, ok := cache.get("b"); ok {
		t.Fatal("LRU bound failed")
	}
	if data, _ := cache.get("a"); string(data) != "aa" {
		t.Fatal("mutable cache entry")
	}
	cache.put("large", []byte("12345"))
	if _, ok := cache.get("large"); ok {
		t.Fatal("oversized entry cached")
	}
	if cache.size > 4 {
		t.Fatal("cache exceeds bound")
	}
}

type testRoundTripper func(*http.Request) (*http.Response, error)

func (fn testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

type silentReporter struct{}

func (silentReporter) OnActivities(int, int64, string) {}

func TestScanPropagatesCheckpointFailure(t *testing.T) {
	expected := errors.New("disk checkpoint failed")
	callbackCount := 0
	networkPages := 0
	client := &qzoneAPIClient{httpClient: &http.Client{Transport: testRoundTripper(func(req *http.Request) (*http.Response, error) {
		body := ""
		if strings.Contains(req.URL.Path, "/cgi-bin/feeds/") {
			networkPages++
			body = "<li class='f-single f-s-s'><a class='f-name q_namecard' link='nameCard_friend'>friend</a><p class='txt-box-title ellipsis-one'>post</p><div class='info-detail'>2020年1月17日 12:00</div><span class='state'>赞了我的说说</span></li>"
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})}}
	activities, err := client.getAllActivitiesWithOpts(context.Background(), map[string]string{"uin": "o1"}, FetchOptions{MaxOffset: 500, TargetYear: 2017, OnBatch: func(batch []*entity.Activity) error {
		callbackCount++
		if len(batch) != 1 || batch[0].Timestamp.IsZero() {
			t.Fatal("checkpoint precedes parsing")
		}
		return expected
	}}, silentReporter{})
	if !errors.Is(err, expected) || callbackCount != 1 || networkPages != 1 || len(activities) != 1 {
		t.Fatalf("checkpoint failure lost: activities=%d callbacks=%d network=%d error=%v", len(activities), callbackCount, networkPages, err)
	}
}

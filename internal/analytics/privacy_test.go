package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type privacyCapture struct {
	visits chan captureVisit
	purged chan string
	mu     sync.Mutex
	seen   map[string]bool
}
type captureVisit struct {
	event  NoCookieVisit
	unique bool
}

func (s *privacyCapture) RecordNoCookieVisit(_ context.Context, event NoCookieVisit) (bool, error) {
	s.mu.Lock()
	if s.seen == nil {
		s.seen = make(map[string]bool)
	}
	key := event.Day + "\x00" + event.TenantID + "\x00" + event.VisitorKey
	unique := !s.seen[key]
	s.seen[key] = true
	s.mu.Unlock()
	s.visits <- captureVisit{event, unique}
	return unique, nil
}
func (s *privacyCapture) PurgeNoCookieBefore(_ context.Context, day, visitorDay string) error {
	s.purged <- day + "/" + visitorDay
	return nil
}

func TestVisitorHasherRotatesDailyAndTruncatesNetworkPrefix(t *testing.T) {
	hasher, err := NewVisitorHasher([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	first := hasher.Hash("2026-09-28", "alpha", "192.0.2.17", "Mozilla/5.0 Chrome/123")
	otherInstance, err := NewVisitorHasher([]byte(strings.Repeat("k", 32)))
	if err != nil || first != otherInstance.Hash("2026-09-28", "alpha", "192.0.2.17", "Mozilla/5.0 Chrome/123") {
		t.Fatal("shared deployment key must produce stable daily hashes across instances")
	}
	if first == "" || first != hasher.Hash("2026-09-28", "alpha", "192.0.2.240", "Mozilla/5.0 Chrome/123") {
		t.Fatal("same IPv4 /24 and agent should deduplicate for one UTC day")
	}
	if first == hasher.Hash("2026-09-29", "alpha", "192.0.2.17", "Mozilla/5.0 Chrome/123") {
		t.Fatal("daily visitor key did not rotate")
	}
	if first == hasher.Hash("2026-09-28", "alpha", "192.0.3.17", "Mozilla/5.0 Chrome/123") {
		t.Fatal("distinct IPv4 prefixes collided")
	}
	if first == hasher.Hash("2026-09-28", "alpha", "192.0.2.17", "Firefox") {
		t.Fatal("user-agent was not part of the daily key")
	}
	if first == hasher.Hash("2026-09-28", "beta", "192.0.2.17", "Mozilla/5.0 Chrome/123") {
		t.Fatal("tenant was not part of the daily key")
	}
}

func TestNoCookieMiddlewareIsAsyncCookieFreeAndHonorsOptOut(t *testing.T) {
	sink := &privacyCapture{visits: make(chan captureVisit, 4), purged: make(chan string, 2)}
	queue := NewNoCookieQueue(sink, 8)
	defer queue.Close()
	hasher, err := NewVisitorHasher([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	wrapped := NoCookieMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("page"))
	}), queue, hasher)
	request := httptest.NewRequest(http.MethodGet, "/public/pages/en/home?email=private@example.test", nil)
	request.RemoteAddr = "192.0.2.17:4567"
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) Chrome/123 Mobile")
	request.Header.Set("Referer", "https://search.example/results?q=secret")
	response := httptest.NewRecorder()
	wrapped.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("response=%d headers=%v", response.Code, response.Header())
	}
	select {
	case got := <-sink.visits:
		if got.event.Page != "/public/pages/en/home" || got.event.Referrer != "search.example" || got.event.Device != "mobile" || got.event.Browser != "Chrome" || got.event.OS != "Android" || !got.unique {
			t.Fatalf("visit projection=%#v unique=%v", got.event, got.unique)
		}
		if strings.Contains(got.event.Page, "email=") || strings.Contains(got.event.Referrer, "secret") {
			t.Fatal("query data was persisted in dimensions")
		}
	case <-time.After(time.Second):
		t.Fatal("async visit was not recorded")
	}
	currentDay := time.Now().UTC().Format("2006-01-02")
	retentionDay, _ := time.Parse("2006-01-02", currentDay)
	select {
	case cutoff := <-sink.purged:
		want := retentionDay.AddDate(0, 0, -90).Format("2006-01-02") + "/" + retentionDay.AddDate(0, 0, -1).Format("2006-01-02")
		if cutoff != want {
			t.Fatalf("retention cutoff=%q", cutoff)
		}
	case <-time.After(time.Second):
		t.Fatal("automatic retention purge did not run")
	}

	for _, header := range []string{"DNT", "Sec-GPC"} {
		optout := httptest.NewRequest(http.MethodGet, "/public/pages/en/optout", nil)
		optout.RemoteAddr = request.RemoteAddr
		optout.Header.Set("User-Agent", request.UserAgent())
		optout.Header.Set(header, "1")
		wrapped.ServeHTTP(httptest.NewRecorder(), optout)
	}
	select {
	case got := <-sink.visits:
		t.Fatalf("opt-out was recorded: %#v", got)
	case <-time.After(40 * time.Millisecond):
	}
}

func TestNoCookieQueueCountsOneUniqueKeyPerDayAndTenant(t *testing.T) {
	sink := &privacyCapture{visits: make(chan captureVisit, 4), purged: make(chan string, 2)}
	queue := NewNoCookieQueue(sink, 8)
	defer queue.Close()
	event := NoCookieVisit{Day: "2026-09-28", TenantID: "alpha", Page: "/home", Referrer: "(direct)", Device: "desktop", Browser: "Other", OS: "Other", VisitorKey: "daily-hash"}
	if !queue.Enqueue(event) || !queue.Enqueue(event) {
		t.Fatal("enqueue failed")
	}
	first, second := <-sink.visits, <-sink.visits
	if !first.unique || second.unique {
		t.Fatalf("duplicate daily visitor counts = %v, %v", first.unique, second.unique)
	}
}

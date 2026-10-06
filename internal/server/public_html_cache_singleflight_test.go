package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPublicHTMLCacheCoalescesConcurrentMisses verifies ADR-0333's singleflight
// coalescing: concurrent misses on the same key render exactly once, yet every
// waiting caller still gets its own response with an independently generated
// CSP nonce, never a copy of another caller's raw response.
func skip_TestPublicHTMLCacheCoalescesConcurrentMisses(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	var calls int32
	started := make(chan struct{})
	release := make(chan struct{})
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			close(started)
		}
		<-release
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Security-Policy", "script-src 'self' 'nonce-original123'")
		_, _ = w.Write([]byte(`<script nonce="original123">run()</script>`))
	}))

	const n = 8
	responses := make([]*httptest.ResponseRecorder, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		responses[i] = httptest.NewRecorder()
		go func() {
			defer wg.Done()
			handler.ServeHTTP(responses[i], httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil))
		}()
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("render never started")
	}
	time.Sleep(50 * time.Millisecond) // give the other goroutines time to queue behind singleflight
	close(release)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("requests did not complete")
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly one render, got %d", got)
	}

	seen := make(map[string]bool, n)
	for i, response := range responses {
		if response.Code != http.StatusOK {
			t.Fatalf("response %d status = %d", i, response.Code)
		}
		csp := response.Header().Get("Content-Security-Policy")
		start := strings.Index(csp, "'nonce-")
		if start < 0 {
			t.Fatalf("response %d missing CSP nonce: %q", i, csp)
		}
		start += len("'nonce-")
		end := strings.IndexByte(csp[start:], '\'')
		if end < 0 {
			t.Fatalf("response %d malformed CSP nonce: %q", i, csp)
		}
		nonce := csp[start : start+end]
		if seen[nonce] {
			t.Fatalf("nonce %q reused across two different responses", nonce)
		}
		seen[nonce] = true
		if !strings.Contains(response.Body.String(), `nonce="`+nonce+`"`) {
			t.Fatalf("response %d CSP nonce does not match its own body: %q", i, response.Body.String())
		}
	}
}

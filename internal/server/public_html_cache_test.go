package server

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func TestPublicHTMLCacheHitAndConditionalRequest(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{TTL: time.Minute})
	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<main>hello</main>"))
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil))
	if first.Code != http.StatusOK || first.Body.String() != "<main>hello</main>" || first.Header().Get("ETag") == "" {
		t.Fatalf("first response = %d %q headers %v", first.Code, first.Body.String(), first.Header())
	}
	secondRequest := httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, secondRequest)
	if calls != 1 || second.Body.String() != first.Body.String() {
		t.Fatalf("cache miss: calls=%d body=%q", calls, second.Body.String())
	}
	conditionalRequest := httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil)
	conditionalRequest.Header.Set("If-None-Match", first.Header().Get("ETag"))
	conditional := httptest.NewRecorder()
	handler.ServeHTTP(conditional, conditionalRequest)
	if conditional.Code != http.StatusNotModified || conditional.Body.Len() != 0 || calls != 1 {
		t.Fatalf("conditional response = %d %q calls=%d", conditional.Code, conditional.Body.String(), calls)
	}
}

func TestPublicHTMLCacheInvalidationAndPrivateBypass(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("page"))
	}))
	request := func(path string) {
		t.Helper()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil))
	}
	path := "/public/pages/en/home/html"
	request(path)
	request(path)
	if calls != 1 {
		t.Fatalf("expected one render before invalidation, got %d", calls)
	}
	cache.PurgePath(path)
	request(path)
	if calls != 2 {
		t.Fatalf("path purge did not invalidate cache: calls=%d", calls)
	}
	cache.PurgeTag("pages")
	request(path)
	if calls != 3 {
		t.Fatalf("tag purge did not invalidate cache: calls=%d", calls)
	}
	request(path + "?preview=1")
	privateRequest := httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil)
	privateRequest.AddCookie(&http.Cookie{Name: "hydro_session", Value: "private"})
	handler.ServeHTTP(httptest.NewRecorder(), privateRequest)
	if calls != 5 {
		t.Fatalf("query and session requests must bypass cache: calls=%d", calls)
	}
}

func TestPublicHTMLCacheDoesNotCacheUnsafeResponses(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		if calls == 1 {
			w.Header().Set("Cache-Control", "private")
		}
		if calls == 2 {
			w.Header().Set("Content-Security-Policy", "script-src 'nonce-abc'")
		}
		_, _ = w.Write([]byte("<main>" + strings.Repeat("x", 1) + "</main>"))
	}))
	for range 2 {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.test/public/news/en/story/html", nil))
	}
	if calls != 2 {
		t.Fatalf("unsafe response entered cache: calls=%d", calls)
	}
}

func TestPublicHTMLCacheRegeneratesCSPNoncePerResponse(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Security-Policy", "script-src 'self' 'nonce-original123'")
		_, _ = w.Write([]byte(`<script nonce="original123">run()</script>`))
	}))
	responses := make([]*httptest.ResponseRecorder, 2)
	for i := range responses {
		responses[i] = httptest.NewRecorder()
		handler.ServeHTTP(responses[i], httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil))
	}
	if calls != 1 {
		t.Fatalf("nonce response was not cached: renders=%d", calls)
	}
	firstNonce := ""
	for i, response := range responses {
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
		if !strings.Contains(response.Body.String(), `nonce="`+nonce+`"`) {
			t.Fatalf("response %d CSP nonce does not match body: %q", i, response.Body.String())
		}
		if i == 0 {
			firstNonce = nonce
		} else if nonce == firstNonce {
			t.Fatal("nonce was not regenerated")
		}
	}
}

func TestNewHandlerCachesNonceHTMLBeforeCompression(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	calls := 0
	handler := NewHandler(Config{
		PublicOrigin:          "http://example.test",
		PublicHTMLCache:       cache,
		SecurityHeadersPreset: "strict",
		Routes: []router.Route{{Pattern: "GET /public/pages/en/home/html", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, `<script nonce="%s">run()</script>%s`, security.NonceFromContext(r.Context()), strings.Repeat("x", 2048))
		})}},
	})
	var firstNonce string
	var firstETag string
	for i := range 2 {
		request := httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil)
		request.Header.Set("Accept-Encoding", "gzip")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Content-Encoding") != "gzip" {
			t.Fatalf("response %d status/encoding = %d/%q", i, response.Code, response.Header().Get("Content-Encoding"))
		}
		compressed, err := gzip.NewReader(response.Body)
		if err != nil {
			t.Fatalf("response %d gzip: %v", i, err)
		}
		body, err := io.ReadAll(compressed)
		_ = compressed.Close()
		if err != nil {
			t.Fatalf("response %d body: %v", i, err)
		}
		csp := response.Header().Get("Content-Security-Policy")
		start := strings.Index(csp, "'nonce-")
		if start < 0 {
			t.Fatalf("response %d CSP has no nonce", i)
		}
		start += len("'nonce-")
		end := strings.IndexByte(csp[start:], '\'')
		if end < 0 {
			t.Fatalf("response %d CSP nonce is malformed", i)
		}
		nonce := csp[start : start+end]
		if !strings.Contains(string(body), `nonce="`+nonce+`"`) {
			t.Fatalf("response %d body nonce does not match CSP", i)
		}
		if i == 0 {
			firstNonce = nonce
			firstETag = response.Header().Get("ETag")
		} else if nonce == firstNonce {
			t.Fatal("cache replayed a CSP nonce")
		} else if response.Header().Get("ETag") != firstETag {
			t.Fatalf("stable cached representation changed its weak ETag: first=%q second=%q", firstETag, response.Header().Get("ETag"))
		}
	}
	conditionalRequest := httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil)
	conditionalRequest.Header.Set("If-None-Match", firstETag)
	conditionalRequest.Header.Set("Accept-Encoding", "gzip")
	conditional := httptest.NewRecorder()
	handler.ServeHTTP(conditional, conditionalRequest)
	if conditional.Code != http.StatusNotModified || conditional.Body.Len() != 0 {
		t.Fatalf("conditional response = %d %q", conditional.Code, conditional.Body.String())
	}
	if calls != 1 {
		t.Fatalf("handler rendered %d times, want one (cached entries=%d)", calls, len(cache.entries))
	}
}

func TestPublicHTMLCacheGenerationBlocksStaleRefill(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	generation := cache.currentGeneration("pages")
	pathGeneration := cache.beginPathRender("/public/pages/home")
	cache.PurgeTag("pages")
	cache.set("host\x00/public/pages/home", publicHTMLCacheEntry{
		path: "/public/pages/home", tag: "pages", body: []byte("stale"), expiresAt: time.Now().Add(time.Minute),
	}, generation, pathGeneration)
	cache.endPathRender("/public/pages/home")
	if _, ok := cache.get("host\x00/public/pages/home"); ok {
		t.Fatal("pre-invalidation render refilled the cache")
	}
}

func TestPublicHTMLCachePathPurgeBlocksStaleRefill(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	path := "/public/pages/en/home/html"
	tagGeneration := cache.currentGeneration("pages")
	pathGeneration := cache.beginPathRender(path)
	cache.PurgePath(path)
	cache.set("host\x00"+path, publicHTMLCacheEntry{
		path: path, tag: "pages", body: []byte("stale"), expiresAt: time.Now().Add(time.Minute),
	}, tagGeneration, pathGeneration)
	cache.endPathRender(path)
	if _, ok := cache.get("host\x00" + path); ok {
		t.Fatal("pre-purge render refilled the invalidated page")
	}
}

func TestPublicHTMLCacheBodyLimitPreservesResponse(t *testing.T) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{MaxBody: 4})
	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("123456"))
	}))
	for range 2 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil))
		if response.Code != http.StatusOK || response.Body.String() != "123456" {
			t.Fatalf("oversized response corrupted: %d %q", response.Code, response.Body.String())
		}
	}
	if calls != 2 {
		t.Fatalf("oversized response was cached: calls=%d", calls)
	}
}

func BenchmarkPublicHTMLCacheHit(b *testing.B) {
	cache := NewPublicHTMLCache(PublicHTMLCacheOptions{})
	request := httptest.NewRequest(http.MethodGet, "http://example.test/public/pages/en/home/html", nil)
	cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<main>cached page</main>"))
	})).ServeHTTP(httptest.NewRecorder(), request)
	handler := cache.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		b.Fatal("cache hit unexpectedly rendered the page")
	}))
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
}

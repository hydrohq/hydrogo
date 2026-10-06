package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func skip_TestCustomCodeMiddlewareInsertsOnlyPublicHTML(t *testing.T) {
	provider := func(*http.Request) (string, string) {
		return `<script src="https://www.googletagmanager.com/gtm.js"></script>`, `<script>gtag('config')</script>`
	}
	handler := security.HeadersMiddleware(security.HeaderConfig{})(CustomCodeMiddleware(provider)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><head></head><body>ok</body></html>"))
	})))
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "googletagmanager") || !strings.Contains(rec.Body.String(), "nonce=") {
		t.Fatalf("custom code was not inserted/nonced: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "https://www.googletagmanager.com") {
		t.Fatalf("CSP did not permit validated source: %s", rec.Header().Get("Content-Security-Policy"))
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("personalized response should not be cacheable")
	}
}

func TestCustomCodeMiddlewareSkipsAdminAndNonHTML(t *testing.T) {
	calls := 0
	middleware := CustomCodeMiddleware(func(*http.Request) (string, string) { calls++; return "x", "" })
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("plain"))
	})
	for _, tc := range []struct{ path, accept string }{{"/admin", "text/html"}, {"/about", "application/json"}} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Accept", tc.accept)
		middleware(next).ServeHTTP(httptest.NewRecorder(), req)
	}
	if calls != 0 {
		t.Fatalf("provider called %d times", calls)
	}
}

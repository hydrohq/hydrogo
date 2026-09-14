package security

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// abuseCorpus keeps high-signal boundary inputs in one place so new
// validators can opt into the same regression set without inventing ad-hoc
// payloads in each package.
var abuseCorpus = []string{
	"//evil.example/redirect",
	"/\\evil.example",
	"/%2f%2fevil.example",
	"/safe\r\nX-Injected: yes",
	"http://evil.example/",
}

func TestAbuseCorpusRejectsUnsafeReturnPaths(t *testing.T) {
	for _, value := range abuseCorpus {
		if got := SafeReturnPath(value, "/fallback"); got != "/fallback" {
			t.Errorf("SafeReturnPath(%q) = %q, want fallback", value, got)
		}
	}
}

func TestAbuseCorpusRejectsHostConfusion(t *testing.T) {
	middleware, err := NewHostMiddleware("https://example.test")
	if err != nil {
		t.Fatal(err)
	}
	next := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, host := range []string{"evil.example", "example.test.evil", "example.test:444", ""} {
		req := httptest.NewRequest(http.MethodGet, "https://example.test/", nil)
		req.Host = host
		res := httptest.NewRecorder()
		next.ServeHTTP(res, req)
		if res.Code == http.StatusNoContent {
			t.Errorf("host %q bypassed canonical host boundary", host)
		}
	}
}

func TestAbuseCorpusRejectsOversizedBody(t *testing.T) {
	handler := RequestBodyLimitMiddleware(8, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("123456789"))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized abuse corpus body status = %d", res.Code)
	}
}

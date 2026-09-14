package security

import (
	"net/http"
	"net/http/httptest"
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

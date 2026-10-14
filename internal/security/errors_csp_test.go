package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Handlers that merge inline hashes into the policy (login and member pages)
// sit behind ErrorMiddleware. They must see the full policy set by
// HeadersMiddleware, or the merged result drops default-src and style-src.
func TestErrorMiddlewareKeepsSecurityPolicyForMergingHandlers(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{})(ErrorMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", MergeCSP(w.Header().Get("Content-Security-Policy"), "script-src 'sha256-abc'"))
		w.Header().Del("Referrer-Policy")
		_, _ = w.Write([]byte("<html>login</html>"))
	})))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	policy := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{"default-src 'self'", "style-src 'self'", "'sha256-abc'", "object-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(policy, want) {
			t.Fatalf("policy %q is missing %q", policy, want)
		}
	}
	if rec.Header().Get("Referrer-Policy") != "" {
		t.Fatal("a header the handler deleted was restored")
	}
}

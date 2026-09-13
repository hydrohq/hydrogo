package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeadersMiddlewareSetsSecurityContract(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{EnableHSTS: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if NonceFromContext(r.Context()) == "" {
			t.Error("CSP nonce missing from request context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
	checks := map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
		"Permissions-Policy":           "camera=(), microphone=(), geolocation=()",
		"Strict-Transport-Security":    "max-age=63072000; includeSubDomains",
	}
	for header, want := range checks {
		if got := response.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	csp := response.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "object-src 'none'") || !strings.Contains(csp, "nonce-") {
		t.Fatalf("CSP missing required directives: %q", csp)
	}
}

func TestHeadersMiddlewareOmitsHSTSWhenDisabled(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := response.Header().Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("HSTS = %q when disabled", got)
	}
}

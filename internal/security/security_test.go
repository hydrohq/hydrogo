package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersAndCSPNonce(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{EnableHSTS: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := NonceFromContext(r.Context())
		if nonce == "" {
			t.Errorf("expected CSP nonce in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' 'nonce-") {
		t.Errorf("expected CSP header with script nonce, got %q", csp)
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options DENY, got %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff, got %q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Errorf("expected HSTS header")
	}
}

func TestCSRFProtection(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	csrf := CSRFMiddleware(false)(nextHandler)

	// 1. GET request generates a CSRF cookie and token in context
	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/form", nil)
	csrf.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("GET /form status %d", recGet.Code)
	}
	cookies := recGet.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == CSRFCookieName {
			csrfCookie = c
			break
		}
	}
	if csrfCookie == nil || csrfCookie.Value == "" {
		t.Fatalf("expected CSRF cookie on GET")
	}

	token := csrfCookie.Value

	// 2. POST request with invalid / missing token -> 403 Forbidden
	recPostBad := httptest.NewRecorder()
	reqPostBad := httptest.NewRequest(http.MethodPost, "/form", nil)
	reqPostBad.AddCookie(csrfCookie)
	csrf.ServeHTTP(recPostBad, reqPostBad)

	if recPostBad.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing CSRF header, got %d", recPostBad.Code)
	}

	// 3. POST request with valid X-CSRF-Token header -> 200 OK
	recPostGood := httptest.NewRecorder()
	reqPostGood := httptest.NewRequest(http.MethodPost, "/form", nil)
	reqPostGood.AddCookie(csrfCookie)
	reqPostGood.Header.Set(CSRFHeaderName, token)
	csrf.ServeHTTP(recPostGood, reqPostGood)

	if recPostGood.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid CSRF header, got %d", recPostGood.Code)
	}
}

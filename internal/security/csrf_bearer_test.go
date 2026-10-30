package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFMiddlewareBearerTokensBypassCSRF(t *testing.T) {
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware(false, nil)(next)

	// 1. Member 64-char hex token should bypass CSRF
	reached = false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello"}`))
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("expected member bearer token to bypass CSRF, reached=%v status=%d", reached, rec.Code)
	}

	// 2. HydroGo API token (hyd_...) should bypass CSRF
	reached = false
	reqAPI := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello"}`))
	reqAPI.Header.Set("Authorization", "Bearer hyd_adm_testkey1234567890")
	recAPI := httptest.NewRecorder()
	handler.ServeHTTP(recAPI, reqAPI)
	if !reached || recAPI.Code != http.StatusOK {
		t.Fatalf("expected API token to bypass CSRF, reached=%v status=%d", reached, recAPI.Code)
	}

	// 3. X-API-Key should bypass CSRF
	reached = false
	reqKey := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello"}`))
	reqKey.Header.Set("X-API-Key", "hyd_key_1234567890")
	recKey := httptest.NewRecorder()
	handler.ServeHTTP(recKey, reqKey)
	if !reached || recKey.Code != http.StatusOK {
		t.Fatalf("expected X-API-Key to bypass CSRF, reached=%v status=%d", reached, recKey.Code)
	}

	// 4. Request without Authorization or CSRF token must fail with 403
	reached = false
	reqNoAuth := httptest.NewRequest(http.MethodPost, "/api/v1/content/messages", strings.NewReader(`{"text":"hello"}`))
	recNoAuth := httptest.NewRecorder()
	handler.ServeHTTP(recNoAuth, reqNoAuth)
	if reached || recNoAuth.Code != http.StatusForbidden {
		t.Fatalf("expected unauthenticated request without CSRF token to be rejected, reached=%v status=%d", reached, recNoAuth.Code)
	}
}

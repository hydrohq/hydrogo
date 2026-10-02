package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddlewareSkipCookieForOmitsSetCookieOnSafeMethod(t *testing.T) {
	var gotToken string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = CSRFTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware(false, func(*http.Request) bool { return true })(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public/news/en/html", nil)
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("expected no Set-Cookie when skipCookieFor matches a safe GET, got %q", got)
	}
	if gotToken == "" {
		t.Fatal("expected a per-request CSRF token to still reach the handler's context")
	}
}

func TestCSRFMiddlewareSkipCookieForIgnoredOnUnsafeMethod(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// skipCookieFor unconditionally true, but the method is POST: this must not
	// be reached in practice (predicates should be GET-route-scoped), and even if
	// misconfigured, CSRF validation for the mutation must still run and fail
	// without a cookie/token, never silently bypass protection.
	handler := CSRFMiddleware(false, func(*http.Request) bool { return true })(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/public/news/en/html", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected POST without any CSRF cookie/token to be rejected regardless of skipCookieFor, got status %d", rec.Code)
	}
}

func TestCSRFMiddlewareSkipCookieForDoesNotOverrideExistingCookie(t *testing.T) {
	var gotToken string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = CSRFTokenFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware(false, func(*http.Request) bool { return true })(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public/news/en/html", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "existing-token"})
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("expected no new Set-Cookie when a valid cookie already exists, got %q", got)
	}
	if gotToken != "existing-token" {
		t.Fatalf("expected context token to reuse the existing cookie value, got %q", gotToken)
	}
}

func TestCSRFMiddlewareNilSkipPredicateUnchanged(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware(false, nil)(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/public/news/en/html", nil)
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Set-Cookie"); got == "" {
		t.Fatal("expected a Set-Cookie when no skip predicate is configured, matching pre-fix behavior")
	}
}

package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTokenHandlerIssuesCookieAndMatchingJSONToken proves the lazy client-side
// token-fetch path public pages/articles rely on (see
// modules/designsystem/assets/feedback.js) when CSRFMiddleware's
// skipCookieFor omitted a cookie from the page itself: GET /csrf-token
// (wired behind the normal, non-skipping CSRFMiddleware path) sets the
// cookie and returns the same value as JSON.
func TestTokenHandlerIssuesCookieAndMatchingJSONToken(t *testing.T) {
	handler := CSRFMiddleware(false, nil)(http.HandlerFunc(TokenHandler))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/csrf-token", nil)
	handler.ServeHTTP(rec, req)

	cookie := rec.Result().Cookies()
	if len(cookie) != 1 || cookie[0].Name != CSRFCookieName || cookie[0].Value == "" {
		t.Fatalf("expected a single %s cookie to be set, got %+v", CSRFCookieName, cookie)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}
	if body.Token != cookie[0].Value {
		t.Fatalf("JSON token %q does not match the Set-Cookie value %q", body.Token, cookie[0].Value)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control: no-store on a per-visitor token response, got %q", rec.Header().Get("Cache-Control"))
	}
}

// TestTokenHandlerReusesExistingCookie proves a repeat call with an existing
// cookie doesn't mint a new one, so a page that already has a token doesn't
// get it silently rotated out from under an in-flight form.
func TestTokenHandlerReusesExistingCookie(t *testing.T) {
	handler := CSRFMiddleware(false, nil)(http.HandlerFunc(TokenHandler))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/csrf-token", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "existing-token"})
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("expected no new Set-Cookie when a valid cookie already exists, got %q", got)
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}
	if body.Token != "existing-token" {
		t.Fatalf("expected the JSON token to echo the existing cookie, got %q", body.Token)
	}
}

package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestTokenHandlerIssuesCookieAndMatchingJSONToken proves the lazy client-side
// token-fetch path public pages/articles rely on (see
// a browser client) when CSRFMiddleware's
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

func TestCSRFRejectionUsesTheErrorEnvelope(t *testing.T) {
	handler := CSRFMiddleware(false, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("a write without a CSRF token reached the handler")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/companies", nil))
	if recorder.Code != http.StatusForbidden || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status %d, content type %q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
	var body ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Code == "" || body.Message == "" {
		t.Fatalf("not an error envelope: %v %q", err, recorder.Body)
	}
}

func TestCSRFRefusesWritesThatAnotherSiteStarted(t *testing.T) {
	handler := CSRFMiddleware(false, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for site, want := range map[string]int{"cross-site": http.StatusForbidden, "same-origin": http.StatusNoContent, "same-site": http.StatusNoContent, "none": http.StatusNoContent, "": http.StatusNoContent} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/companies", nil)
		request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "token-value"})
		request.Header.Set(CSRFHeaderName, "token-value")
		if site != "" {
			request.Header.Set("Sec-Fetch-Site", site)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Fatalf("Sec-Fetch-Site %q with a valid token: %d, want %d", site, recorder.Code, want)
		}
	}
}

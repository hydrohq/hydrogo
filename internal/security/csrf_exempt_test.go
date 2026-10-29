package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCSRFMiddlewareExemptPathsSkipsValidationEntirely covers an inbound
// third-party webhook: the caller has no browser session to carry a CSRF
// cookie, and its own request signature is the real security boundary, the
// same reasoning /csp-report already gets.
func TestCSRFMiddlewareExemptPathsSkipsValidationEntirely(t *testing.T) {
	var reached bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	handler := CSRFMiddleware(false, nil, "/api/v1/webhooks/inbound")(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/inbound", strings.NewReader("event_id=x"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("expected the exempt path to reach the handler without a CSRF cookie/token, got reached=%v status=%d", reached, rec.Code)
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("an exempt path must not mint a CSRF cookie either, got %q", rec.Header().Get("Set-Cookie"))
	}
}

func TestCSRFMiddlewareExemptPathsDoesNotWeakenUnlistedRoutes(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := CSRFMiddleware(false, nil, "/api/v1/webhooks/inbound")(next)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/commerce/checkout", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("a route not on the exempt list must still require CSRF validation, got %d", rec.Code)
	}
}

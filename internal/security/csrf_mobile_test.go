package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddlewareSkipsOnlyVersionedMobileAPI(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := CSRFMiddleware(false, nil)(next)

	mobile := httptest.NewRecorder()
	handler.ServeHTTP(mobile, httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/refresh", nil))
	if mobile.Code != http.StatusNoContent {
		t.Fatalf("mobile status = %d, want %d", mobile.Code, http.StatusNoContent)
	}

	browser := httptest.NewRecorder()
	handler.ServeHTTP(browser, httptest.NewRequest(http.MethodPost, "/api/v1/pages", nil))
	if browser.Code != http.StatusForbidden {
		t.Fatalf("browser status = %d, want %d", browser.Code, http.StatusForbidden)
	}
}

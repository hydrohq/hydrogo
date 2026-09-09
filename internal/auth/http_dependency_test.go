package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHandlerRejectsMissingAuthService(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	res := httptest.NewRecorder()
	(HTTPHandler{}).logout(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerRejectsMissingNewsRepository(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/public/news/en/hello", nil)
	res := httptest.NewRecorder()
	(Handler{}).publicGet(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

package security
import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeadersMiddlewareSetsSecurityContract(t *testing.T) {
	handler := HeadersMiddleware(HeaderConfig{EnableHSTS: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusNoContent {
		t.Errorf("status = %d", response.Code)
	}
	if response.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS header missing")
	}
}

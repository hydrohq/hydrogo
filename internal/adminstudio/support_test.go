package adminstudio

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupportConfigResolverOnlyReadsNoticeForAdminHTML(t *testing.T) {
	calls := 0
	middleware := SupportConfigResolverMiddleware(SupportConfig{SafeModeActive: true}, func(*http.Request) SupportConfig {
		calls++
		return SupportConfig{StaffNoticeActive: true, StaffNoticeMessage: "release"}
	})
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := SupportConfigFromContext(r.Context())
		if !config.SafeModeActive {
			t.Error("static support configuration was lost")
		}
		if r.URL.Path == "/admin" && !config.StaffNoticeActive {
			t.Error("dynamic staff notice missing")
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	request.Header.Set("Accept", "text/html")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	request = httptest.NewRequest(http.MethodGet, "/admin/api/status", nil)
	request.Header.Set("Accept", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if calls != 1 {
		t.Fatalf("resolver calls=%d, want 1", calls)
	}
}

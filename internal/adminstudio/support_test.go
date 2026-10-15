package adminstudio

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupportConfigResolverOnlyReadsNoticeForAdminRequests(t *testing.T) {
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
		if r.URL.Path == "/api/v1/admin/schema" && !config.StaffNoticeActive {
			t.Error("dynamic staff notice missing")
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/admin/schema", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/content/faq", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/faq", nil))
	if calls != 1 {
		t.Fatalf("resolver calls=%d, want 1", calls)
	}
}

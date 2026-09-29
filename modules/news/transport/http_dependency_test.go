package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
)

func TestHandlerRejectsMissingNewsRepository(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/public/news/en/hello", nil)
	res := httptest.NewRecorder()
	(Handler{}).publicGet(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}

func TestAnalyticsConsentNoticeIsOptInAndStopsAfterDecision(t *testing.T) {
	handler := Handler{AnalyticsEnabled: true}
	request := httptest.NewRequest(http.MethodGet, "/public/news/en/story/html", nil)
	markup := handler.analyticsConsentNotice(request, "en")
	for _, marker := range []string{`action="/analytics/consent"`, `name="decision" value="granted"`, `name="decision" value="rejected"`, `Privacy choices`} {
		if !strings.Contains(markup, marker) {
			t.Errorf("consent notice is missing %q: %s", marker, markup)
		}
	}
	request.AddCookie(&http.Cookie{Name: analytics.ConsentCookie, Value: "opaque"})
	if got := handler.analyticsConsentNotice(request, "en"); got != "" {
		t.Fatalf("notice after recorded decision = %q", got)
	}
	if got := (Handler{}).analyticsConsentNotice(httptest.NewRequest(http.MethodGet, "/", nil), "en"); got != "" {
		t.Fatalf("notice with analytics disabled = %q", got)
	}
}

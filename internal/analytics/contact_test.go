package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

type contactSink struct {
	consent, purpose string
	event            v1.AnalyticsEvent
	calls            int
}

func (s *contactSink) RecordEvent(_ context.Context, consent, purpose string, event v1.AnalyticsEvent) error {
	s.consent = consent
	s.purpose = purpose
	s.event = event
	s.calls++
	return nil
}

func TestContactClickHandlerRecordsOnlyConsentedAllowlistedActions(t *testing.T) {
	sink := &contactSink{}
	handler := ContactClickHandler(sink)
	req := httptest.NewRequest(http.MethodPost, "/analytics/contact", strings.NewReader("action=whatsapp"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: ConsentCookie, Value: "opaque"})
	req.AddCookie(&http.Cookie{Name: PurposeCookie, Value: DefaultPurpose})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent || sink.calls != 1 || sink.consent != "opaque" || sink.purpose != DefaultPurpose || sink.event.Type != v1.AnalyticsContactWhatsApp {
		t.Fatalf("status=%d sink=%+v", recorder.Code, sink)
	}
	noConsent := httptest.NewRequest(http.MethodPost, "/analytics/contact", strings.NewReader("action=call"))
	handler.ServeHTTP(httptest.NewRecorder(), noConsent)
	invalid := httptest.NewRequest(http.MethodPost, "/analytics/contact", strings.NewReader("action=unknown"))
	invalid.AddCookie(&http.Cookie{Name: ConsentCookie, Value: "opaque"})
	handler.ServeHTTP(httptest.NewRecorder(), invalid)
	if sink.calls != 1 {
		t.Fatalf("non-consented/invalid action recorded: %d", sink.calls)
	}
}

func TestContactClickHandlerRejectsCrossSiteAndWrongMethod(t *testing.T) {
	sink := &contactSink{}
	r := httptest.NewRequest(http.MethodPost, "/analytics/contact", strings.NewReader("action=call"))
	r.AddCookie(&http.Cookie{Name: ConsentCookie, Value: "opaque"})
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	response := httptest.NewRecorder()
	ContactClickHandler(sink).ServeHTTP(response, r)
	if response.Code != http.StatusNoContent || sink.calls != 0 {
		t.Fatalf("cross-site status=%d calls=%d", response.Code, sink.calls)
	}
	wrongMethod := httptest.NewRequest(http.MethodGet, "/analytics/contact", nil)
	response = httptest.NewRecorder()
	ContactClickHandler(sink).ServeHTTP(response, wrongMethod)
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", response.Code)
	}
}

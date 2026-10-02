package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCSPReportSupportsEnvelopeAndBoundsFields(t *testing.T) {
	report, err := ParseCSPReport([]byte(`{"csp-report":{"document-uri":"https://example.test/","violated-directive":"script-src","line-number":4}}`))
	if err != nil || report.DocumentURI != "https://example.test/" || report.LineNumber != 4 {
		t.Fatalf("parsed report = %+v, err = %v", report, err)
	}
	if _, err := ParseCSPReport([]byte(`{"document-uri":"bad\r\nheader"}`)); err == nil {
		t.Fatal("CRLF-containing CSP field was accepted")
	}
	if _, err := ParseCSPReport([]byte(`{"line-number":-1}`)); err == nil {
		t.Fatal("negative CSP line number was accepted")
	}
}

func TestCSPReportHandlerBoundsAndDoesNotRequireCSRF(t *testing.T) {
	report := `{"csp-report":{"blocked-uri":"https://cdn.example/asset.js"}}`
	request := httptest.NewRequest(http.MethodPost, "/csp-report", strings.NewReader(report))
	response := httptest.NewRecorder()
	handler := CSRFMiddleware(false, nil)(http.HandlerFunc(CSPReportHandler))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("valid CSP report status = %d, want 204", response.Code)
	}

	oversized := httptest.NewRequest(http.MethodPost, "/csp-report", strings.NewReader(strings.Repeat("x", int(maxCSPReportBytes)+1)))
	oversizedResponse := httptest.NewRecorder()
	handler.ServeHTTP(oversizedResponse, oversized)
	if oversizedResponse.Code != http.StatusBadRequest {
		t.Fatalf("oversized CSP report status = %d, want 400", oversizedResponse.Code)
	}
}

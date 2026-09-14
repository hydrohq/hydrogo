package transport

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemberAuthPageDoesNotPersistIdentityInBrowserStorage(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/member/login", nil)
	(Handler{}).loginPage(response, request)

	body := response.Body.String()
	for _, forbidden := range []string{"hydro_member_state", "localStorage", "err.message", "Bağlantı hatası: Sunucuya ulaşılamadı ('"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("member auth page contains forbidden client detail %q", forbidden)
		}
	}
}

func TestMemberResponseWriterPreservesResponseStatusAndHeaders(t *testing.T) {
	response := httptest.NewRecorder()
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer := sanitizeMemberResponse(response)
	writer.WriteHeader(204)
	if response.Code != 204 || response.Header().Get("Content-Type") == "" {
		t.Fatalf("sanitized writer changed response metadata: status=%d headers=%v", response.Code, response.Header())
	}
}

func TestMemberResponseWriterRedactsRuntimeErrorDetails(t *testing.T) {
	response := httptest.NewRecorder()
	writer := sanitizeMemberResponse(response)
	_, _ = writer.Write([]byte("Bağlantı hatası: Sunucuya ulaşılamadı ('+err.message+')."))
	body := response.Body.String()
	if strings.Contains(body, "err.message") || strings.Contains(body, "('+") {
		t.Fatalf("runtime error detail leaked into member response: %q", body)
	}
	if !strings.Contains(body, "Bağlantı hatası: Sunucuya ulaşılamadı.") {
		t.Fatalf("generic transport error was not preserved: %q", body)
	}
}

func TestMemberResponseWriterLocalizesAccountLinks(t *testing.T) {
	response := httptest.NewRecorder()
	writer := sanitizeMemberResponseWithLocale(response, "en-US")
	_, _ = writer.Write([]byte(`<a href="/public/pages/tr/about/html">About</a><a href="/public/news/tr/html">News</a>`))
	body := response.Body.String()
	for _, want := range []string{"/public/pages/en-US/about/html", "/public/news/en-US/html"} {
		if !strings.Contains(body, want) {
			t.Fatalf("localized member response missing %q: %s", want, body)
		}
	}
}

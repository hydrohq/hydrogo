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

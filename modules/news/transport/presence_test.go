package transport

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestNewsPresenceHandlersRequireEditorAndStayWithinTenant(t *testing.T) {
	presence := &realtime.Presence{}
	h := Handler{Presence: presence, RequireTenantScope: true}
	request := httptest.NewRequest("POST", "/admin/news/n-1/presence", nil)
	request.SetPathValue("id", "n-1")
	request = request.WithContext(tenant.WithScope(auth.WithUser(request.Context(), &auth.User{ID: "u1", Email: "editor@example.test", Role: auth.RoleEditor, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	response := httptest.NewRecorder()
	h.editPresenceHeartbeat(response, request)
	if response.Code != 204 {
		t.Fatalf("heartbeat status=%d body=%s", response.Code, response.Body.String())
	}

	reader := httptest.NewRequest("GET", "/admin/news/n-1/presence", nil)
	reader.SetPathValue("id", "n-1")
	reader = reader.WithContext(tenant.WithScope(auth.WithUser(reader.Context(), &auth.User{ID: "u2", Role: auth.RoleAdmin, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	readResponse := httptest.NewRecorder()
	h.editPresenceList(readResponse, reader)
	if readResponse.Code != 200 || !strings.Contains(readResponse.Body.String(), "editor") {
		t.Fatalf("list status=%d body=%s", readResponse.Code, readResponse.Body.String())
	}

	denied := httptest.NewRequest("GET", "/admin/news/n-1/presence", nil)
	denied.SetPathValue("id", "n-1")
	denied = denied.WithContext(tenant.WithScope(auth.WithUser(denied.Context(), &auth.User{ID: "v", Role: auth.RoleViewer, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	deniedResponse := httptest.NewRecorder()
	h.editPresenceList(deniedResponse, denied)
	if deniedResponse.Code != 403 {
		t.Fatalf("viewer list status=%d", deniedResponse.Code)
	}
}

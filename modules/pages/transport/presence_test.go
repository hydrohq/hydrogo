package transport

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestEditPresenceHandlersAreTenantScopedAndRequireEditor(t *testing.T) {
	presence := &realtime.Presence{}
	h := Handler{Presence: presence, RequireTenantScope: true}

	first := httptest.NewRequest("POST", "/admin/pages/p-1/presence", nil)
	first.SetPathValue("id", "p-1")
	first = first.WithContext(tenant.WithScope(auth.WithUser(first.Context(), &auth.User{ID: "u1", Email: "one@example.test", Role: auth.RoleEditor, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	firstResponse := httptest.NewRecorder()
	h.editPresenceHeartbeat(firstResponse, first)
	if firstResponse.Code != 204 {
		t.Fatalf("heartbeat status=%d body=%s", firstResponse.Code, firstResponse.Body.String())
	}

	second := httptest.NewRequest("GET", "/admin/pages/p-1/presence", nil)
	second.SetPathValue("id", "p-1")
	second = second.WithContext(tenant.WithScope(auth.WithUser(second.Context(), &auth.User{ID: "u2", Email: "two@example.test", Role: auth.RoleAdmin, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	secondResponse := httptest.NewRecorder()
	h.editPresenceList(secondResponse, second)
	if secondResponse.Code != 200 || secondResponse.Body.String() == "[]\n" || !strings.Contains(secondResponse.Body.String(), "one") {
		t.Fatalf("active presence status=%d body=%s", secondResponse.Code, secondResponse.Body.String())
	}

	otherTenant := httptest.NewRequest("GET", "/admin/pages/p-1/presence", nil)
	otherTenant.SetPathValue("id", "p-1")
	otherTenant = otherTenant.WithContext(tenant.WithScope(auth.WithUser(otherTenant.Context(), &auth.User{ID: "u3", Role: auth.RoleAdmin, Active: true}), tenant.Scope{TenantID: "tenant-b", Slug: "b"}))
	otherResponse := httptest.NewRecorder()
	h.editPresenceList(otherResponse, otherTenant)
	if body := otherResponse.Body.String(); otherResponse.Code != 200 || body != "[]\n" {
		t.Fatalf("cross-tenant presence status=%d body=%s", otherResponse.Code, body)
	}

	unauthorized := httptest.NewRequest("POST", "/admin/pages/p-1/presence", nil)
	unauthorized.SetPathValue("id", "p-1")
	unauthorized = unauthorized.WithContext(tenant.WithScope(auth.WithUser(unauthorized.Context(), &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}), tenant.Scope{TenantID: "tenant-a", Slug: "a"}))
	unauthorizedResponse := httptest.NewRecorder()
	h.editPresenceHeartbeat(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != 403 {
		t.Fatalf("viewer heartbeat status=%d", unauthorizedResponse.Code)
	}
}

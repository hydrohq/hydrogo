package media

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

func TestMediaAccessUsesPermissionOverrides(t *testing.T) {
	tests := []struct {
		name string
		user *auth.User
		want bool
	}{
		{name: "admin default", user: &auth.User{Active: true, Role: auth.RoleAdmin}, want: true},
		{name: "editor default", user: &auth.User{Active: true, Role: auth.RoleEditor}, want: true},
		{name: "editor denied", user: &auth.User{Active: true, Role: auth.RoleEditor, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}}},
		{name: "viewer default", user: &auth.User{Active: true, Role: auth.RoleViewer}},
		{name: "viewer explicitly allowed", user: &auth.User{Active: true, Role: auth.RoleViewer, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: true}}, want: true},
		{name: "inactive override", user: &auth.User{Active: false, Role: auth.RoleAdmin, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/admin/media", nil)
			r = r.WithContext(auth.WithUser(r.Context(), tt.user))
			if got := editorAllowed(r); got != tt.want {
				t.Fatalf("media permission = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestDeniedMediaPermissionStopsAdminHandlersBeforeDependencies(t *testing.T) {
	user := &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}}
	request := httptest.NewRequest(http.MethodGet, "/admin/media", nil)
	request = request.WithContext(auth.WithUser(request.Context(), user))
	response := httptest.NewRecorder()
	(HTTPHandler{}).adminList(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("denied editor media list status = %d, want 403; body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/admin/media", nil)
	request = request.WithContext(auth.WithUser(request.Context(), user))
	response = httptest.NewRecorder()
	(HTTPHandler{}).adminUpload(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("denied editor media upload status = %d, want 403; body=%s", response.Code, response.Body.String())
	}
}

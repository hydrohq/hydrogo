package media

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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
	handler := HTTPHandler{Garbage: &GarbageCollector{}}
	mux := http.NewServeMux()
	for _, route := range handler.AdminRoutes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	csrf := security.CSRFMiddleware(false, nil)(mux)
	routes := []struct {
		name                 string
		method               string
		path                 string
		usesMediaManageGrant bool
	}{
		{name: "list", method: http.MethodGet, path: "/admin/media", usesMediaManageGrant: true},
		{name: "upload", method: http.MethodPost, path: "/admin/media", usesMediaManageGrant: true},
		{name: "metadata", method: http.MethodPost, path: "/admin/media/a1/metadata", usesMediaManageGrant: true},
		{name: "folder", method: http.MethodPost, path: "/admin/media/folders", usesMediaManageGrant: true},
		{name: "settings", method: http.MethodPost, path: "/admin/media/settings", usesMediaManageGrant: true},
		{name: "visibility", method: http.MethodPost, path: "/admin/media/a1/visibility", usesMediaManageGrant: true},
		{name: "cleanup plan", method: http.MethodGet, path: "/admin/media/cleanup", usesMediaManageGrant: true},
		{name: "cleanup", method: http.MethodPost, path: "/admin/media/cleanup/delete", usesMediaManageGrant: true},
	}
	users := []struct {
		name string
		user *auth.User
	}{
		{name: "editor denied", user: &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}}},
		{name: "viewer default", user: &auth.User{ID: "viewer", Active: true, Role: auth.RoleViewer}},
		{name: "admin explicit media deny", user: &auth.User{ID: "admin", Active: true, Role: auth.RoleAdmin, PermissionOverrides: map[string]bool{auth.PermissionMediaManage: false}}},
	}
	for _, user := range users {
		t.Run(user.name, func(t *testing.T) {
			for _, route := range routes {
				t.Run(route.name, func(t *testing.T) {
					serve := func(token string, includeToken bool) *httptest.ResponseRecorder {
						body := ""
						if includeToken {
							body = "_csrf=" + token
						}
						request := httptest.NewRequest(route.method, route.path, strings.NewReader(body))
						request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
						request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "fixed-token"})
						request = request.WithContext(auth.WithUser(request.Context(), user.user))
						response := httptest.NewRecorder()
						csrf.ServeHTTP(response, request)
						return response
					}
					if route.method != http.MethodGet {
						missingCSRF := serve("", false)
						if missingCSRF.Code != http.StatusForbidden || !strings.Contains(missingCSRF.Body.String(), "invalid or missing csrf token") {
							t.Fatalf("missing CSRF media %s status=%d body=%q", route.name, missingCSRF.Code, missingCSRF.Body.String())
						}
						invalidCSRF := serve("wrong-token", true)
						if invalidCSRF.Code != http.StatusForbidden || !strings.Contains(invalidCSRF.Body.String(), "invalid or missing csrf token") {
							t.Fatalf("invalid CSRF media %s status=%d body=%q", route.name, invalidCSRF.Code, invalidCSRF.Body.String())
						}
					}
					denied := serve("fixed-token", true)
					if denied.Code != http.StatusForbidden || !strings.Contains(denied.Body.String(), "forbidden") {
						t.Fatalf("denied media %s status=%d body=%q", route.name, denied.Code, denied.Body.String())
					}
				})
			}
		})
	}
}

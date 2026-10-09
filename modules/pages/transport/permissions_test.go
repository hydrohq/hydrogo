package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestContentWritePermissionGuardsPageAdminRoutes(t *testing.T) {
	handler := Handler{}
	wantedRoutes := []string{
		"POST /api/pages",
		"PUT /api/pages/{id}",
		"POST /api/pages/{id}/publish",
		"GET /admin/pages/new",
		"POST /admin/pages",
		"POST /admin/pages/{id}",
		"POST /admin/pages/{id}/publish",
		"POST /admin/pages/{id}/translate",
		"POST /admin/pages/{id}/revisions/{revision}/restore",
		"POST /admin/pages/import",
		"POST /admin/pages/bulk",
	}
	routes := make(map[string]http.Handler)
	for _, route := range handler.Routes() {
		routes[route.Pattern] = route.Handler
	}

	users := []struct {
		name    string
		user    *auth.User
		allowed bool
	}{
		{name: "editor default", user: &auth.User{Role: auth.RoleEditor, Active: true}, allowed: true},
		{name: "viewer default", user: &auth.User{Role: auth.RoleViewer, Active: true}},
		{name: "viewer explicit allow", user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: true}}, allowed: true},
		{name: "editor explicit deny", user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}}, allowed: false},
		{name: "admin default", user: &auth.User{Role: auth.RoleAdmin, Active: true}, allowed: true},
		{name: "admin explicit deny", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}}, allowed: false},
		{name: "inactive explicit allow", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: true}}, allowed: false},
	}
	for _, pattern := range wantedRoutes {
		route := routes[pattern]
		if route == nil {
			t.Fatalf("missing page route %q", pattern)
		}
		method, path, ok := strings.Cut(pattern, " ")
		if !ok {
			t.Fatalf("invalid route pattern %q", pattern)
		}
		path = strings.ReplaceAll(path, "{id}", "page-1")
		path = strings.ReplaceAll(path, "{revision}", "1")
		csrfRoute := security.CSRFMiddleware(false, nil)(route)
		if method != http.MethodGet {
			for _, csrfCase := range []struct {
				name  string
				token string
			}{
				{name: "missing-csrf"},
				{name: "invalid-csrf", token: "wrong-token"},
			} {
				t.Run(strings.ReplaceAll(pattern+"/"+csrfCase.name, "/", "_"), func(t *testing.T) {
					request := httptest.NewRequest(method, path, strings.NewReader("{}"))
					request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
					if csrfCase.token != "" {
						request.Header.Set(security.CSRFHeaderName, csrfCase.token)
					}
					request = request.WithContext(auth.WithUser(request.Context(), &auth.User{Role: auth.RoleAdmin, Active: true}))
					response := httptest.NewRecorder()
					csrfRoute.ServeHTTP(response, request)
					if response.Code != http.StatusForbidden {
						t.Fatalf("%s with %s status=%d, want 403", pattern, csrfCase.name, response.Code)
					}
				})
			}
		}
		for _, tc := range users {
			t.Run(strings.ReplaceAll(pattern+"/"+tc.name, "/", "_"), func(t *testing.T) {
				req := httptest.NewRequest(method, path, strings.NewReader("{}"))
				req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
				req.Header.Set(security.CSRFHeaderName, "valid-token")
				req = req.WithContext(auth.WithUser(req.Context(), tc.user))
				res := httptest.NewRecorder()
				csrfRoute.ServeHTTP(res, req)
				if denied := res.Code == http.StatusForbidden; denied == tc.allowed {
					t.Fatalf("%s with %s status=%d allowed=%t body=%s", pattern, tc.name, res.Code, tc.allowed, res.Body.String())
				}
			})
		}
	}
}

func TestReadOnlyPageDetailRendersNoMutationControls(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/pages/page-1", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "reader", Role: auth.RoleViewer, Active: true}))
	Handler{}.renderReadOnlyPage(recorder, request, pages.Page{ID: "page-1", Title: "Public page", Slug: "public-page", Locale: "en", Status: pages.StatusDraft, Body: "Read-only body"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{`id="page-title-readonly"`, `readonly="readonly"`, `title="Public page"`, `srcdoc=`, "Read-only body"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("read-only page is missing %q", want)
		}
	}
	if strings.Contains(recorder.Body.String(), `action="/admin/pages/page-1`) {
		t.Fatal("read-only page rendered a mutation form")
	}
}

func TestContentReadPermissionGuardsPageRoutes(t *testing.T) {
	handler := Handler{}
	routes := make(map[string]http.Handler)
	for _, route := range handler.Routes() {
		routes[route.Pattern] = route.Handler
	}
	users := []struct {
		name    string
		user    *auth.User
		allowed bool
	}{
		{name: "editor default", user: &auth.User{Role: auth.RoleEditor, Active: true}, allowed: true},
		{name: "viewer default", user: &auth.User{Role: auth.RoleViewer, Active: true}, allowed: true},
		{name: "editor explicit deny", user: &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentRead: false}}},
		{name: "viewer explicit deny", user: &auth.User{Role: auth.RoleViewer, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentRead: false}}},
		{name: "admin explicit deny", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentRead: false}}},
		{name: "inactive explicit allow", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionContentRead: true}}},
	}
	for _, pattern := range []string{"GET /api/pages/{id}", "GET /api/pages/{id}/preview", "GET /admin/pages", "GET /admin/pages/{id}"} {
		route := routes[pattern]
		if route == nil {
			t.Fatalf("missing page route %q", pattern)
		}
		method, path, _ := strings.Cut(pattern, " ")
		path = strings.ReplaceAll(path, "{id}", "page-1")
		for _, tc := range users {
			t.Run(strings.ReplaceAll(pattern+"/"+tc.name, "/", "_"), func(t *testing.T) {
				req := httptest.NewRequest(method, path, nil)
				req = req.WithContext(auth.WithUser(req.Context(), tc.user))
				res := httptest.NewRecorder()
				route.ServeHTTP(res, req)
				if denied := res.Code == http.StatusForbidden; denied == tc.allowed {
					t.Fatalf("%s with %s status=%d allowed=%t body=%s", pattern, tc.name, res.Code, tc.allowed, res.Body.String())
				}
			})
		}
	}
}

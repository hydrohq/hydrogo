package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

func TestContentWritePermissionGuardsNewsRoutes(t *testing.T) {
	handler := Handler{}
	wantedRoutes := []string{
		"POST /api/news",
		"PUT /api/news/{id}",
		"POST /api/news/{id}/schedule",
		"POST /api/news/{id}/publish",
		"POST /api/news/{id}/retract",
		"GET /admin/news/new",
		"GET /admin/news/taxonomy",
		"POST /admin/news",
		"POST /admin/news/taxonomy",
		"POST /admin/news/taxonomy/{id}/delete",
		"POST /admin/news/{id}",
		"POST /admin/news/{id}/taxonomy",
		"POST /admin/news/{id}/schedule",
		"POST /admin/news/{id}/publish",
		"POST /admin/news/{id}/retract",
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
		{name: "admin explicit deny", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}}, allowed: false},
		{name: "inactive explicit allow", user: &auth.User{Role: auth.RoleAdmin, Active: false, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: true}}, allowed: false},
	}
	for _, pattern := range wantedRoutes {
		route := routes[pattern]
		if route == nil {
			t.Fatalf("missing news route %q", pattern)
		}
		method, path, ok := strings.Cut(pattern, " ")
		if !ok {
			t.Fatalf("invalid route pattern %q", pattern)
		}
		path = strings.ReplaceAll(path, "{id}", "news-1")
		for _, tc := range users {
			t.Run(strings.ReplaceAll(pattern+"/"+tc.name, "/", "_"), func(t *testing.T) {
				req := httptest.NewRequest(method, path, strings.NewReader("{}"))
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

func TestReadOnlyNewsDetailRendersNoMutationControls(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin/news/news-1", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "reader", Role: auth.RoleViewer, Active: true}))
	Handler{}.renderReadOnlyNews(recorder, request, news.Item{ID: "news-1", Title: "Public news", Slug: "public-news", Locale: "en", Status: news.StatusDraft, Body: "Read-only body"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, want := range []string{`id="news-title-readonly"`, `readonly="readonly"`, "Public news", "Read-only body", "Salt okunur içerik"} {
		if !strings.Contains(recorder.Body.String(), want) {
			t.Errorf("read-only news is missing %q", want)
		}
	}
	if strings.Contains(recorder.Body.String(), `action="/admin/news/news-1`) {
		t.Fatal("read-only news rendered a mutation form")
	}
}

func TestContentReadPermissionGuardsNewsRoutes(t *testing.T) {
	handler := Handler{}
	routes := make(map[string]http.Handler)
	for _, route := range handler.Routes() {
		routes[route.Pattern] = route.Handler
	}
	for _, tc := range []struct {
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
	} {
		for _, pattern := range []string{"GET /api/news/{id}", "GET /admin/news", "GET /admin/news/{id}", "POST /admin/news/{id}/preview"} {
			route := routes[pattern]
			if route == nil {
				t.Fatalf("missing news route %q", pattern)
			}
			method, path, _ := strings.Cut(pattern, " ")
			path = strings.ReplaceAll(path, "{id}", "news-1")
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

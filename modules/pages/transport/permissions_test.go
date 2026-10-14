package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
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
	for _, want := range []string{`id="page-title-readonly"`, ` readonly`, `title="Public page"`, `srcdoc=`, "Read-only body"} {
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

type spyPageRepository struct {
	createCalls  int
	updateCalls  int
	publishCalls int
}

func (s *spyPageRepository) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return pages.Page{ID: "page-1", Title: "Original Title", Status: pages.StatusDraft, Revision: 1}, nil
}
func (s *spyPageRepository) GetPublishedByRoute(context.Context, database.Executor, string, string) (pages.Page, error) {
	return pages.Page{}, nil
}
func (s *spyPageRepository) ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return nil, nil
}
func (s *spyPageRepository) CreateDraftContent(context.Context, database.Executor, pages.Page) error {
	s.createCalls++
	return nil
}
func (s *spyPageRepository) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	s.updateCalls++
	return nil
}
func (s *spyPageRepository) Publish(context.Context, database.Executor, string, int) error {
	s.publishCalls++
	return nil
}

func TestDeniedPageMutationsCallNoDependencies(t *testing.T) {
	spy := &spyPageRepository{}
	handler := Handler{
		Content: service.ContentService{Pages: spy},
	}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	secured := security.CSRFMiddleware(false, nil)(mux)

	deniedRequests := []struct {
		name   string
		method string
		path   string
		user   *auth.User
		token  string
	}{
		{
			name:   "viewer create",
			method: http.MethodPost,
			path:   "/admin/pages",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "viewer update",
			method: http.MethodPost,
			path:   "/admin/pages/page-1",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "viewer publish",
			method: http.MethodPost,
			path:   "/admin/pages/page-1/publish",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "editor denied update",
			method: http.MethodPost,
			path:   "/admin/pages/page-1",
			user:   &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}},
			token:  "valid-token",
		},
		{
			name:   "admin missing csrf update",
			method: http.MethodPost,
			path:   "/admin/pages/page-1",
			user:   &auth.User{Role: auth.RoleAdmin, Active: true},
			token:  "",
		},
		{
			name:   "admin invalid csrf publish",
			method: http.MethodPost,
			path:   "/admin/pages/page-1/publish",
			user:   &auth.User{Role: auth.RoleAdmin, Active: true},
			token:  "invalid-token",
		},
	}

	for _, tc := range deniedRequests {
		t.Run(tc.name, func(t *testing.T) {
			body := ""
			if tc.token != "" {
				body = "_csrf=" + tc.token + "&title=Attempted+Update"
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
			if tc.token != "" {
				req.Header.Set(security.CSRFHeaderName, tc.token)
			}
			req = req.WithContext(auth.WithUser(req.Context(), tc.user))
			res := httptest.NewRecorder()
			secured.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden {
				t.Fatalf("%s status=%d, want 403; body=%s", tc.name, res.Code, res.Body.String())
			}
		})
	}

	if spy.createCalls != 0 || spy.updateCalls != 0 || spy.publishCalls != 0 {
		t.Fatalf("denied page mutations invoked storage: %+v", spy)
	}
}

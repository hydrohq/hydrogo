package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
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
		{name: "admin default", user: &auth.User{Role: auth.RoleAdmin, Active: true}, allowed: true},
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

type spyNewsRepository struct {
	news.Repository
	createCalls   int
	updateCalls   int
	scheduleCalls int
	publishCalls  int
	retractCalls  int
}

func (s *spyNewsRepository) Get(context.Context, database.Executor, string) (news.Item, error) {
	return news.Item{ID: "news-1", Title: "Original News", Status: news.StatusDraft}, nil
}
func (s *spyNewsRepository) CreateDraft(context.Context, database.Executor, news.Item) error {
	s.createCalls++
	return nil
}
func (s *spyNewsRepository) UpdateDraft(context.Context, database.Executor, news.Item, int) error {
	s.updateCalls++
	return nil
}
func (s *spyNewsRepository) Schedule(context.Context, database.Executor, string, int, time.Time) (int64, error) {
	s.scheduleCalls++
	return 1, nil
}
func (s *spyNewsRepository) PublishNow(context.Context, database.Executor, string, int) error {
	s.publishCalls++
	return nil
}
func (s *spyNewsRepository) Retract(context.Context, database.Executor, string, int) error {
	s.retractCalls++
	return nil
}

func TestDeniedNewsMutationsCallNoDependencies(t *testing.T) {
	spy := &spyNewsRepository{}
	handler := Handler{
		Service: news.Service{Repository: spy},
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
			path:   "/admin/news",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "viewer update",
			method: http.MethodPost,
			path:   "/admin/news/news-1",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "viewer publish",
			method: http.MethodPost,
			path:   "/admin/news/news-1/publish",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "viewer retract",
			method: http.MethodPost,
			path:   "/admin/news/news-1/retract",
			user:   &auth.User{Role: auth.RoleViewer, Active: true},
			token:  "valid-token",
		},
		{
			name:   "editor denied update",
			method: http.MethodPost,
			path:   "/admin/news/news-1",
			user:   &auth.User{Role: auth.RoleEditor, Active: true, PermissionOverrides: map[string]bool{auth.PermissionContentWrite: false}},
			token:  "valid-token",
		},
		{
			name:   "admin missing csrf publish",
			method: http.MethodPost,
			path:   "/admin/news/news-1/publish",
			user:   &auth.User{Role: auth.RoleAdmin, Active: true},
			token:  "",
		},
		{
			name:   "admin invalid csrf schedule",
			method: http.MethodPost,
			path:   "/admin/news/news-1/schedule",
			user:   &auth.User{Role: auth.RoleAdmin, Active: true},
			token:  "invalid-token",
		},
	}

	for _, tc := range deniedRequests {
		t.Run(tc.name, func(t *testing.T) {
			body := ""
			if tc.token != "" {
				body = "_csrf=" + tc.token + "&title=Attempted+News+Update"
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

	if spy.createCalls != 0 || spy.updateCalls != 0 || spy.scheduleCalls != 0 || spy.publishCalls != 0 || spy.retractCalls != 0 {
		t.Fatalf("denied news mutations invoked storage: %+v", spy)
	}
}

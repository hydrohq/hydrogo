package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitesql "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestNotFoundTrackingAndRedirectConversion(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "missing.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTenantTransactor(db, database.SQLite, false), Store: store}
	registry := &site.RedirectRegistry{}
	handler := Handler{DB: db, Store: store, Service: service, Redirects: registry}
	request := httptest.NewRequest("GET", "https://example.test/missing?private=value", nil)
	custom := site.PublicErrorPage{Headline: "Friendly miss", Message: "The page moved.", ActionLabel: "Homepage", ActionURL: "/"}
	encoded, err := json.Marshal(custom)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SaveSetting(ctx, site.Setting{Key: "site.error_page.404", Value: string(encoded), Public: true}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.redirectFallback(response, request)
	if response.Code != 404 || !strings.Contains(response.Body.String(), "Friendly miss") || strings.Contains(response.Body.String(), "private=value") {
		t.Fatalf("missing path status=%d body=%s", response.Code, response.Body.String())
	}
	items, err := service.ListNotFound(ctx, 10)
	if err != nil || len(items) != 1 || items[0].Path != "/missing" || items[0].Hits != 1 {
		t.Fatalf("not found report=%+v err=%v", items, err)
	}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /admin/redirects/convert" {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	secured := security.CSRFMiddleware(false, nil)(mux)
	convert := func(user *auth.User, token, target string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"source": {"/missing"}, "target": {target}, "_csrf": {token}}
		request := httptest.NewRequest(http.MethodPost, "/admin/redirects/convert", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		request.Header.Set(security.CSRFHeaderName, token)
		request = request.WithContext(auth.WithUser(request.Context(), user))
		response := httptest.NewRecorder()
		secured.ServeHTTP(response, request)
		return response
	}
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	for _, denied := range []struct {
		name  string
		user  *auth.User
		token string
	}{
		{name: "invalid CSRF", user: admin, token: "wrong-token"},
		{name: "Editor", user: &auth.User{ID: "editor", Role: auth.RoleEditor, Active: true}, token: "valid-token"},
		{name: "permission denied", user: &auth.User{ID: "denied", Role: auth.RoleAdmin, Active: true, PermissionOverrides: map[string]bool{auth.PermissionSiteManage: false}}, token: "valid-token"},
	} {
		if deniedResponse := convert(denied.user, denied.token, "/available"); deniedResponse.Code != http.StatusForbidden {
			t.Fatalf("%s convert status=%d body=%s", denied.name, deniedResponse.Code, deniedResponse.Body.String())
		}
		if _, ok := registry.Match(tenant.Scope{}, "/missing"); ok {
			t.Fatalf("%s request registered a redirect", denied.name)
		}
		if remaining, err := service.ListNotFound(ctx, 10); err != nil || len(remaining) != 1 {
			t.Fatalf("%s request changed not-found report: %+v err=%v", denied.name, remaining, err)
		}
	}
	if invalid := convert(admin, "valid-token", ""); invalid.Code != http.StatusBadRequest {
		t.Fatalf("empty target status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	if remaining, err := service.ListNotFound(ctx, 10); err != nil || len(remaining) != 1 {
		t.Fatalf("invalid request changed not-found report: %+v err=%v", remaining, err)
	}
	response = convert(admin, "valid-token", "/available")
	if response.Code != 303 {
		t.Fatalf("conversion status=%d body=%s", response.Code, response.Body.String())
	}
	if got, ok := registry.Match(tenant.Scope{}, "/missing"); !ok || got.TargetPath != "/available" || got.Status != 301 {
		t.Fatalf("registered redirect=%+v ok=%v", got, ok)
	}
	items, err = service.ListNotFound(ctx, 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("converted report=%+v err=%v", items, err)
	}
}

func TestPublic500ErrorPageUsesConfiguredFriendlyCopy(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "custom-error.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTenantTransactor(db, database.SQLite, false), Store: store}
	encoded, err := json.Marshal(site.PublicErrorPage{Headline: "Back soon", Message: "Please try again later.", ActionLabel: "Home", ActionURL: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SaveSetting(ctx, site.Setting{Key: "site.error_page.500", Value: string(encoded), Public: true}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	(Handler{DB: db, Store: store}).renderPublicErrorPage(response, httptest.NewRequest("GET", "/", nil), 500)
	if response.Code != 500 || !strings.Contains(response.Body.String(), "Back soon") || strings.Contains(response.Body.String(), "SQL") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestNotFoundTrackerOmitsPrivateRouteFamilies(t *testing.T) {
	for _, path := range []string{"/admin/login", "/api/secret", "/auth/callback", "/password-reset/token", "/.well-known/acme-challenge/a"} {
		if trackableNotFoundPath(path) {
			t.Errorf("private route %q should not be tracked", path)
		}
	}
	if !trackableNotFoundPath("/old-content") {
		t.Fatal("public content path was not tracked")
	}
}

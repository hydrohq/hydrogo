package transport

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	form := url.Values{"source": {"/missing"}, "target": {"/available"}}
	request = httptest.NewRequest("POST", "/admin/redirects/convert", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response = httptest.NewRecorder()
	handler.convertNotFoundPath(response, request)
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

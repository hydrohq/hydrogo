package transport

import (
	"context"
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
	handler := Handler{DB: db, Service: service, Redirects: registry}
	request := httptest.NewRequest("GET", "https://example.test/missing?private=value", nil)
	response := httptest.NewRecorder()
	handler.redirectFallback(response, request)
	if response.Code != 404 {
		t.Fatalf("missing path status=%d", response.Code)
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

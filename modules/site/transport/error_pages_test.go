package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitesql "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestAdminCanConfigurePublicErrorPages(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "error-pages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := sitesql.Store{Dialect: database.SQLite}
	service := &site.Service{Transactions: database.NewTenantTransactor(db, database.SQLite, false), Store: store}
	handler := Handler{DB: db, Store: store, Service: service}
	request := httptest.NewRequest(http.MethodGet, "/admin/site/error-pages", nil).WithContext(auth.WithUser(ctx, &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	response := httptest.NewRecorder()
	handler.errorPages(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "/admin/site/error-pages/404") || !strings.Contains(response.Body.String(), "/admin/site/error-pages/500") {
		t.Fatalf("editor status=%d body=%s", response.Code, response.Body.String())
	}
	form := url.Values{"headline": {"Page moved"}, "message": {"Try one of these pages."}, "image_url": {"/media/lost.png"}, "action_label": {"Homepage"}, "action_url": {"/"}, "links": {"Contact | /contact"}}
	request = httptest.NewRequest(http.MethodPost, "/admin/site/error-pages/404", strings.NewReader(form.Encode())).WithContext(auth.WithUser(ctx, &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetPathValue("status", "404")
	response = httptest.NewRecorder()
	handler.saveErrorPage(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("save status=%d body=%s", response.Code, response.Body.String())
	}
	settings, err := store.ListPublicSettings(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	page := site.ErrorPageFromSettings(settings, 404)
	if page.Headline != "Page moved" || len(page.Links) != 1 || page.Links[0].URL != "/contact" {
		t.Fatalf("saved page=%+v", page)
	}
	response = httptest.NewRecorder()
	handler.renderPublicErrorPage(response, httptest.NewRequest(http.MethodGet, "/lost", nil), 404)
	if response.Code != 404 || !strings.Contains(response.Body.String(), "Page moved") || !strings.Contains(response.Body.String(), "/media/lost.png") {
		t.Fatalf("public page status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestErrorPageEditorRoutesRegistered(t *testing.T) {
	want := map[string]bool{"GET /admin/site/error-pages": false, "POST /admin/site/error-pages/{status}": false}
	for _, route := range (Handler{}).Routes() {
		if _, ok := want[route.Pattern]; ok {
			want[route.Pattern] = true
		}
	}
	for pattern, found := range want {
		if !found {
			t.Errorf("missing route %s", pattern)
		}
	}
}

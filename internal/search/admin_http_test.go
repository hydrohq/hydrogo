package search

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestAdminSearchReturnsCategorizedContentWithoutSettingsValues(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "admin-search.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE pages (id TEXT PRIMARY KEY, locale TEXT, title TEXT, body TEXT, status TEXT, tenant_id TEXT, deleted_at TEXT)`,
		`CREATE TABLE news_items (id TEXT PRIMARY KEY, locale TEXT, title TEXT, excerpt TEXT, status TEXT, tenant_id TEXT, deleted_at TEXT)`,
		`CREATE TABLE members (id TEXT PRIMARY KEY, email TEXT, active INTEGER, tenant_id TEXT)`,
		`CREATE TABLE media_assets (id TEXT PRIMARY KEY, original_filename TEXT, mime_type TEXT, visibility TEXT, status TEXT, tenant_id TEXT, deleted_at TEXT)`,
		`CREATE TABLE site_settings (key TEXT PRIMARY KEY, value TEXT, tenant_id TEXT)`,
		`INSERT INTO pages(id,locale,title,body,status,tenant_id) VALUES ('p1','tr','HydroGo Rehberi','Go tabanlı içerik','draft','')`,
		`INSERT INTO news_items(id,locale,title,excerpt,status,tenant_id) VALUES ('n1','tr','HydroGo Duyurusu','Platform güncellemesi','published','')`,
		`INSERT INTO members VALUES ('m1','hydro@example.test',1,'')`,
		`INSERT INTO media_assets(id,original_filename,mime_type,visibility,status,tenant_id) VALUES ('a1','hydro-logo.png','image/png','public','ready','')`,
		`INSERT INTO site_settings VALUES ('site.title','PRIVATE SETTING VALUE','')`,
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatalf("setup: %v\n%s", err, statement)
		}
	}
	handler := AdminSearchHandler{DB: db, Dialect: database.SQLite, Modules: map[string]bool{"pages": true, "news": true, "members": true, "media": true, "site": true}}
	if _, err := handler.search(context.Background(), Query{Text: "Hydro", Locale: "tr"}, "", true, true); err != nil {
		t.Fatalf("search backend: %v", err)
	}
	req := httptest.NewRequest("GET", "/admin/api/search?q=Hydro&locale=tr", nil)
	req = req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Items []adminSearchItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	badges := map[string]bool{}
	for _, item := range response.Items {
		badges[item.Badge] = true
	}
	for _, badge := range []string{"Sayfa", "Haber", "Üye", "Medya"} {
		if !badges[badge] {
			t.Errorf("missing %s result in %#v", badge, response.Items)
		}
	}
	if strings.Contains(rec.Body.String(), "PRIVATE SETTING VALUE") {
		t.Fatal("search exposed a setting value")
	}
	settingsRequest := httptest.NewRequest("GET", "/admin/api/search?q=site&locale=tr", nil)
	settingsRequest = settingsRequest.WithContext(auth.WithUser(settingsRequest.Context(), &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}))
	settingsResponse := httptest.NewRecorder()
	handler.ServeHTTP(settingsResponse, settingsRequest)
	if !strings.Contains(settingsResponse.Body.String(), `"badge":"Ayar"`) || strings.Contains(settingsResponse.Body.String(), "PRIVATE SETTING VALUE") {
		t.Fatalf("settings search was not key-only: %s", settingsResponse.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}
}

func TestAdminSearchRanksTitleMatchesAheadOfBodyMatches(t *testing.T) {
	terms := tokenize("hydro studio")
	if got := adminSearchScore("Hydro Studio", "", terms); got != 4 {
		t.Fatalf("title score=%v, want 4", got)
	}
	if got := adminSearchScore("Overview", "Hydro Studio", terms); got != 2 {
		t.Fatalf("body score=%v, want 2", got)
	}
}

func TestAdminSearchRestrictsMembersAndRequiresTenant(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "admin-search-scope.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE members (id TEXT PRIMARY KEY, email TEXT, active INTEGER, tenant_id TEXT); INSERT INTO members VALUES ('m1','hydro@example.test',1,'tenant-a')`); err != nil {
		t.Fatal(err)
	}
	handler := AdminSearchHandler{DB: db, Dialect: database.SQLite, RequireTenantScope: true, Modules: map[string]bool{"members": true}}
	ctx := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", Slug: "tenant-a"})
	if _, err := handler.search(ctx, Query{Text: "hydro"}, "tenant-a", true, true); err != nil {
		t.Fatalf("tenant search backend: %v", err)
	}
	request := func(role auth.Role, scope bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/admin/api/search?q=hydro", nil)
		ctx := auth.WithUser(req.Context(), &auth.User{ID: "user", Role: role, Active: true})
		if scope {
			ctx = tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-a", Slug: "tenant-a"})
		}
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if got := request(auth.RoleAdmin, false).Code; got != 403 {
		t.Fatalf("missing tenant scope status=%d", got)
	}
	if body := request(auth.RoleEditor, true).Body.String(); strings.Contains(body, "hydro@example.test") {
		t.Fatalf("editor saw member result: %s", body)
	}
	if body := request(auth.RoleAdmin, true).Body.String(); !strings.Contains(body, "hydro@example.test") {
		t.Fatalf("admin missing member result: %s", body)
	}
}

package search

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestPublicAutocompleteReturnsOnlyPublishedTenantContent(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "autocomplete.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{`CREATE TABLE pages(id TEXT,locale TEXT,title TEXT,slug TEXT,status TEXT,tenant_id TEXT)`, `CREATE TABLE news_items(id TEXT,locale TEXT,title TEXT,slug TEXT,status TEXT,tenant_id TEXT)`, `INSERT INTO pages VALUES('pub','tr','Hydro Public','hydro-public','published','a')`, `INSERT INTO pages VALUES('draft','tr','Hydro Secret','hydro-secret','draft','a')`, `INSERT INTO pages VALUES('foreign','tr','Hydro Foreign','hydro-foreign','published','b')`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	handler := AutocompleteHandler{DB: db, Dialect: database.SQLite, RequireTenantScope: true, Modules: map[string]bool{"pages": true}}
	req := httptest.NewRequest("GET", "/?q=Hydro&locale=tr", nil).WithContext(tenant.WithScope(context.Background(), tenant.Scope{TenantID: "a", Slug: "a"}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Hydro Public") || strings.Contains(rec.Body.String(), "Hydro Secret") || strings.Contains(rec.Body.String(), "Hydro Foreign") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAutocompleteBoundsQuery(t *testing.T) {
	h := AutocompleteHandler{}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/?q=x", nil))
	if rec.Code != 400 {
		t.Fatalf("short query status=%d", rec.Code)
	}
}

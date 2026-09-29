package search

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
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

type searchRecorderStub struct{ events []analytics.SearchQueryEvent }

func (s *searchRecorderStub) Enqueue(event analytics.SearchQueryEvent) bool {
	s.events = append(s.events, event)
	return true
}

func TestPublicSearchRecordsOnlyNormalizedAggregate(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "public-search.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{`CREATE TABLE pages(id TEXT,locale TEXT,title TEXT,slug TEXT,status TEXT,tenant_id TEXT)`, `CREATE TABLE news_items(id TEXT,locale TEXT,title TEXT,slug TEXT,status TEXT,tenant_id TEXT)`, `INSERT INTO pages VALUES('pub','tr','Hydro Public','hydro-public','published','a')`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	recorder := &searchRecorderStub{}
	h := PublicSearchHandler{Autocomplete: AutocompleteHandler{DB: db, Dialect: database.SQLite, RequireTenantScope: true, Modules: map[string]bool{"pages": true}}, Recorder: recorder, Now: func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC) }}
	req := httptest.NewRequest("GET", "/search?q=%20HYDRO%20&locale=tr", nil).WithContext(tenant.WithScope(context.Background(), tenant.Scope{TenantID: "a", Slug: "a"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Hydro Public") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(recorder.events) != 1 || recorder.events[0].Term != "hydro" || recorder.events[0].Locale != "tr" || recorder.events[0].TenantID != "a" || recorder.events[0].ResultCount != 1 || recorder.events[0].Day != "2026-09-29" {
		t.Fatalf("events=%+v", recorder.events)
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

package search

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func newSQLSearchTestProvider(t *testing.T, max int) SQLProvider {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "search.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE hydro_search_documents (id TEXT PRIMARY KEY, locale TEXT NOT NULL, title TEXT NOT NULL, body TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return SQLProvider{Executor: db, Dialect: database.SQLite, Max: max}
}

func TestSQLProviderRebuildSearchAndDelete(t *testing.T) {
	provider := newSQLSearchTestProvider(t, 3)
	ctx := context.Background()
	if err := provider.Rebuild(ctx, []Document{{ID: "a", Locale: "tr", Title: "Go Haber", Body: "Platform"}, {ID: "b", Locale: "tr", Title: "HydroGo", Body: "Go performansı"}}); err != nil {
		t.Fatal(err)
	}
	response, err := provider.Search(ctx, Query{Text: "go", Locale: "tr", Limit: 1})
	if err != nil || response.Total != 2 || len(response.Items) != 1 || response.Items[0].ID != "b" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	if err := provider.Delete(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	response, err = provider.Search(ctx, Query{Text: "go", Locale: "tr"})
	if err != nil || response.Total != 1 || response.Items[0].ID != "a" {
		t.Fatalf("after delete=%#v err=%v", response, err)
	}
}

func TestSQLProviderBoundsAndValidation(t *testing.T) {
	provider := newSQLSearchTestProvider(t, 1)
	ctx := context.Background()
	if err := provider.Upsert(ctx, Document{ID: "one", Locale: "en", Title: "One"}); err != nil {
		t.Fatal(err)
	}
	if err := provider.Upsert(ctx, Document{ID: "two", Locale: "en", Title: "Two"}); !errors.Is(err, ErrSQLIndexFull) {
		t.Fatalf("full error=%v", err)
	}
	if err := provider.Rebuild(ctx, []Document{{ID: "one", Locale: "en", Title: "One"}, {ID: "one", Locale: "en", Title: "Duplicate"}}); err == nil {
		t.Fatal("duplicate rebuild accepted")
	}
}

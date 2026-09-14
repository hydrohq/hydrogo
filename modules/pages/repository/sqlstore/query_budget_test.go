package sqlstore

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

type countingQueryDB struct {
	*sql.DB
	queries int
}

func (db *countingQueryDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	db.queries++
	return db.DB.QueryContext(ctx, query, args...)
}

func TestListPublishedUsesOneBoundedQueryWithoutNPlusOne(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "pages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pages
		(id, title, locale, slug, body, status, published_locale, published_slug, published_title, published_revision, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'published', ?, ?, ?, 1, ?, ?)`,
		"page-1", "Ana sayfa", "tr", "ana-sayfa", "{}", "tr", "ana-sayfa", "Ana sayfa", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	queryDB := &countingQueryDB{DB: db}
	items, err := (Store{Dialect: database.SQLite}).ListPublished(ctx, queryDB, "tr", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "page-1" {
		t.Fatalf("list result = %#v, want one page", items)
	}
	if queryDB.queries != 1 {
		t.Fatalf("published list queries = %d, want exactly 1 bounded query", queryDB.queries)
	}
}

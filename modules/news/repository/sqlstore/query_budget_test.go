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

func TestListAdminUsesOneBoundedQueryWithoutNPlusOne(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO news_items
		(id, locale, slug, title, body, status, published_revision, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'published', 1, ?, ?)`,
		"news-1", "tr", "tek-haber", "Tek haber", "{}", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}

	queryDB := &countingQueryDB{DB: db}
	items, err := (Store{Dialect: database.SQLite}).ListAdmin(ctx, queryDB, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "news-1" {
		t.Fatalf("list result = %#v, want one news item", items)
	}
	if queryDB.queries != 1 {
		t.Fatalf("admin list queries = %d, want exactly 1 bounded query", queryDB.queries)
	}
}

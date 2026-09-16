package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestReplaceAndListTaxonomySQLite(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "taxonomy.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at) VALUES (?,?,?,?,?,'draft',NULL,?,?)`, "article-1", "tr", "article-1", "Article", "{}", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	terms := []v1.TaxonomyTerm{{ID: "news", Slug: "news", Kind: "category", Labels: map[string]string{"en": "News"}}, {ID: "release", Slug: "release", Kind: "tag", Parent: "news", Labels: map[string]string{"tr": "Sürüm"}}}
	store := Store{Dialect: database.SQLite}
	if err := store.ReplaceTaxonomy(ctx, db, "article-1", terms); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListTaxonomy(ctx, db, "article-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "news" || got[1].Parent != "news" {
		t.Fatalf("taxonomy = %#v", got)
	}
	if err := store.ReplaceTaxonomy(ctx, db, "article-1", nil); err != nil {
		t.Fatal(err)
	}
	got, err = store.ListTaxonomy(ctx, db, "article-1")
	if err != nil || len(got) != 0 {
		t.Fatalf("expected cleared taxonomy, got %#v err=%v", got, err)
	}
}

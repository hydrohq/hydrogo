package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
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

func TestTaxonomyTermCRUDGuardsChildrenAndRelations(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "taxonomy-crud.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	root := v1.TaxonomyTerm{ID: "platform", Slug: "platform", Kind: "category", Labels: map[string]string{"en": "Platform"}}
	child := v1.TaxonomyTerm{ID: "go", Slug: "go", Kind: "tag", Parent: "platform", Labels: map[string]string{"en": "Go"}}
	if err := store.UpsertTaxonomyTerm(ctx, db, root); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTaxonomyTerm(ctx, db, child); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTaxonomyTerm(ctx, db, "platform"); !errors.Is(err, ErrTaxonomyTermHasChildren) {
		t.Fatalf("delete parent error = %v, want ErrTaxonomyTermHasChildren", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at) VALUES (?,?,?,?,?,'draft',NULL,?,?)`, "article-2", "tr", "article-2", "Article", "{}", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, "article-2", []v1.TaxonomyTerm{child}); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTaxonomyTerm(ctx, db, "go"); !errors.Is(err, ErrTaxonomyTermInUse) {
		t.Fatalf("delete related term error = %v, want ErrTaxonomyTermInUse", err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, "article-2", nil); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTaxonomyTerm(ctx, db, "go"); err != nil {
		t.Fatalf("delete unused leaf: %v", err)
	}
}

func TestListRelatedPublishedUsesSharedTermsAndExcludesDrafts(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "taxonomy-related.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	term := v1.TaxonomyTerm{ID: "go", Slug: "go", Kind: "tag", Labels: map[string]string{"en": "Go"}}
	termHydro := v1.TaxonomyTerm{ID: "hydro", Slug: "hydro", Kind: "tag", Labels: map[string]string{"en": "Hydro"}}
	for _, value := range []v1.TaxonomyTerm{term, termHydro} {
		if err := store.UpsertTaxonomyTerm(ctx, db, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		id, status, excerpt, cover string
		published                  any
	}{
		{"source", "published", "", "", 1},
		{"related", "published", "excerpt for a two minute read", "cover-1", 1},
		{"best", "published", "", "cover-2", 1},
		{"draft", "draft", "", "", nil},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,excerpt,cover_media_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, item.id, "en", item.id, item.id, "{}", item.status, item.published, item.excerpt, item.cover, "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.ReplaceTaxonomy(ctx, db, "source", []v1.TaxonomyTerm{term, termHydro}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, "related", []v1.TaxonomyTerm{term}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, "best", []v1.TaxonomyTerm{term, termHydro}); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, "draft", []v1.TaxonomyTerm{term}); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListRelatedPublished(ctx, db, "source", "en", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "best" || items[1].ID != "related" || items[1].Status != news.StatusPublished || items[1].Excerpt != "excerpt for a two minute read" || items[1].CoverMediaID != "cover-1" {
		t.Fatalf("related items = %#v", items)
	}
}

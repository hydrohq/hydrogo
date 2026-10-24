package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/testpostgres"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

// news_taxonomy_terms/news_taxonomy_relations had no tenant_id at all until
// 202610241800_news_taxonomy_tenant_scope: an authenticated admin in one
// tenant could read, overwrite or delete another tenant's taxonomy terms
// outright, in a shared-database multi-tenant deployment. This contract
// proves the fix on every dialect a developer can reach.
func TestSQLiteTaxonomyTenantScopedStoreContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "taxonomy-tenant.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runTaxonomyTenantScopedContract(t, db, database.SQLite)
}

func TestPostgresTaxonomyTenantScopedStoreContract(t *testing.T) {
	db := testpostgres.Open(t)
	if err := database.ApplyMigrations(context.Background(), db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runTaxonomyTenantScopedContract(t, db, database.Postgres)
}

func newsTaxonomyTenantCtx(t *testing.T, id string) context.Context {
	t.Helper()
	scope, err := tenant.NewScope(id, id)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithScope(context.Background(), scope)
}

func runTaxonomyTenantScopedContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	store := Store{Dialect: dialect, RequireTenantScope: true}
	a := newsTaxonomyTenantCtx(t, "tenant-a")
	b := newsTaxonomyTenantCtx(t, "tenant-b")

	termA := v1.TaxonomyTerm{ID: "shared-id", Slug: "a-category", Kind: "category", Labels: map[string]string{"en": "A"}}
	if err := store.UpsertTaxonomyTerm(a, db, termA); err != nil {
		t.Fatalf("create tenant a's term: %v", err)
	}

	t.Run("a term is invisible to another tenant's catalog", func(t *testing.T) {
		terms, err := store.ListTaxonomyTerms(b, db)
		if err != nil {
			t.Fatal(err)
		}
		for _, term := range terms {
			if term.ID == termA.ID {
				t.Fatalf("tenant b can see tenant a's term: %#v", term)
			}
		}
	})

	t.Run("another tenant cannot reuse the same id to overwrite it", func(t *testing.T) {
		attempt := v1.TaxonomyTerm{ID: "shared-id", Slug: "b-hijack", Kind: "category", Labels: map[string]string{"en": "B"}}
		err := store.UpsertTaxonomyTerm(b, db, attempt)
		if !errors.Is(err, news.ErrTaxonomyTermIDTaken) {
			t.Fatalf("upsert with another tenant's id = %v, want ErrTaxonomyTermIDTaken", err)
		}
		terms, err := store.ListTaxonomyTerms(a, db)
		if err != nil {
			t.Fatal(err)
		}
		if len(terms) != 1 || terms[0].Slug != "a-category" {
			t.Fatalf("tenant a's term was modified by tenant b's attempt: %#v", terms)
		}
	})

	t.Run("another tenant cannot delete it", func(t *testing.T) {
		err := store.DeleteTaxonomyTerm(b, db, termA.ID)
		if err == nil {
			t.Fatal("tenant b deleted tenant a's term without error")
		}
		terms, err := store.ListTaxonomyTerms(a, db)
		if err != nil {
			t.Fatal(err)
		}
		if len(terms) != 1 {
			t.Fatalf("tenant a's term was deleted by tenant b: %#v", terms)
		}
	})

	t.Run("relations and ListTaxonomy stay inside their tenant", func(t *testing.T) {
		scopedInsert := `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at,tenant_id) VALUES (?,?,?,?,?,'draft',NULL,?,?,?)`
		if dialect == database.Postgres {
			scopedInsert = `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at,tenant_id) VALUES ($1,$2,$3,$4,$5,'draft',NULL,$6,$7,$8)`
		}
		if _, err := db.ExecContext(a, scopedInsert, "article-a", "en", "article-a", "Article", "{}", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z", "tenant-a"); err != nil {
			t.Fatal(err)
		}
		termB := v1.TaxonomyTerm{ID: "term-b", Slug: "b-category", Kind: "category", Labels: map[string]string{"en": "B"}}
		if err := store.UpsertTaxonomyTerm(b, db, termB); err != nil {
			t.Fatal(err)
		}
		if err := store.ReplaceTaxonomy(a, db, "article-a", []v1.TaxonomyTerm{termA}); err != nil {
			t.Fatal(err)
		}
		// Tenant b must not be able to attach tenant a's article to its own term,
		// nor read tenant a's attached terms back out.
		got, err := store.ListTaxonomy(b, db, "article-a")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Fatalf("tenant b can read tenant a's article taxonomy: %#v", got)
		}
		gotA, err := store.ListTaxonomy(a, db, "article-a")
		if err != nil {
			t.Fatal(err)
		}
		if len(gotA) != 1 || gotA[0].ID != termA.ID {
			t.Fatalf("tenant a's own article taxonomy = %#v", gotA)
		}
	})
}

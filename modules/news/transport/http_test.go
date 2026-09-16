package transport

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type taxonomyTransactionProbe struct{}

func (taxonomyTransactionProbe) ListTaxonomy(context.Context, database.Executor, string) ([]v1.TaxonomyTerm, error) {
	return nil, nil
}

func (taxonomyTransactionProbe) ListTaxonomyTerms(context.Context, database.Executor) ([]v1.TaxonomyTerm, error) {
	return nil, nil
}

func (taxonomyTransactionProbe) ReplaceTaxonomy(ctx context.Context, executor database.Executor, _ string, _ []v1.TaxonomyTerm) error {
	if _, err := executor.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS taxonomy_transaction_probe (id INTEGER PRIMARY KEY)`); err != nil {
		return err
	}
	_, err := executor.ExecContext(ctx, `INSERT INTO taxonomy_transaction_probe (id) VALUES (1)`)
	return err
}

func (taxonomyTransactionProbe) UpsertTaxonomyTerm(context.Context, database.Executor, v1.TaxonomyTerm) error {
	return nil
}

func (taxonomyTransactionProbe) DeleteTaxonomyTerm(context.Context, database.Executor, string) error {
	return nil
}

func TestReplaceTaxonomyAtomicallyUsesDatabaseTransaction(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "taxonomy-transaction.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := replaceTaxonomyAtomically(ctx, db, taxonomyTransactionProbe{}, "article-1", nil); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM taxonomy_transaction_probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("transaction probe count = %d, want 1", count)
	}
}

func TestRoutesIncludeCSRFProtectedAdminNewsForms(t *testing.T) {
	routes := (Handler{}).Routes()
	want := map[string]bool{
		"GET /admin/news/{id}":                  true,
		"POST /admin/news/{id}":                 true,
		"GET /admin/news/taxonomy":              true,
		"POST /admin/news/taxonomy":             true,
		"POST /admin/news/taxonomy/{id}/delete": true,
		"POST /admin/news/{id}/taxonomy":        true,
		"POST /admin/news/{id}/schedule":        true,
		"POST /admin/news/{id}/publish":         true,
		"POST /admin/news/{id}/retract":         true,
	}
	for _, route := range routes {
		if _, ok := want[route.Pattern]; ok {
			if !route.Protected {
				t.Errorf("route %q is not protected", route.Pattern)
			}
			delete(want, route.Pattern)
		}
	}
	for pattern := range want {
		t.Errorf("missing admin route %q", pattern)
	}
}

func TestRoutesIncludePublicPublishedNewsList(t *testing.T) {
	foundList, foundBySlug := false, false
	for _, route := range (Handler{}).Routes() {
		if route.Pattern == "GET /api/news" {
			if route.Protected {
				t.Fatal("published news list must be public")
			}
			foundList = true
		}
		if route.Pattern == "GET /api/news/by-slug" {
			if route.Protected {
				t.Fatal("published news lookup must be public")
			}
			foundBySlug = true
		}
	}
	if !foundList || !foundBySlug {
		t.Fatalf("missing public news routes list=%t by_slug=%t", foundList, foundBySlug)
	}
}

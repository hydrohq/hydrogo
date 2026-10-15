package contract_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
)

func TestSQLiteNewsTaxonomyContract(t *testing.T) {
	runNewsTaxonomyContract(t, database.SQLite, filepath.Join(t.TempDir(), "taxonomy.sqlite"))
}

func TestPostgresNewsTaxonomyContract(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runNewsTaxonomyContract(t, database.Postgres, os.Getenv("HYDRO_TEST_POSTGRES_URL"))
}

func runNewsTaxonomyContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	marker := time.Now().UTC().UnixNano()
	id := fmt.Sprintf("taxonomy-news-%d", marker)
	insertQuery := `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at) VALUES (?,?,?,?,?,'draft',NULL,?,?)`
	if dialect == database.Postgres {
		insertQuery = `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,'draft',NULL,$6,$7)`
	}
	if _, err := db.ExecContext(ctx, insertQuery, id, "en", id, "Taxonomy", "{}", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: dialect}
	termID := fmt.Sprintf("contract-news-%d", marker)
	terms := []v1.TaxonomyTerm{{ID: termID, Slug: termID, Kind: "category", Labels: map[string]string{"en": "Contract News"}}}
	if err := store.ReplaceTaxonomy(ctx, db, id, terms); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertTaxonomyTerm(ctx, db, v1.TaxonomyTerm{ID: termID, Slug: termID, Kind: "category", Labels: map[string]string{"en": "Updated Contract News"}}); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListTaxonomy(ctx, db, id)
	if err != nil || len(got) != 1 || got[0].ID != terms[0].ID {
		t.Fatalf("taxonomy=%#v err=%v", got, err)
	}
	relatedID := fmt.Sprintf("taxonomy-related-%d", marker)
	relatedInsert := `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at) VALUES (?,?,?,?,?,'published',1,?,?)`
	if dialect == database.Postgres {
		relatedInsert = `INSERT INTO news_items (id,locale,slug,title,body,status,published_revision,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,'published',1,$6,$7)`
	}
	if _, err := db.ExecContext(ctx, relatedInsert, relatedID, "en", relatedID, "Related", "{}", "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, relatedID, terms); err != nil {
		t.Fatal(err)
	}
	related, err := store.ListRelatedPublished(ctx, db, id, "en", 10)
	if err != nil || len(related) != 1 || related[0].ID != relatedID {
		t.Fatalf("related=%#v err=%v", related, err)
	}
	if err := store.ReplaceTaxonomy(ctx, db, id, nil); err != nil {
		t.Fatal(err)
	}
	got, err = store.ListTaxonomy(ctx, db, id)
	if err != nil || len(got) != 0 {
		t.Fatalf("taxonomy clear=%#v err=%v", got, err)
	}
}

package contract_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteCollectionSchemaContract(t *testing.T) {
	runCollectionContract(t, database.SQLite, filepath.Join(t.TempDir(), "collections.sqlite"))
}

func TestPostgresCollectionSchemaContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runCollectionContract(t, database.Postgres, dsn)
}

func runCollectionContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrationsForModules(ctx, db, dialect, migrations.FS, map[string]bool{"collections": true}); err != nil {
		t.Fatal(err)
	}
	marker := fmt.Sprintf("contract_%d", time.Now().UnixNano())
	store := collectionstore.Store{DB: db, Dialect: dialect}
	service := collections.Service{Repository: store}
	schema := collections.Schema{ID: marker, Label: "Contract", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true, Unique: true}}}
	if err := service.CreateSchema(ctx, schema); err != nil {
		t.Fatal(err)
	}
	first, err := service.CreateRecord(ctx, marker, map[string]any{"name": "one"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecord(ctx, marker, map[string]any{"name": "one"}); err != collections.ErrUniqueValue {
		t.Fatalf("duplicate unique value error = %v", err)
	}
	updated, err := service.UpdateRecord(ctx, marker, first.ID, map[string]any{"name": "two"})
	if err != nil || updated.Version != first.Version+1 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	if err := service.DeleteRecord(ctx, marker, first.ID); err != nil {
		t.Fatal(err)
	}
}

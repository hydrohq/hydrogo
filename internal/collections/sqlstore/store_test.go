package sqlstore

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestStaticStoreSupportsSchemasUniqueValuesRelationsAndCRUD(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "collections.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runStaticStoreContract(t, db, database.SQLite)
	runTenantIsolationContract(t, db, database.SQLite)
	runQueryContract(t, db, database.SQLite)
	runRevisionContract(t, db, database.SQLite)
	runTranslationContract(t, db, database.SQLite)
}

func TestPostgresStaticStoreSupportsSchemasUniqueValuesRelationsAndCRUD(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	adminDB, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schemaName := "collections_contract_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		adminDB.Close()
		t.Fatal(err)
	}
	defer func() {
		_, _ = adminDB.ExecContext(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		_ = adminDB.Close()
	}()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schemaName)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.Postgres, migrations.FS, map[string]bool{"collections": true}); err != nil {
		t.Fatal(err)
	}
	runStaticStoreContract(t, db, database.Postgres)
	runTenantIsolationContract(t, db, database.Postgres)
	runConcurrentUniqueContract(t, db)
	runQueryContract(t, db, database.Postgres)
	runRevisionContract(t, db, database.Postgres)
	runTranslationContract(t, db, database.Postgres)
	runFilterContract(t, db, database.Postgres)
	runIndexContract(t, db)
}

func runStaticStoreContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	topics := collections.Schema{ID: "topics", Label: "Topics", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true, Unique: true}}}
	if err := service.CreateSchema(ctx, topics); err != nil {
		t.Fatal(err)
	}
	topic, err := service.CreateRecord(ctx, topics.ID, map[string]any{"name": "Hydro"})
	if err != nil {
		t.Fatal(err)
	}
	articles := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "slug", Label: "Slug", Kind: collections.Text, Required: true, Unique: true}, {ID: "topic", Label: "Topic", Kind: collections.Relation, Relation: topics.ID}}}
	if err := service.CreateSchema(ctx, articles); err != nil {
		t.Fatal(err)
	}
	article, err := service.CreateRecord(ctx, articles.ID, map[string]any{"slug": "one", "topic": topic.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecord(ctx, articles.ID, map[string]any{"slug": "one", "topic": topic.ID}); !errors.Is(err, collections.ErrUniqueValue) {
		t.Fatalf("duplicate unique value error=%v", err)
	}
	list, err := service.ListRecords(ctx, articles.ID, 10, 0)
	if err != nil || len(list) != 1 || list[0].ID != article.ID {
		t.Fatalf("failed unique write left a partial record: list=%+v err=%v", list, err)
	}
	if err := service.DeleteRecord(ctx, topics.ID, topic.ID); err == nil {
		t.Fatal("target record with a relation was deleted")
	}
	updated, err := service.UpdateRecord(ctx, articles.ID, article.ID, map[string]any{"slug": "two", "topic": topic.ID})
	if err != nil || updated.Version != 2 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	if _, err := service.UpdateRecordVersion(ctx, articles.ID, article.ID, map[string]any{"slug": "stale", "topic": topic.ID}, 1); !errors.Is(err, collections.ErrVersionConflict) {
		t.Fatalf("stale version update error=%v, want ErrVersionConflict", err)
	}
	if current, err := service.UpdateRecordVersion(ctx, articles.ID, article.ID, map[string]any{"slug": "two", "topic": topic.ID}, 2); err != nil || current.Version != 3 {
		t.Fatalf("matching version update=%+v err=%v", current, err)
	}
	list, err = service.ListRecords(ctx, articles.ID, 10, 0)
	if err != nil || len(list) != 1 || list[0].ID != article.ID {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	if err := service.DeleteRecord(ctx, articles.ID, article.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteRecord(ctx, topics.ID, topic.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteSchema(ctx, topics.ID); err == nil {
		t.Fatal("schema with incoming relation was deleted")
	}
	if err := service.DeleteSchema(ctx, articles.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteSchema(ctx, topics.ID); err != nil {
		t.Fatal(err)
	}
}

func runConcurrentUniqueContract(t *testing.T, db database.Executor) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: database.Postgres}}
	schema := collections.Schema{ID: "unique_race", Label: "Unique race", Fields: []collections.Field{{ID: "key", Label: "Key", Kind: collections.Text, Required: true, Unique: true}}}
	if err := service.CreateSchema(ctx, schema); err != nil {
		t.Fatal(err)
	}
	const writers = 8
	start := make(chan struct{})
	results := make(chan error, writers)
	var workers sync.WaitGroup
	for range writers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, err := service.CreateRecord(ctx, schema.ID, map[string]any{"key": "same-value"})
			results <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, collections.ErrUniqueValue):
			conflicts++
		default:
			t.Fatalf("concurrent unique write error = %v", err)
		}
	}
	if successes != 1 || conflicts != writers-1 {
		t.Fatalf("concurrent unique writes: successes=%d conflicts=%d, want 1/%d", successes, conflicts, writers-1)
	}
	records, err := service.ListRecords(ctx, schema.ID, 10, 0)
	if err != nil || len(records) != 1 {
		t.Fatalf("concurrent unique writes left records=%d err=%v", len(records), err)
	}
	if err := service.DeleteSchema(ctx, schema.ID); err != nil {
		t.Fatalf("delete concurrency schema: %v", err)
	}
}

func runTenantIsolationContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	alpha, err := tenant.NewScope("tenant-alpha", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("tenant-beta", "beta")
	if err != nil {
		t.Fatal(err)
	}
	alphaCtx := tenant.WithScope(ctx, alpha)
	betaCtx := tenant.WithScope(ctx, beta)
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect, RequireTenantScope: true}}
	schema := collections.Schema{ID: "tenant_catalog", Label: "Catalog", Fields: []collections.Field{{ID: "title", Label: "Title", Kind: collections.Text, Required: true, Unique: true}}}
	if err := service.CreateSchema(alphaCtx, schema); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateSchema(betaCtx, schema); err != nil {
		t.Fatal(err)
	}
	record, err := service.CreateRecord(alphaCtx, schema.ID, map[string]any{"title": "Alpha item"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRecord(betaCtx, schema.ID, record.ID); !errors.Is(err, collections.ErrRecordNotFound) {
		t.Fatalf("cross-tenant get error = %v, want not found", err)
	}
	betaRecords, err := service.ListRecords(betaCtx, schema.ID, 10, 0)
	if err != nil || len(betaRecords) != 0 {
		t.Fatalf("cross-tenant list = %+v, %v", betaRecords, err)
	}
	if _, err := service.CreateRecord(betaCtx, schema.ID, map[string]any{"title": "Alpha item"}); err != nil {
		t.Fatalf("unique value should be scoped to tenant: %v", err)
	}
	if err := service.DeleteSchema(alphaCtx, schema.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteSchema(betaCtx, schema.ID); err != nil {
		t.Fatal(err)
	}
}

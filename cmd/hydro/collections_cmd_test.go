package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/generator"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

type fakeBlueprintCatalog struct {
	schemas []collections.Schema
	created []collections.Schema
	failID  string
}

func (c *fakeBlueprintCatalog) ListSchemas(context.Context) ([]collections.Schema, error) {
	return append([]collections.Schema(nil), c.schemas...), nil
}

func (c *fakeBlueprintCatalog) CreateSchema(_ context.Context, schema collections.Schema) error {
	if schema.ID == c.failID {
		return errors.New("write failed")
	}
	c.schemas = append(c.schemas, schema)
	c.created = append(c.created, schema)
	return nil
}

func TestBlueprintCollectionSchemaMapsSupportedFields(t *testing.T) {
	minimum := 1.0
	spec := generator.Spec{
		Name: "articles", Type: "collection", Table: "articles",
		Permissions: []string{"articles.read", "articles.write"},
		Admin:       generator.Admin{Label: "Articles", Description: "Editorial content"},
		Fields: []generator.Field{
			{Name: "title", Type: "string", Admin: generator.FieldAdmin{Label: "Title"}},
			{Name: "state", Type: "enum", Admin: generator.FieldAdmin{Options: []string{"draft", "published"}}},
			{Name: "views", Type: "number", Validation: generator.Validation{Min: &minimum}},
		},
	}
	schema, err := blueprintCollectionSchema(spec)
	if err != nil {
		t.Fatal(err)
	}
	if schema.ID != "articles" || schema.Label != "Articles" || len(schema.Fields) != 3 {
		t.Fatalf("unexpected schema: %#v", schema)
	}
	if schema.Fields[0].Kind != collections.Text || !schema.Fields[0].Required {
		t.Fatalf("unexpected title field: %#v", schema.Fields[0])
	}
	if schema.Fields[1].Kind != collections.Select || len(schema.Fields[1].Options) != 2 {
		t.Fatalf("unexpected state field: %#v", schema.Fields[1])
	}
	if schema.Fields[2].Kind != collections.Number || schema.Fields[2].Min == nil || *schema.Fields[2].Min != minimum {
		t.Fatalf("unexpected views field: %#v", schema.Fields[2])
	}
}

func TestBlueprintCollectionSchemaRejectsUnsupportedRuntimeFeatures(t *testing.T) {
	tests := []struct {
		name string
		spec generator.Spec
	}{
		{name: "localized", spec: generator.Spec{Name: "articles", Type: "collection", Fields: []generator.Field{{Name: "title", Type: "string", Admin: generator.FieldAdmin{Localized: true}}}}},
		{name: "workflow", spec: generator.Spec{Name: "articles", Type: "collection", Workflow: &generator.WorkflowSpec{}, Fields: []generator.Field{{Name: "title", Type: "string"}}}},
		{name: "unsupported type", spec: generator.Spec{Name: "articles", Type: "collection", Fields: []generator.Field{{Name: "body", Type: "json"}}}},
		{name: "integer precision", spec: generator.Spec{Name: "articles", Type: "collection", Fields: []generator.Field{{Name: "count", Type: "integer"}}}},
		{name: "select without options", spec: generator.Spec{Name: "articles", Type: "collection", Fields: []generator.Field{{Name: "state", Type: "select"}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := blueprintCollectionSchema(test.spec); err == nil {
				t.Fatal("expected unsupported Blueprint to be rejected")
			}
		})
	}
}

func TestCollectionsSyncRequiresAdminCRUDAcknowledgment(t *testing.T) {
	if err := collectionsCommand([]string{"sync"}, io.Discard, io.Discard); err == nil {
		t.Fatal("expected explicit Admin CRUD acknowledgment to be required")
	}
	if !hasGeneratedCRUDPermissions("article", []string{"article.read", "article.write"}) {
		t.Fatal("read/write permission pair should represent generated CRUD permission contract")
	}
	if hasGeneratedCRUDPermissions("article", []string{"article.read"}) {
		t.Fatal("read-only resource must not be enabled for Admin CRUD")
	}
	if hasGeneratedCRUDPermissions("article", []string{"other.read", "other.write"}) {
		t.Fatal("permissions from another resource must not enable Admin CRUD")
	}
}

func TestSyncBlueprintCollectionSchemasCreatesDependenciesAndIsIdempotent(t *testing.T) {
	catalog := &fakeBlueprintCatalog{}
	category := collections.Schema{ID: "categories", Label: "Categories", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text}}}
	article := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "category", Label: "Category", Kind: collections.Relation, Relation: "categories"}}}
	created, unchanged, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{article, category})
	if err != nil {
		t.Fatal(err)
	}
	if created != 2 || unchanged != 0 || len(catalog.created) != 2 || catalog.created[0].ID != "categories" {
		t.Fatalf("unexpected sync order/result: created=%d unchanged=%d schemas=%#v", created, unchanged, catalog.created)
	}
	created, unchanged, err = syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{category, article})
	if err != nil || created != 0 || unchanged != 2 {
		t.Fatalf("repeat sync: created=%d unchanged=%d err=%v", created, unchanged, err)
	}
}

func TestSyncBlueprintCollectionSchemasRejectsDriftAndInvalidRelationsBeforeWrites(t *testing.T) {
	existing := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "title", Label: "Title", Kind: collections.Text}}}
	changed := existing
	changed.Label = "News"
	catalog := &fakeBlueprintCatalog{schemas: []collections.Schema{existing}}
	if _, _, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{changed}); err == nil {
		t.Fatal("expected schema drift to be rejected")
	}
	if len(catalog.created) != 0 {
		t.Fatal("schema drift must not write")
	}

	catalog = &fakeBlueprintCatalog{}
	missingTarget := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "category", Label: "Category", Kind: collections.Relation, Relation: "categories"}}}
	if _, _, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{missingTarget}); err == nil {
		t.Fatal("expected missing relation target to be rejected")
	}
	cycleA := collections.Schema{ID: "alpha", Label: "Alpha", Fields: []collections.Field{{ID: "beta", Label: "Beta", Kind: collections.Relation, Relation: "bravo"}}}
	cycleB := collections.Schema{ID: "bravo", Label: "Bravo", Fields: []collections.Field{{ID: "alpha", Label: "Alpha", Kind: collections.Relation, Relation: "alpha"}}}
	if _, _, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{cycleA, cycleB}); err == nil {
		t.Fatal("expected relation cycle to be rejected")
	}
	if len(catalog.created) != 0 {
		t.Fatal("invalid relation graphs must not write")
	}
}

func TestSyncBlueprintCollectionSchemasCanResumeAfterDatabaseFailure(t *testing.T) {
	catalog := &fakeBlueprintCatalog{failID: "articles"}
	target := collections.Schema{ID: "categories", Label: "Categories", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text}}}
	source := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "category", Label: "Category", Kind: collections.Relation, Relation: "categories"}}}
	created, _, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{source, target})
	if err == nil || created != 1 || len(catalog.schemas) != 1 || catalog.schemas[0].ID != "categories" {
		t.Fatalf("expected partial target-first insert: created=%d schemas=%#v err=%v", created, catalog.schemas, err)
	}
	catalog.failID = ""
	created, unchanged, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{source, target})
	if err != nil || created != 1 || unchanged != 1 {
		t.Fatalf("resume: created=%d unchanged=%d err=%v", created, unchanged, err)
	}
}

func TestSyncBlueprintCollectionSchemasSQLiteContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "collections.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "202609290002_dynamic_collections", "sqlite.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	catalog := collections.Service{Repository: collectionstore.Store{DB: db, Dialect: database.SQLite}}
	target := collections.Schema{ID: "categories", Label: "Categories", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true}}}
	source := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "category", Label: "Category", Kind: collections.Relation, Relation: "categories"}}}
	created, unchanged, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{source, target})
	if err != nil || created != 2 || unchanged != 0 {
		t.Fatalf("first sync: created=%d unchanged=%d err=%v", created, unchanged, err)
	}
	created, unchanged, err = syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{source, target})
	if err != nil || created != 0 || unchanged != 2 {
		t.Fatalf("repeat sync: created=%d unchanged=%d err=%v", created, unchanged, err)
	}
	changed := target
	changed.Label = "Changed"
	if _, _, err := syncBlueprintCollectionSchemas(context.Background(), catalog, []collections.Schema{changed}); err == nil {
		t.Fatal("expected persisted schema drift to be rejected")
	}
	stored, err := catalog.ListSchemas(context.Background())
	if err != nil || len(stored) != 2 {
		t.Fatalf("stored schemas: %#v err=%v", stored, err)
	}
}

func TestCollectionsCommandSyncSQLiteEndToEnd(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "edition.sqlite")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"collections": true}); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	spec := generator.Spec{
		Name: "articles", Type: "collection", Table: "articles",
		Permissions: []string{"articles.read", "articles.write"}, Dialects: []string{"sqlite", "postgres"},
		Fields: []generator.Field{{Name: "title", Type: "string"}},
	}
	generatedRoot := filepath.Join(dir, "edition")
	if _, err := generator.Generate(generatedRoot, spec); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	config := fmt.Sprintf("profile: compact\nmodules: [core, collections]\ndatabase:\n  dialect: sqlite\n  dsn: %q\n", dbPath)
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"sync", "--config", configPath, "--generated", filepath.Join(generatedRoot, "generated"), "--confirm-admin-crud"}
	var output strings.Builder
	if err := collectionsCommand(args, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "created=1 unchanged=0") || !strings.Contains(output.String(), "/admin/collections") {
		t.Fatalf("unexpected first sync output: %s", output.String())
	}
	output.Reset()
	if err := collectionsCommand(args, &output, io.Discard); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "created=0 unchanged=1") {
		t.Fatalf("unexpected repeat sync output: %s", output.String())
	}
}

func TestLoadBlueprintCollectionSchemasRequiresTenantScopeMatch(t *testing.T) {
	spec := generator.Spec{
		Name: "articles", Type: "collection", Table: "articles", TenantScoped: true,
		Permissions: []string{"articles.read", "articles.write"}, Dialects: []string{"sqlite", "postgres"},
		Fields: []generator.Field{{Name: "title", Type: "string"}},
	}
	root := t.TempDir()
	if _, err := generator.Generate(root, spec); err != nil {
		t.Fatal(err)
	}
	generatedRoot := filepath.Join(root, "generated")
	if schemas, err := loadBlueprintCollectionSchemas(generatedRoot, true); err != nil || len(schemas) != 1 {
		t.Fatalf("tenant Edition load: schemas=%#v err=%v", schemas, err)
	}
	if _, err := loadBlueprintCollectionSchemas(generatedRoot, false); err == nil {
		t.Fatal("tenant-scoped Blueprint must not be imported into an unscoped Edition")
	}
}

func TestSyncBlueprintCollectionSchemasPostgresContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	adminDB, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schemaName := "blueprint_sync_" + strconv.FormatInt(time.Now().UnixNano(), 10)
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
	catalog := collections.Service{Repository: collectionstore.Store{DB: db, Dialect: database.Postgres}}
	target := collections.Schema{ID: "categories", Label: "Categories", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true}}}
	source := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "category", Label: "Category", Kind: collections.Relation, Relation: "categories"}}}
	created, unchanged, err := syncBlueprintCollectionSchemas(ctx, catalog, []collections.Schema{source, target})
	if err != nil || created != 2 || unchanged != 0 {
		t.Fatalf("first sync: created=%d unchanged=%d err=%v", created, unchanged, err)
	}
	created, unchanged, err = syncBlueprintCollectionSchemas(ctx, catalog, []collections.Schema{source, target})
	if err != nil || created != 0 || unchanged != 2 {
		t.Fatalf("repeat sync: created=%d unchanged=%d err=%v", created, unchanged, err)
	}
	changed := target
	changed.Label = "Changed"
	if _, _, err := syncBlueprintCollectionSchemas(ctx, catalog, []collections.Schema{changed}); err == nil {
		t.Fatal("expected persisted schema drift to be rejected")
	}
}

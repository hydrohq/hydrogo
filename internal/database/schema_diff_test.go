package database_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteSchemaDiffReportsPhysicalDriftWithoutMutatingLiveDatabase(t *testing.T) {
	ctx := context.Background()
	live, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "schema-diff.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	changes, err := database.SQLiteSchemaDiff(ctx, live, migrations.FS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) == 0 || changes[0].Kind != "missing" {
		t.Fatalf("changes = %#v, want missing reference objects", changes)
	}
	var hasLedger bool
	if err := live.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations')`).Scan(&hasLedger); err != nil {
		t.Fatal(err)
	}
	if hasLedger {
		t.Fatal("schema inspection mutated the live database")
	}
}

func TestPostgresSchemaDiffFindsTypeIndexAndForeignKeyDrift(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	suffix := fmt.Sprintf("%d", os.Getpid())
	parent, child, index := "hydro_diff_parent_"+suffix, "hydro_diff_child_"+suffix, "hydro_diff_idx_"+suffix
	for _, statement := range []string{
		"CREATE TABLE " + parent + " (id TEXT PRIMARY KEY, name VARCHAR(100) NOT NULL)",
		"CREATE TABLE " + child + " (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL)",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	defer db.ExecContext(context.Background(), "DROP TABLE "+child)
	defer db.ExecContext(context.Background(), "DROP TABLE "+parent)
	migration := fmt.Sprintf(`CREATE TABLE %s (id TEXT PRIMARY KEY, name TEXT NOT NULL);
CREATE INDEX %s ON %s (name);
CREATE TABLE %s (id TEXT PRIMARY KEY, parent_id TEXT NOT NULL, CONSTRAINT fk_%s FOREIGN KEY (parent_id) REFERENCES %s(id));`, parent, index, parent, child, suffix, parent)
	bundle := fstest.MapFS{
		"001_test_schema/migration.yaml": &fstest.MapFile{Data: []byte("module: core\ntransactional:\n  postgres: true\n")},
		"001_test_schema/postgres.sql":   &fstest.MapFile{Data: []byte(migration)},
	}
	changes, err := database.PostgresSchemaDiff(ctx, db, bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"definition_mismatch:column:" + parent + ".name": false,
		"missing:index:" + parent + "." + index:          false,
		"missing:constraint:" + child + ".fk_" + suffix:  false,
	}
	for _, change := range changes {
		key := change.Kind + ":" + change.Object + ":" + change.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("changes = %#v, want %s", changes, key)
		}
	}
}

func TestPostgresSchemaDiffBuildsAndRemovesReferenceSchema(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("hydro_diff_test_%d", os.Getpid())
	if _, err := db.ExecContext(ctx, "CREATE TABLE "+name+" (id TEXT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE "+name)
	changes, err := database.PostgresSchemaDiff(ctx, db, migrations.FS, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"unexpected:base table:" + name:                        false,
		"unexpected:column:" + name + ".id":                    false,
		"unexpected:index:" + name + "." + name + "_pkey":      false,
		"unexpected:constraint:" + name + "." + name + "_pkey": false,
	}
	for _, change := range changes {
		key := change.Kind + ":" + change.Object + ":" + change.Name
		if _, ok := want[key]; ok {
			want[key] = true
		} else {
			t.Errorf("unexpected schema difference: %s", key)
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("changes = %#v, want %s", changes, key)
		}
	}
	var leaked int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_namespace WHERE nspname LIKE 'hydro_diff_%'`).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Fatalf("temporary schema count = %d, want 0", leaked)
	}
}

func TestSQLiteSchemaDiffFindsDefinitionChanges(t *testing.T) {
	ctx := context.Background()
	reference, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	reference.SetMaxOpenConns(1)
	defer reference.Close()
	if err := database.ApplyMigrations(ctx, reference, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := reference.ExecContext(ctx, `DROP INDEX idx_news_published_route`); err != nil {
		t.Fatal(err)
	}
	if _, err := reference.ExecContext(ctx, `CREATE TABLE manual_table (id TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := reference.ExecContext(ctx, `ALTER TABLE news_items ADD COLUMN manual_column TEXT`); err != nil {
		t.Fatal(err)
	}
	changes, err := database.SQLiteSchemaDiff(ctx, reference, migrations.FS, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"missing:index:idx_news_published_route": false, "unexpected:table:manual_table": false, "definition_mismatch:table:news_items": false}
	for _, change := range changes {
		key := change.Kind + ":" + change.Object + ":" + change.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("missing expected drift %s (changes %#v)", key, changes)
		}
	}
}

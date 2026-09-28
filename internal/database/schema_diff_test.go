package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

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

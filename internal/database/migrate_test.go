package database_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestMigrationAPIsRejectMissingDatabaseAndDialect(t *testing.T) {
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, nil, database.SQLite, migrations.FS); err == nil {
		t.Fatal("ApplyMigrations accepted a nil database")
	}
	if err := database.CheckCompatibility(ctx, nil, database.SQLite, migrations.FS); err == nil {
		t.Fatal("CheckCompatibility accepted a nil database")
	}
	db := &sql.DB{}
	if err := database.ApplyMigrations(ctx, db, database.Dialect("oracle"), migrations.FS); err == nil {
		t.Fatal("ApplyMigrations accepted an unsupported dialect")
	}
	if err := database.CheckCompatibility(ctx, db, database.Dialect("oracle"), migrations.FS); err == nil {
		t.Fatal("CheckCompatibility accepted an unsupported dialect")
	}
}

func TestMigrationAPIsNormalizeNilContext(t *testing.T) {
	db, err := database.Open(nil, database.SQLite, filepath.Join(t.TempDir(), "nil-migration-context.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(nil, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("migrations with nil context failed: %v", err)
	}
	if err := database.CheckCompatibility(nil, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("compatibility with nil context failed: %v", err)
	}
}

func TestCompatibilityCheckNamesTheMigrationThatIsStillPending(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "pending.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE id = '202610210001_collection_sequences'"); err != nil {
		t.Fatal(err)
	}
	err = database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS)
	if err == nil || !strings.Contains(err.Error(), "202610210001_collection_sequences has not been applied yet") || strings.Contains(err.Error(), "no rows") {
		t.Fatalf("a pending migration must be named in plain words, got %v", err)
	}
}

func TestSQLiteMigrationRestartAndCompatibilityContract(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "hydro.sqlite")
	db, err := database.Open(ctx, database.SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); err != nil {
		db.Close()
		t.Fatalf("compatibility after first apply: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// A restarted binary sees the same ledger and performs no duplicate DDL.
	db, err = database.Open(ctx, database.SQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("restart apply: %v", err)
	}
	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != migrationCount(t) {
		t.Fatalf("migration rows = %d, want %d", applied, migrationCount(t))
	}

	// A binary with changed SQL must refuse to serve the existing schema.
	var firstID string
	if err := db.QueryRowContext(ctx, "SELECT id FROM schema_migrations ORDER BY id LIMIT 1").Scan(&firstID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE schema_migrations SET checksum = ? WHERE id = ?", strings.Repeat("0", 64), firstID); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); err == nil {
		t.Fatal("checksum mismatch was accepted")
	}
}

func TestSQLiteModuleFilteringLeavesInactiveModuleSchemaAbsent(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "modules.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	enabled := map[string]bool{"pages": true, "media": true}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
		t.Fatalf("filtered migrations failed: %v", err)
	}
	for _, table := range []struct {
		name string
		want bool
	}{
		{name: "schema_migrations", want: true},
		{name: "media_assets", want: true},
		{name: "media_variants", want: true},
		{name: "page_tags", want: true},
		{name: "page_published_tags", want: true},
		{name: "commerce_products", want: false},
		{name: "news_items", want: false},
		{name: "news_taxonomy_terms", want: false},
		{name: "news_taxonomy_relations", want: false},
		{name: "sectoral_portfolio_tags", want: false},
		{name: "hydro_collections", want: false},
		{name: "hydro_collection_records", want: false},
		{name: "hydro_collection_unique", want: false},
		{name: "hydro_collection_relations", want: false},
		{name: "hydro_collection_revisions", want: false},
		{name: "hydro_collection_translations", want: false},
	} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table.name).Scan(&count); err != nil {
			t.Fatalf("inspect table %q: %v", table.name, err)
		}
		if (count == 1) != table.want {
			t.Fatalf("table %q present = %v, want %v", table.name, count == 1, table.want)
		}
	}

	if err := database.CheckCompatibilityForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
		t.Fatalf("filtered compatibility check failed: %v", err)
	}
}

func TestSQLiteCollectionsMigrationIsOptIn(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "collections-module.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"collections": true}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"hydro_collections", "hydro_collection_records", "hydro_collection_unique", "hydro_collection_relations", "hydro_collection_revisions", "hydro_collection_translations"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("enabled collections table %q exists=%v, want true", table, count == 1)
		}
	}
}

func TestSQLiteNewsModuleFilteringIncludesTaxonomySchema(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news-module.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"news": true}); err != nil {
		t.Fatalf("news-filtered migrations failed: %v", err)
	}
	for _, table := range []string{"news_items", "news_taxonomy_terms", "news_taxonomy_relations"} {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatalf("inspect table %q: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("table %q absent from enabled news schema", table)
		}
	}
}

func TestSQLiteSectoralModulesCreateSchemaOnlyWhenEnabled(t *testing.T) {
	ctx := context.Background()
	modules := []string{"locations", "team", "partners", "announcements", "portfolio", "certificates", "pricing", "events", "downloads", "testimonials", "faq", "legal"}
	for _, active := range [][]string{{"locations"}, modules} {
		db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "sectoral.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		enabled := map[string]bool{}
		for _, module := range active {
			enabled[module] = true
		}
		if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
			db.Close()
			t.Fatalf("enabled %v: %v", active, err)
		}
		for _, module := range modules {
			var count int
			table := "sectoral_" + module
			if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
				db.Close()
				t.Fatal(err)
			}
			want := enabled[module]
			if (count == 1) != want {
				db.Close()
				t.Fatalf("module %s table exists=%v want %v with enabled=%v", module, count == 1, want, active)
			}
		}
		var portfolioTags int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='sectoral_portfolio_tags'").Scan(&portfolioTags); err != nil {
			db.Close()
			t.Fatal(err)
		}
		if (portfolioTags == 1) != enabled["portfolio"] {
			db.Close()
			t.Fatalf("portfolio tag relation exists=%v want enabled=%v", portfolioTags == 1, enabled["portfolio"])
		}
		if err := database.CheckCompatibilityForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
			db.Close()
			t.Fatalf("compatibility enabled %v: %v", active, err)
		}
		db.Close()
	}
}

func TestSQLiteCoreOnlyMigrationSetOwnsAuditEvents(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "core-only.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	enabled := map[string]bool{}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, enabled); err != nil {
		t.Fatalf("core-only migrations failed: %v", err)
	}
	var auditColumns int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('audit_events') WHERE name = 'request_id'").Scan(&auditColumns); err != nil {
		t.Fatal(err)
	}
	if auditColumns != 1 {
		t.Fatal("core-only migration set must create audit_events with request_id: audit does not depend on pages")
	}
}

func TestFilteredMigrationsCanonicalizeEnabledModuleNames(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "mixed-case.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrationFS := fstest.MapFS{
		"0001_base/sqlite.sql":      &fstest.MapFile{Data: []byte("CREATE TABLE base (id TEXT PRIMARY KEY);\n")},
		"0001_base/migration.yaml":  &fstest.MapFile{Data: []byte("module: core\ntransactional:\n  sqlite: true\n")},
		"0002_pages/sqlite.sql":     &fstest.MapFile{Data: []byte("CREATE TABLE pages (id TEXT PRIMARY KEY);\n")},
		"0002_pages/migration.yaml": &fstest.MapFile{Data: []byte("module: pages\ntransactional:\n  sqlite: true\n")},
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrationFS, map[string]bool{" Pages ": true}); err != nil {
		t.Fatalf("mixed-case module migration failed: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages").Scan(&count); err != nil || count != 0 {
		t.Fatalf("pages table was not selected: count=%d err=%v", count, err)
	}
}

func TestFilteredMigrationsRejectInvalidRequiredModuleMetadata(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "invalid-metadata.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrationFS := fstest.MapFS{
		"0001_base/sqlite.sql":     &fstest.MapFile{Data: []byte("CREATE TABLE base (id INTEGER PRIMARY KEY);\n")},
		"0001_base/migration.yaml": &fstest.MapFile{Data: []byte("module: pages\ntransactional:\n  sqlite: true\n")},
		"0002_bad/sqlite.sql":      &fstest.MapFile{Data: []byte("SELECT 1;\n")},
		"0002_bad/migration.yaml":  &fstest.MapFile{Data: []byte("module: core\nrequires_modules: [pages, pages]\ntransactional:\n  sqlite: true\n")},
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrationFS, map[string]bool{"pages": true}); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("invalid required module metadata error = %v", err)
	}
}

func TestNonTransactionalMigrationUsesMetadataExecutionMode(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "nontransactional.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrationFS := fstest.MapFS{
		"0001_base/sqlite.sql":       &fstest.MapFile{Data: []byte("CREATE TABLE sample (id INTEGER PRIMARY KEY);\n")},
		"0001_base/migration.yaml":   &fstest.MapFile{Data: []byte("transactional:\n  sqlite: true\n")},
		"0002_vacuum/sqlite.sql":     &fstest.MapFile{Data: []byte("VACUUM;\n")},
		"0002_vacuum/migration.yaml": &fstest.MapFile{Data: []byte("transactional:\n  sqlite: false\n")},
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrationFS); err != nil {
		t.Fatal(err)
	}
	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 2 {
		t.Fatalf("migration rows = %d, want 2", applied)
	}
}

func TestCompatibilityRejectsUnknownAppliedMigration(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "future.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO schema_migrations (id, checksum) VALUES (?, ?)", "209901010001_future", strings.Repeat("f", 64)); err != nil {
		t.Fatal(err)
	}
	if err := database.CheckCompatibility(ctx, db, database.SQLite, migrations.FS); !errors.Is(err, database.ErrUnknownMigration) {
		t.Fatalf("unknown migration compatibility error = %v, want %v", err, database.ErrUnknownMigration)
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); !errors.Is(err, database.ErrUnknownMigration) {
		t.Fatalf("unknown migration apply error = %v, want %v", err, database.ErrUnknownMigration)
	}
}

func migrationCount(t *testing.T) int {
	t.Helper()
	entries, err := sqlFSReadDir(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

func sqlFSReadDir(fsys fs.ReadDirFS) ([]fs.DirEntry, error) {
	return fsys.ReadDir(".")
}

// Without the baseline (the chain alone) the pages migrations collide with the
// audit and outbox tables core created, and the runner says so before any SQL.
func TestEnablingPagesOnACoreCreatedDatabaseWithTheChainAloneIsRejectedBeforeAnySQL(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "core-then-pages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&before); err != nil {
		t.Fatal(err)
	}
	err = database.ApplyMigrationsForModules(ctx, db, database.SQLite, database.ChainOnly(migrations.FS), map[string]bool{"pages": true})
	if !errors.Is(err, database.ErrPagesNeedsFreshDatabase) {
		t.Fatalf("err = %v, want ErrPagesNeedsFreshDatabase", err)
	}
	var after int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("ledger changed from %d to %d rows: nothing may run before the rejection", before, after)
	}
}

func TestEnablingPagesOnAFreshDatabaseStillWorks(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "pages-first.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"pages": true}); err != nil {
		t.Fatalf("fresh database with pages: %v", err)
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"pages": true}); err != nil {
		t.Fatalf("second run must be a no-op: %v", err)
	}
}

// With the baseline the same step works: the pages group is applied as a unit.
func TestEnablingPagesOnACoreCreatedDatabaseWorksWithTheBaseline(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "core-then-pages-baseline.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"pages": true}); err != nil {
		t.Fatalf("enabling pages later: %v", err)
	}
	var tables int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='pages'").Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 1 {
		t.Fatal("the pages table was not created")
	}
}

package database_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// bundle is a migration directory that also carries a baseline.
type bundle struct {
	fs.FS
	baseline fs.FS
}

func (b bundle) BaselineFS() fs.FS { return b.baseline }

func sum(text string) string {
	digest := sha256.Sum256([]byte(text))
	return hex.EncodeToString(digest[:])
}

const (
	coreBaseline  = "CREATE TABLE chain_core (id TEXT PRIMARY KEY, extra TEXT);\nCREATE TABLE baseline_marker_core (id TEXT);\n"
	pagesBaseline = "CREATE TABLE chain_pages (id TEXT PRIMARY KEY);\nCREATE TABLE baseline_marker_pages (id TEXT);\n"
)

// fixture has a two-migration core, a one-migration pages module, and a baseline
// whose scripts leave a marker table so a test can tell which path ran.
func fixture(coreChecksum string) bundle {
	chain := fstest.MapFS{
		"0001_core/sqlite.sql":      {Data: []byte("CREATE TABLE chain_core (id TEXT PRIMARY KEY);\n")},
		"0001_core/migration.yaml":  {Data: []byte("module: core\ntransactional:\n  sqlite: true\n")},
		"0002_core/sqlite.sql":      {Data: []byte("ALTER TABLE chain_core ADD COLUMN extra TEXT;\n")},
		"0002_core/migration.yaml":  {Data: []byte("module: core\ntransactional:\n  sqlite: true\n")},
		"0003_pages/sqlite.sql":     {Data: []byte("CREATE TABLE chain_pages (id TEXT PRIMARY KEY);\n")},
		"0003_pages/migration.yaml": {Data: []byte("module: pages\ntransactional:\n  sqlite: true\n")},
	}
	if coreChecksum == "" {
		coreChecksum = sum(coreBaseline)
	}
	manifest := fmt.Sprintf(`version: 1
groups:
  - id: core
    modules: []
    migrations: [0001_core, 0002_core]
    checksums: {sqlite: %s}
  - id: pages
    modules: [pages]
    migrations: [0003_pages]
    checksums: {sqlite: %s}
`, coreChecksum, sum(pagesBaseline))
	baseline := fstest.MapFS{
		"manifest.yml":     {Data: []byte(manifest)},
		"core/sqlite.sql":  {Data: []byte(coreBaseline)},
		"pages/sqlite.sql": {Data: []byte(pagesBaseline)},
	}
	return bundle{FS: chain, baseline: baseline}
}

func openFixtureDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "baseline.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func hasTable(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count == 1
}

func ledgerIDs(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "SELECT id FROM schema_migrations ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return strings.Join(ids, ",")
}

func TestBaselineStandsInForTheChainOnAFreshDatabase(t *testing.T) {
	ctx := context.Background()
	db := openFixtureDB(t)
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, fixture(""), map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	if !hasTable(t, db, "baseline_marker_core") || !hasTable(t, db, "baseline_marker_pages") {
		t.Fatal("the baseline scripts did not run on a fresh database")
	}
	if got := ledgerIDs(t, db); got != "0001_core,0002_core,0003_pages" {
		t.Fatalf("ledger = %s: the baseline must record the chain's migrations", got)
	}

	// The ledger reads exactly as if the chain had run: same checksums.
	chainDB := openFixtureDB(t)
	if err := database.ApplyMigrationsForModules(ctx, chainDB, database.SQLite, database.ChainOnly(fixture("")), map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	read := func(db *sql.DB) string {
		var lines []string
		rows, err := db.QueryContext(ctx, "SELECT id || ' ' || checksum FROM schema_migrations ORDER BY id")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n")
	}
	if read(db) != read(chainDB) {
		t.Fatalf("baseline ledger differs from chain ledger:\n%s\n--\n%s", read(db), read(chainDB))
	}
}

func TestBaselineSkipsGroupsOfModulesThatAreNotEnabled(t *testing.T) {
	ctx := context.Background()
	db := openFixtureDB(t)
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, fixture(""), map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if !hasTable(t, db, "baseline_marker_core") || hasTable(t, db, "baseline_marker_pages") || hasTable(t, db, "chain_pages") {
		t.Fatal("only the core group may run while pages is off")
	}
	if got := ledgerIDs(t, db); got != "0001_core,0002_core" {
		t.Fatalf("ledger = %s", got)
	}
}

func TestEnablingAModuleLaterAppliesOnlyItsGroup(t *testing.T) {
	ctx := context.Background()
	db := openFixtureDB(t)
	// An existing installation: core came from the chain.
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, database.ChainOnly(fixture("")), map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, fixture(""), map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	if hasTable(t, db, "baseline_marker_core") {
		t.Fatal("the core group is already applied and must not run again")
	}
	if !hasTable(t, db, "baseline_marker_pages") {
		t.Fatal("the pages group should come from the baseline")
	}
}

func TestBaselineStepsAsideForADatabaseThatRanPartOfAGroup(t *testing.T) {
	ctx := context.Background()
	db := openFixtureDB(t)
	// An older release applied the first core migration only.
	older := fstest.MapFS{}
	for name, file := range fixture("").FS.(fstest.MapFS) {
		if strings.HasPrefix(name, "0001_core/") {
			older[name] = file
		}
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, older); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, fixture(""), map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	if hasTable(t, db, "baseline_marker_core") || hasTable(t, db, "baseline_marker_pages") {
		t.Fatal("a half-applied group means objects exist; the chain must finish the upgrade")
	}
	if got := ledgerIDs(t, db); got != "0001_core,0002_core,0003_pages" {
		t.Fatalf("ledger = %s", got)
	}
}

func TestBaselineRejectsAScriptThatDoesNotMatchItsManifest(t *testing.T) {
	db := openFixtureDB(t)
	err := database.ApplyMigrationsForModules(context.Background(), db, database.SQLite, fixture(strings.Repeat("0", 64)), map[string]bool{})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want a checksum mismatch", err)
	}
}

func TestBaselineFallsBackToTheChainWithoutAScriptForTheDialect(t *testing.T) {
	ctx := context.Background()
	db := openFixtureDB(t)
	b := fixture("")
	manifest, err := fs.ReadFile(b.baseline, "manifest.yml")
	if err != nil {
		t.Fatal(err)
	}
	b.baseline.(fstest.MapFS)["manifest.yml"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(string(manifest), "sqlite:", "postgres:"))}
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, b, map[string]bool{"pages": true}); err != nil {
		t.Fatal(err)
	}
	if hasTable(t, db, "baseline_marker_core") {
		t.Fatal("no sqlite script exists, so the chain must run")
	}
	if got := ledgerIDs(t, db); got != "0001_core,0002_core,0003_pages" {
		t.Fatalf("ledger = %s", got)
	}
}

func TestBaselineRunTwiceIsANoOp(t *testing.T) {
	ctx := context.Background()
	db := openFixtureDB(t)
	for run := 0; run < 2; run++ {
		if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, fixture(""), map[string]bool{"pages": true}); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
	}
	if got := ledgerIDs(t, db); got != "0001_core,0002_core,0003_pages" {
		t.Fatalf("ledger = %s", got)
	}
}

func TestMigrationNeedsListTheModulesAMigrationRequires(t *testing.T) {
	needs, err := database.MigrationNeeds(fstest.MapFS{
		"0001_a/migration.yaml": {Data: []byte("module: core\n")},
		"0002_b/migration.yaml": {Data: []byte("module: pages\n")},
		"0003_c/migration.yaml": {Data: []byte("module: core\nrequires_modules: [pages, media]\n")},
		"0004_d/migration.yaml": {Data: []byte("module: news\nrequires_modules: [media]\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"0001_a": "", "0002_b": "pages", "0003_c": "media,pages", "0004_d": "media,news"}
	for id, modules := range want {
		if got := strings.Join(needs[id], ","); got != modules {
			t.Errorf("%s needs %q, want %q", id, got, modules)
		}
	}
}

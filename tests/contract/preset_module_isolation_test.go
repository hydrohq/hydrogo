package contract

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/projectgen"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

var presetModuleTables = map[string]string{
	"pages":    "pages",
	"media":    "media_assets",
	"news":     "news_items",
	"commerce": "commerce_products",
	"members":  "members",
	"site":     "site_settings",
}

// TestGeneratedPresetsKeepModuleSchemasIsolated certifies that every shipped
// preset selects only its own module-owned schema. SQLite always runs; the
// PostgreSQL half runs when HYDRO_TEST_POSTGRES_URL is configured.
func TestGeneratedPresetsKeepModuleSchemasIsolated(t *testing.T) {
	presets := []string{"news", "editorial", "commerce", "compact-site", "corporate", "portal", "headless", "membership", "dashboard", "erp"}
	for _, dialect := range []database.Dialect{database.SQLite, database.Postgres} {
		dialect := dialect
		if dialect == database.Postgres && os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
			continue
		}
		for _, preset := range presets {
			t.Run(string(dialect)+"/"+preset, func(t *testing.T) {
				runPresetIsolation(t, dialect, preset)
			})
		}
	}
}

func runPresetIsolation(t *testing.T, dialect database.Dialect, preset string) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	projectPath, err := projectgen.Generate(projectgen.Options{
		Root: root, Name: "isolation-" + preset, Preset: preset,
		Profile: "compact", Database: string(dialect), Edition: "client",
	})
	if err != nil {
		t.Fatalf("generate %s: %v", preset, err)
	}
	lockBytes, err := os.ReadFile(filepath.Join(projectPath, "hydrogo.lock"))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	var lock projectgen.Lock
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatalf("decode lock: %v", err)
	}
	enabled := make(map[string]bool, len(lock.Features))
	for _, feature := range lock.Features {
		enabled[feature] = true
	}

	var db *sql.DB
	var schema string
	if dialect == database.SQLite {
		db, err = database.Open(ctx, dialect, filepath.Join(t.TempDir(), "preset.sqlite"))
	} else {
		db, err = database.OpenWithOptions(ctx, dialect, os.Getenv("HYDRO_TEST_POSTGRES_URL"), database.OpenOptions{MaxOpenConns: 1, MaxIdleConns: 1})
	}
	if err != nil {
		t.Fatalf("open %s: %v", dialect, err)
	}
	defer db.Close()
	if dialect == database.Postgres {
		schema = fmt.Sprintf("hydro_preset_%d", time.Now().UnixNano())
		if _, err := db.ExecContext(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
			t.Fatalf("create schema: %v", err)
		}
		defer func() { _, _ = db.ExecContext(ctx, `DROP SCHEMA "`+schema+`" CASCADE`) }()
		if _, err := db.ExecContext(ctx, `SET search_path TO "`+schema+`"`); err != nil {
			t.Fatalf("set search path: %v", err)
		}
	}
	if err := database.ApplyMigrationsForModules(ctx, db, dialect, migrations.FS, enabled); err != nil {
		t.Fatalf("apply %s migrations: %v", preset, err)
	}
	for module, table := range presetModuleTables {
		present := tablePresent(t, ctx, db, dialect, table)
		if present != enabled[module] {
			t.Fatalf("preset %s module %s table %s present=%v enabled=%v", preset, module, table, present, enabled[module])
		}
	}
	if err := database.CheckCompatibilityForModules(ctx, db, dialect, migrations.FS, enabled); err != nil {
		t.Fatalf("compatibility %s: %v", preset, err)
	}
}

func tablePresent(t *testing.T, ctx context.Context, db *sql.DB, dialect database.Dialect, table string) bool {
	t.Helper()
	var count int
	var err error
	if dialect == database.SQLite {
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count)
	} else {
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = $1", table).Scan(&count)
	}
	if err != nil {
		t.Fatalf("inspect table %s: %v", table, err)
	}
	return count == 1
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestTenantExportAndImportCLI(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tenant_cli.sqlite")
	db, err := database.Open(context.Background(), database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	// Insert test data for tenant-acme
	_, err = db.Exec(`INSERT INTO tenants (tenant_id, state, default_locale, timezone, created_at, updated_at) VALUES ('tenant-acme', 'active', 'en', 'UTC', '2026-01-01', '2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO site_settings (tenant_id, key, value, is_public, updated_at) VALUES ('tenant-acme', 'site.name', 'Acme Corp', 1, '2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}

	// Also insert test data for tenant-other to test isolation
	_, err = db.Exec(`INSERT INTO tenants (tenant_id, state, default_locale, timezone, created_at, updated_at) VALUES ('tenant-other', 'active', 'en', 'UTC', '2026-01-01', '2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO site_settings (tenant_id, key, value, is_public, updated_at) VALUES ('tenant-other', 'site.name', 'Other Corp', 1, '2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}

	snapshotFile := filepath.Join(t.TempDir(), "acme.tar.gz")

	// 1. Run tenant export CLI with --json
	var stdout, stderr bytes.Buffer
	exportArgs := []string{
		"tenant", "export", "tenant-acme",
		"--database-dialect=sqlite",
		"--database-dsn=" + dbPath,
		"--output=" + snapshotFile,
		"--json",
	}
	if err := run(exportArgs, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("tenant export failed: %v (%s)", err, stderr.String())
	}

	var exportOut struct {
		OK       bool   `json:"ok"`
		Path     string `json:"path"`
		Manifest struct {
			TenantID  string           `json:"tenant_id"`
			TotalRows int64            `json:"total_rows"`
			Counts    map[string]int64 `json:"table_counts"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &exportOut); err != nil {
		t.Fatalf("unmarshal export output: %v, raw: %s", err, stdout.String())
	}
	if !exportOut.OK || exportOut.Manifest.TenantID != "tenant-acme" {
		t.Fatalf("unexpected export manifest: %+v", exportOut)
	}

	// 2. Run tenant import CLI into tenant-cloned
	stdout.Reset()
	stderr.Reset()
	importArgs := []string{
		"tenant", "import", snapshotFile,
		"--database-dialect=sqlite",
		"--database-dsn=" + dbPath,
		"--new-tenant-id=tenant-cloned",
		"--json",
	}
	if err := run(importArgs, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("tenant import failed: %v (%s)", err, stderr.String())
	}

	var importOut struct {
		OK       bool `json:"ok"`
		Manifest struct {
			TenantID string `json:"tenant_id"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &importOut); err != nil {
		t.Fatalf("unmarshal import output: %v", err)
	}
	if !importOut.OK || importOut.Manifest.TenantID != "tenant-cloned" {
		t.Fatalf("unexpected import manifest: %+v", importOut)
	}

	// 3. Verify database state for tenant-cloned
	var val string
	err = db.QueryRow("SELECT value FROM site_settings WHERE tenant_id = 'tenant-cloned' AND key = 'site.name'").Scan(&val)
	if err != nil {
		t.Fatalf("query cloned setting: %v", err)
	}
	if val != "Acme Corp" {
		t.Fatalf("expected 'Acme Corp', got %s", val)
	}
}

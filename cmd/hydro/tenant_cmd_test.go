package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestTenantCLIValidateAndList(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tenant_cli_test.db")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer

	// 1. Validate with no tenancy
	err = run([]string{"tenant", "validate"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("tenant validate failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "tenancy-disabled") {
		t.Fatalf("expected tenancy-disabled output, got: %s", stdout.String())
	}

	// 1b. Validate with valid config file
	configPath := filepath.Join(tempDir, "tenant_config.yaml")
	configContent := `profile: compact
database:
  dialect: sqlite
  dsn: ` + filepath.ToSlash(dbPath) + `
tenancy:
  enabled: true
  hosts:
    alpha.example.com: tenant-alpha
    beta.example.com: tenant-beta
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{"tenant", "validate", "--config", configPath, "--json"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("tenant validate with config failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"valid":true`) || !strings.Contains(stdout.String(), `"tenant-alpha"`) {
		t.Fatalf("expected valid JSON response, got: %s", stdout.String())
	}

	// 2. Backfill with --dry-run
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"tenant", "backfill", "--tenant-id", "tenant-cli-1", "--dry-run", "--database-dsn", dbPath, "--database-dialect", "sqlite", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("tenant backfill dry-run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"dry_run":true`) || !strings.Contains(stdout.String(), `"target_tenant_id":"tenant-cli-1"`) {
		t.Fatalf("expected tenant-backfill JSON plan output, got: %s", stdout.String())
	}

	// 2b. Backfill missing both --dry-run and --confirm fails
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"tenant", "backfill", "--tenant-id", "tenant-cli-1", "--database-dsn", dbPath, "--database-dialect", "sqlite"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected error when neither --dry-run nor --confirm is passed")
	}

	// 3. Backfill with --confirm
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"tenant", "backfill", "--tenant-id", "tenant-cli-1", "--confirm", "--database-dsn", dbPath, "--database-dialect", "sqlite", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("tenant backfill confirm failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"dry_run":false`) || !strings.Contains(stdout.String(), `"target_tenant_id":"tenant-cli-1"`) {
		t.Fatalf("expected tenant-backfill-complete JSON output, got: %s", stdout.String())
	}

	// 4. Tenant list
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"tenant", "list", "--database-dsn", dbPath, "--database-dialect", "sqlite", "--json"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("tenant list failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"count":`) {
		t.Fatalf("expected JSON total_tenants output, got: %s", stdout.String())
	}
}

func TestTenantCLIProvisionReadsPasswordFromStandardInput(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tenant_provision_cli.db")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	configPath := filepath.Join(t.TempDir(), "tenant_config.yaml")
	configContent := `profile: compact
database:
  dialect: sqlite
  dsn: ` + filepath.ToSlash(dbPath) + `
tenancy:
  enabled: true
  hosts:
    tenant.example.test: tenant-cli
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"tenant", "provision", "--config", configPath, "--tenant-id", "tenant-cli", "--admin-email", "owner@example.test", "--admin-password-stdin", "--json"}, strings.NewReader("correct horse battery staple\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("tenant provision failed: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"tenant_id":"tenant-cli"`) || !strings.Contains(stdout.String(), `"state":"active"`) {
		t.Fatalf("unexpected provision output: %s", stdout.String())
	}
}

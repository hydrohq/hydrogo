package tenant

import (
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func setupSnapshotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test_snapshot.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE tenants (
		tenant_id TEXT PRIMARY KEY,
		state TEXT NOT NULL DEFAULT 'active',
		default_locale TEXT NOT NULL DEFAULT 'en',
		timezone TEXT NOT NULL DEFAULT 'UTC',
		created_at TEXT NOT NULL DEFAULT '2026-01-01',
		updated_at TEXT NOT NULL DEFAULT '2026-01-01'
	);

	CREATE TABLE site_settings (
		tenant_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		is_public INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL,
		PRIMARY KEY (tenant_id, key)
	);

	CREATE TABLE hydro_collection_records (
		tenant_id TEXT NOT NULL,
		collection_id TEXT NOT NULL,
		id TEXT NOT NULL,
		data_json TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT '2026-01-01',
		updated_at TEXT NOT NULL DEFAULT '2026-01-01',
		PRIMARY KEY (tenant_id, collection_id, id)
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func TestTenantExportAndImportSnapshot(t *testing.T) {
	db := setupSnapshotTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Seed Tenant A
	_, err := db.Exec(`INSERT INTO tenants (tenant_id) VALUES ('tenant-a'), ('tenant-b')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO site_settings (tenant_id, key, value, updated_at) VALUES 
		('tenant-a', 'site.name', 'Tenant A Portal', '2026-01-01'),
		('tenant-b', 'site.name', 'Tenant B Portal', '2026-01-01')
	`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO hydro_collection_records (id, tenant_id, collection_id, data_json) VALUES 
		('rec-a1', 'tenant-a', 'articles', '{"title":"Post A"}'),
		('rec-b1', 'tenant-b', 'articles', '{"title":"Post B"}')
	`)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Export Tenant A
	var buf bytes.Buffer
	manifest, err := ExportSnapshot(ctx, db, "sqlite", "tenant-a", &buf)
	if err != nil {
		t.Fatalf("ExportSnapshot failed: %v", err)
	}
	if manifest.TenantID != "tenant-a" {
		t.Fatalf("expected tenant-a, got %s", manifest.TenantID)
	}
	if manifest.TableCounts["site_settings"] != 1 || manifest.TableCounts["hydro_collection_records"] != 1 {
		t.Fatalf("unexpected table counts: %+v", manifest.TableCounts)
	}

	// 2. Import into new Tenant C
	importManifest, err := ImportSnapshot(ctx, db, "sqlite", bytes.NewReader(buf.Bytes()), "tenant-c")
	if err != nil {
		t.Fatalf("ImportSnapshot failed: %v", err)
	}
	if importManifest.TenantID != "tenant-c" {
		t.Fatalf("expected imported tenant to be tenant-c, got %s", importManifest.TenantID)
	}

	// 3. Verify Tenant C has records mapped with tenant_id = 'tenant-c'
	var siteName string
	err = db.QueryRow("SELECT value FROM site_settings WHERE tenant_id = 'tenant-c' AND key = 'site.name'").Scan(&siteName)
	if err != nil {
		t.Fatalf("verify site_settings for tenant-c: %v", err)
	}
	if siteName != "Tenant A Portal" {
		t.Fatalf("expected 'Tenant A Portal', got %s", siteName)
	}

	var postTitle string
	err = db.QueryRow("SELECT data_json FROM hydro_collection_records WHERE tenant_id = 'tenant-c' AND id = 'rec-a1'").Scan(&postTitle)
	if err != nil {
		t.Fatalf("verify record for tenant-c: %v", err)
	}
	if postTitle != `{"title":"Post A"}` {
		t.Fatalf("unexpected post title: %s", postTitle)
	}

	// 4. Verify Tenant B records were NOT exported/imported into Tenant C
	var bCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM hydro_collection_records WHERE tenant_id = 'tenant-c' AND id = 'rec-b1'").Scan(&bCount)
	if bCount != 0 {
		t.Fatalf("isolation violation: tenant B record was imported into tenant C!")
	}
}

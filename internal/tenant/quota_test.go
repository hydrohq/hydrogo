package tenant

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupQuotaTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "test_quotas.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	schema := `
	CREATE TABLE site_settings (
		tenant_id TEXT NOT NULL,
		key TEXT NOT NULL,
		value TEXT NOT NULL,
		is_public INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL,
		PRIMARY KEY (tenant_id, key)
	);

	CREATE TABLE media_assets (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		deleted_at TEXT
	);

	CREATE TABLE media_variants (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		size_bytes INTEGER NOT NULL
	);

	CREATE TABLE hydro_collection_records (
		id TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		collection_id TEXT NOT NULL,
		data_json TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func TestTenantQuotasAndUsage(t *testing.T) {
	db := setupQuotaTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Default Quotas
	quotas, err := GetTenantQuotas(ctx, db, "sqlite", "tenant-1")
	if err != nil {
		t.Fatalf("GetTenantQuotas: %v", err)
	}
	if quotas.MaxStorageBytes != DefaultMaxStorageBytes {
		t.Fatalf("expected default storage %d, got %d", DefaultMaxStorageBytes, quotas.MaxStorageBytes)
	}

	// 2. Custom Quota in site_settings
	_, err = db.Exec(`INSERT INTO site_settings (tenant_id, key, value, updated_at) VALUES 
		('tenant-1', 'tenant.quota.max_storage_bytes', '1000', '2026-01-01'),
		('tenant-1', 'tenant.quota.max_records_per_collection', '5', '2026-01-01')
	`)
	if err != nil {
		t.Fatalf("insert site_settings: %v", err)
	}

	quotas, err = GetTenantQuotas(ctx, db, "sqlite", "tenant-1")
	if err != nil {
		t.Fatalf("GetTenantQuotas: %v", err)
	}
	if quotas.MaxStorageBytes != 1000 || quotas.MaxRecordsPerCollection != 5 {
		t.Fatalf("expected custom quotas (1000, 5), got (%d, %d)", quotas.MaxStorageBytes, quotas.MaxRecordsPerCollection)
	}

	// 3. Media Assets and Variants Usage
	_, err = db.Exec(`INSERT INTO media_assets (id, tenant_id, size_bytes) VALUES ('a1', 'tenant-1', 400), ('a2', 'tenant-1', 300)`)
	if err != nil {
		t.Fatalf("insert media_assets: %v", err)
	}
	_, err = db.Exec(`INSERT INTO media_variants (id, tenant_id, size_bytes) VALUES ('v1', 'tenant-1', 150)`)
	if err != nil {
		t.Fatalf("insert media_variants: %v", err)
	}
	_, err = db.Exec(`INSERT INTO hydro_collection_records (id, tenant_id, collection_id, data_json) VALUES 
		('r1', 'tenant-1', 'articles', '{}'),
		('r2', 'tenant-1', 'articles', '{}')
	`)
	if err != nil {
		t.Fatalf("insert records: %v", err)
	}

	usage, err := GetTenantUsage(ctx, db, "sqlite", "tenant-1")
	if err != nil {
		t.Fatalf("GetTenantUsage: %v", err)
	}
	if usage.StorageBytes != 850 { // 400 + 300 + 150
		t.Fatalf("expected 850 storage bytes, got %d", usage.StorageBytes)
	}
	if usage.TotalMediaAssets != 2 {
		t.Fatalf("expected 2 media assets, got %d", usage.TotalMediaAssets)
	}
	if usage.TotalRecords != 2 {
		t.Fatalf("expected 2 collection records, got %d", usage.TotalRecords)
	}

	// 4. CheckStorageQuota: fits (850 + 100 <= 1000)
	if err := CheckStorageQuota(ctx, db, "sqlite", "tenant-1", 100); err != nil {
		t.Fatalf("unexpected storage quota error: %v", err)
	}

	// CheckStorageQuota: exceeds (850 + 200 > 1000)
	err = CheckStorageQuota(ctx, db, "sqlite", "tenant-1", 200)
	if err == nil {
		t.Fatal("expected storage quota error, got nil")
	}
	var qErr *QuotaError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *QuotaError, got %T: %v", err, err)
	}
	if qErr.Code != "tenant_storage_quota_exceeded" || qErr.Current != 850 || qErr.Limit != 1000 {
		t.Fatalf("unexpected QuotaError fields: %+v", qErr)
	}

	// 5. CheckRecordQuota: fits (2 < 5)
	if err := CheckRecordQuota(ctx, db, "sqlite", "tenant-1", "articles"); err != nil {
		t.Fatalf("unexpected record quota error: %v", err)
	}

	// Fill records up to limit 5
	_, _ = db.Exec(`INSERT INTO hydro_collection_records (id, tenant_id, collection_id, data_json) VALUES 
		('r3', 'tenant-1', 'articles', '{}'),
		('r4', 'tenant-1', 'articles', '{}'),
		('r5', 'tenant-1', 'articles', '{}')
	`)

	// CheckRecordQuota: exceeds (5 >= 5)
	err = CheckRecordQuota(ctx, db, "sqlite", "tenant-1", "articles")
	if err == nil {
		t.Fatal("expected record quota error, got nil")
	}
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *QuotaError, got %T: %v", err, err)
	}
	if qErr.Code != "tenant_record_quota_exceeded" || qErr.Current != 5 || qErr.Limit != 5 {
		t.Fatalf("unexpected record QuotaError: %+v", qErr)
	}
}

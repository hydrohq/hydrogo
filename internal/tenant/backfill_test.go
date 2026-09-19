package tenant_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestPlanAndExecuteBackfill(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test_backfill.db")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	// Insert rows with empty tenant_id
	_, err = db.ExecContext(ctx, `INSERT INTO pages
		(id, title, version, locale, slug, body, status, revision, publication_generation, tenant_id, created_at, updated_at)
		VALUES ('p1', 'Legacy 1', 1, 'en', 'l1', 'body1', 'draft', 1, 0, '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO pages
		(id, title, version, locale, slug, body, status, revision, publication_generation, tenant_id, created_at, updated_at)
		VALUES ('p2', 'Legacy 2', 1, 'en', 'l2', 'body2', 'draft', 1, 0, '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO site_settings (tenant_id, key, value, is_public, updated_at)
		VALUES ('', 'site.title', 'Legacy Site', 1, '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Plan Backfill (Dry Run)
	plan, err := tenant.PlanBackfill(ctx, db, "sqlite", "tenant-migrated")
	if err != nil {
		t.Fatalf("PlanBackfill failed: %v", err)
	}
	if !plan.DryRun {
		t.Errorf("expected plan.DryRun = true")
	}
	if plan.TotalRows < 3 {
		t.Errorf("expected at least 3 total unassigned rows, got %d", plan.TotalRows)
	}
	if plan.RowsAffected["pages"] < 2 {
		t.Errorf("expected at least 2 pages rows, got %d", plan.RowsAffected["pages"])
	}

	// 2. Execute Backfill
	execReport, err := tenant.ExecuteBackfill(ctx, db, "sqlite", "tenant-migrated")
	if err != nil {
		t.Fatalf("ExecuteBackfill failed: %v", err)
	}
	if execReport.DryRun {
		t.Errorf("expected execReport.DryRun = false")
	}
	if execReport.TotalRows != plan.TotalRows {
		t.Errorf("expected executed rows %d to match plan %d", execReport.TotalRows, plan.TotalRows)
	}

	// 3. Verify No Unassigned Rows Remain
	postPlan, err := tenant.PlanBackfill(ctx, db, "sqlite", "tenant-migrated")
	if err != nil {
		t.Fatalf("post-backfill plan failed: %v", err)
	}
	if postPlan.TotalRows != 0 {
		t.Errorf("expected 0 unassigned rows after backfill, got %d", postPlan.TotalRows)
	}

	// 4. Verify Pages have new tenant_id
	var count int
	err = db.QueryRowContext(ctx, "SELECT count(*) FROM pages WHERE tenant_id = 'tenant-migrated'").Scan(&count)
	if err != nil || count < 2 {
		t.Errorf("expected at least 2 pages with tenant-migrated, got count=%d, err=%v", count, err)
	}
}

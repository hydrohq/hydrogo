package tenant_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestProvisionIsIdempotentAndAudited(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "provision.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	request := tenant.ProvisionRequest{TenantID: "tenant-alpha", AdminEmail: "owner@example.test", AdminPasswordHash: "argon2-hash", Locale: "tr", Timezone: "Europe/Istanbul"}
	first, err := tenant.Provision(ctx, db, "sqlite", request)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != "active" || !first.AdminCreated || first.DefaultsApplied != 3 {
		t.Fatalf("first provision = %+v", first)
	}
	second, err := tenant.Provision(ctx, db, "sqlite", request)
	if err != nil {
		t.Fatal(err)
	}
	if second.AdminCreated || second.DefaultsApplied != 0 {
		t.Fatalf("second provision must not duplicate bootstrap rows: %+v", second)
	}
	var tenants, users, defaults, audits, lifecycleSteps int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenants WHERE tenant_id = 'tenant-alpha' AND state = 'active'`).Scan(&tenants); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = 'tenant-alpha' AND email = 'owner@example.test' AND role = 'admin'`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_settings WHERE tenant_id = 'tenant-alpha'`).Scan(&defaults); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenant_system_audit WHERE tenant_id = 'tenant-alpha' AND action = 'tenant.provision.completed'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenant_system_audit WHERE tenant_id = 'tenant-alpha' AND action IN ('tenant.provision.registry.applied', 'tenant.provision.defaults.applied', 'tenant.provision.admin.created', 'tenant.provision.admin.reused', 'tenant.provision.activated')`).Scan(&lifecycleSteps); err != nil {
		t.Fatal(err)
	}
	if tenants != 1 || users != 1 || defaults != 3 || audits != 2 || lifecycleSteps != 8 {
		t.Fatalf("registry=%d users=%d defaults=%d completed=%d lifecycle steps=%d", tenants, users, defaults, audits, lifecycleSteps)
	}
}

func TestActiveBackfillRestoresOnlyMissingDefaults(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "active-backfill.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	_, err = tenant.Provision(ctx, db, "sqlite", tenant.ProvisionRequest{TenantID: "tenant-alpha", AdminEmail: "owner@example.test", AdminPasswordHash: "argon2-hash", Locale: "en", Timezone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM site_settings WHERE tenant_id = 'tenant-alpha' AND key = 'media.default_folder'`); err != nil {
		t.Fatal(err)
	}
	dryRun, err := tenant.BackfillActiveTenants(ctx, db, "sqlite", true)
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.TenantsProcessed != 1 || dryRun.DefaultsApplied != 1 {
		t.Fatalf("dry run = %+v", dryRun)
	}
	report, err := tenant.BackfillActiveTenants(ctx, db, "sqlite", false)
	if err != nil {
		t.Fatal(err)
	}
	if report.TenantsProcessed != 1 || report.DefaultsApplied != 1 {
		t.Fatalf("report = %+v", report)
	}
	var restored, audit int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_settings WHERE tenant_id = 'tenant-alpha' AND key = 'media.default_folder'`).Scan(&restored); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tenant_system_audit WHERE tenant_id = 'tenant-alpha' AND action = 'tenant.backfill.applied'`).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if restored != 1 || audit != 1 {
		t.Fatalf("restored=%d audit=%d", restored, audit)
	}
}

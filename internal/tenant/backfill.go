package tenant

import (
	"context"
	"database/sql"
	"fmt"
)

// BackfillReport tracks unassigned rows migrated per table.
type BackfillReport struct {
	TargetTenantID string           `json:"target_tenant_id"`
	DryRun         bool             `json:"dry_run"`
	RowsAffected   map[string]int64 `json:"rows_affected"`
	TotalRows      int64            `json:"total_rows"`
}

// BackfillTables lists all 29 tenant-scoped database tables.
var BackfillTables = []string{
	"pages",
	"page_revisions",
	"audit_events",
	"outbox_events",
	"users",
	"sessions",
	"media_assets",
	"media_variants",
	"user_mfa",
	"mfa_recovery_codes",
	"auth_recovery_tokens",
	"auth_invites",
	"site_settings",
	"navigation_items",
	"page_redirects",
	"form_inbox",
	"news_items",
	"members",
	"member_sessions",
	"member_mobile_device_sessions",
	"commerce_products",
	"commerce_product_variants",
	"commerce_orders",
	"commerce_order_items",
	"commerce_carts",
	"commerce_cart_items",
	"commerce_payment_transactions",
	"commerce_payment_webhook_events",
	"hydro_forms",
}

// PlanBackfill counts rows with empty or null tenant_id in each tenant-scoped table.
func PlanBackfill(ctx context.Context, db *sql.DB, dialect string, targetTenantID string) (*BackfillReport, error) {
	if _, err := NewScope(targetTenantID, targetTenantID); err != nil {
		return nil, err
	}
	report := &BackfillReport{
		TargetTenantID: targetTenantID,
		DryRun:         true,
		RowsAffected:   make(map[string]int64),
	}
	for _, table := range BackfillTables {
		var count int64
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE tenant_id = '' OR tenant_id IS NULL", table)
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			// Table may not exist if optional module is not active
			continue
		}
		if count > 0 {
			report.RowsAffected[table] = count
			report.TotalRows += count
		}
	}
	return report, nil
}

// ExecuteBackfill assigns unassigned tenant rows to targetTenantID inside an atomic transaction.
func ExecuteBackfill(ctx context.Context, db *sql.DB, dialect string, targetTenantID string) (*BackfillReport, error) {
	if _, err := NewScope(targetTenantID, targetTenantID); err != nil {
		return nil, err
	}
	report := &BackfillReport{
		TargetTenantID: targetTenantID,
		DryRun:         false,
		RowsAffected:   make(map[string]int64),
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if dialect == "postgres" {
		if _, err := tx.ExecContext(ctx, "SELECT set_config('app.tenant_rls_enforced', 'false', true)"); err != nil {
			return nil, err
		}
	}

	for _, table := range BackfillTables {
		var query string
		if dialect == "postgres" {
			query = fmt.Sprintf("UPDATE %s SET tenant_id = $1 WHERE tenant_id = '' OR tenant_id IS NULL", table)
		} else {
			query = fmt.Sprintf("UPDATE %s SET tenant_id = ? WHERE tenant_id = '' OR tenant_id IS NULL", table)
		}
		res, err := tx.ExecContext(ctx, query, targetTenantID)
		if err != nil {
			// Fail closed and abort transaction on error
			return nil, fmt.Errorf("backfill failed for table %s: %w", table, err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve rows affected for table %s: %w", table, err)
		}
		if affected > 0 {
			report.RowsAffected[table] = affected
			report.TotalRows += affected
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return report, nil
}

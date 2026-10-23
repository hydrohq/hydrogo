package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// BackfillReport tracks unassigned rows migrated per table.
type BackfillReport struct {
	TargetTenantID string           `json:"target_tenant_id"`
	DryRun         bool             `json:"dry_run"`
	RowsAffected   map[string]int64 `json:"rows_affected"`
	TotalRows      int64            `json:"total_rows"`
}

// BackfillTables lists all 28 tenant-scoped database tables.
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
	"commerce_stock",
	"commerce_shopping_carts",
	"commerce_shopping_cart_items",
	"commerce_customer_orders",
	"commerce_customer_order_items",
	"commerce_payment_charges",
	"commerce_payment_notifications",
	"hydro_forms",
}

// isMissingTableError identifies when a table does not exist across PostgreSQL and SQLite
// without leaking driver types to caller domains.
func isMissingTableError(err error) bool {
	if err == nil {
		return false
	}

	// PostgreSQL exposes a stable SQLSTATE for an undefined relation. Use the
	// typed error instead of matching the human-readable message: messages such
	// as "does not exist" can also describe a missing column, schema, function,
	// or role and must never be silently skipped by a data migration.
	var sqlState interface{ SQLState() string }
	if errors.As(err, &sqlState) {
		return sqlState.SQLState() == "42P01"
	}

	// modernc SQLite reports missing relations with this specific prefix. Keep
	// the match narrow; generic SQLite errors (permissions, constraints, locks,
	// and syntax errors) must remain fatal.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table:")
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
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE tenant_id = '' OR tenant_id IS NULL", table) // #nosec G201 -- the table comes from the fixed BackfillTables list
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			if isMissingTableError(err) {
				// Table does not exist in this database profile (optional module inactive)
				continue
			}
			// Fail closed on permission, connection, or other SQL errors during planning
			return nil, fmt.Errorf("plan backfill error querying table %s: %w", table, err)
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
			if isMissingTableError(err) {
				// Table does not exist in this deployment profile (optional module inactive)
				continue
			}
			// Fail closed and abort transaction on real constraint, permission, or SQL error
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

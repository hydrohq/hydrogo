package contract_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/migrations"
)

var expectedRLSTables = []string{
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
	"news_taxonomy_terms",
	"news_taxonomy_relations",
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
	"refund_required",
	"customer_sms_dedup",
	"hydro_forms",
	"hydro_collections",
	"hydro_collection_records",
	"hydro_collection_unique",
	"hydro_collection_relations",
	"hydro_collection_revisions",
	"hydro_collection_translations",
	"hydro_collection_sequences",
	"hydro_roles",
	"hydro_role_permissions",
	"admin_user_permissions",
	"api_tokens",
	"notification_outbox",
	"push_subscriptions",
}

// tenantColumnWithoutRLS lists tables that have a tenant_id column but no RLS,
// with the reason. Shrink it; do not grow it without a reason.
var tenantColumnWithoutRLS = map[string]string{
	// Known gaps: their stores query outside a tenant-bound transaction
	// (and admin_notifications stores NULL tenant ids), so a fail-closed
	// policy would hide rows. Move the stores onto database.Transactor first.
	"admin_notifications":      "gap: internal/adminstudio/sqlstore plain queries, NULL tenant_id",
	"admin_digest_preferences": "gap: internal/adminstudio/sqlstore plain queries",
	"analytics_content_daily":  "gap: internal/analytics/sqlstore plain queries, cross-tenant rollups",
	"analytics_feedback_daily": "gap: internal/analytics/sqlstore plain queries",
	"outbox_dead_letters":      "gap: internal/outbox/sqlstore plain queries, cross-tenant worker",
	// Platform control plane and SaaS billing: cross-tenant by design.
	"tenants":             "platform: tenant registry",
	"tenant_governance":   "platform: orchestrator governance",
	"tenant_system_audit": "platform: provisioning audit",
	"plans":               "platform: billing",
	"subscriptions":       "platform: billing",
	"orders":              "platform: billing",
	"payments":            "platform: billing",
	"payment_events":      "platform: billing",
}

// TestPostgresTenantRLSContract certifies ADR-0156 and ADR-0180:
// 1. Every tenant-scoped table has RLS enabled and forced.
// 2. The tenant isolation policy exists on each table.
// 3. Transactions bound to Tenant A cannot read, update, or delete Tenant B data.
// 4. Inserting with mismatched tenant_id violates WITH CHECK policy.
// 5. Raw queries without a tenant context cannot read named-tenant rows.
func TestPostgresTenantRLSContract(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	var currentRole string
	var bypassesRLS bool
	if err := db.QueryRowContext(ctx, `SELECT current_user, rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user`).Scan(&currentRole, &bypassesRLS); err != nil {
		t.Fatalf("check current PostgreSQL role: %v", err)
	}
	if bypassesRLS {
		t.Fatalf("tenant RLS contract requires a non-superuser role without BYPASSRLS; %q bypasses row policies", currentRole)
	}

	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// 0. No tenant policy may stay fail-open behind the old app.tenant_rls_enforced
	// switch, which nothing sets; ADR 0032 requires every policy to fail closed.
	rows, err := db.QueryContext(ctx, `SELECT tablename FROM pg_policies
		WHERE schemaname = CURRENT_SCHEMA() AND policyname = 'hydrogo_tenant_isolation'
		  AND (coalesce(qual, '') LIKE '%tenant_rls_enforced%' OR coalesce(with_check, '') LIKE '%tenant_rls_enforced%')`)
	if err != nil {
		t.Fatalf("query fail-open policies: %v", err)
	}
	var failOpen []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		failOpen = append(failOpen, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	_ = rows.Close()
	if len(failOpen) > 0 {
		t.Fatalf("tenant policies still fail open via app.tenant_rls_enforced: %v", failOpen)
	}

	// 0b. Every table with a tenant_id column must enforce RLS, except the ones
	// listed in tenantColumnWithoutRLS. A new tenant table without a policy
	// fails here instead of silently relying on application WHERE filters.
	missingRLS, err := db.QueryContext(ctx, `SELECT c.relname FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped
		WHERE n.nspname = CURRENT_SCHEMA() AND c.relkind = 'r'
		  AND NOT (c.relrowsecurity AND c.relforcerowsecurity)
		ORDER BY c.relname`)
	if err != nil {
		t.Fatalf("query tenant tables without RLS: %v", err)
	}
	var unprotected []string
	for missingRLS.Next() {
		var name string
		if err := missingRLS.Scan(&name); err != nil {
			t.Fatal(err)
		}
		if _, allowed := tenantColumnWithoutRLS[name]; !allowed {
			unprotected = append(unprotected, name)
		}
	}
	if err := missingRLS.Err(); err != nil {
		t.Fatal(err)
	}
	_ = missingRLS.Close()
	if len(unprotected) > 0 {
		t.Fatalf("tables with tenant_id but no forced RLS (add a policy or justify them in tenantColumnWithoutRLS): %v", unprotected)
	}

	// 1. Verify RLS enabled and forced on all expected tables
	for _, tableName := range expectedRLSTables {
		t.Run("RLSEnabled/"+tableName, func(t *testing.T) {
			var rowSecurity, forceSecurity bool
			query := `SELECT c.relrowsecurity, c.relforcerowsecurity
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relname = $1 AND n.nspname = CURRENT_SCHEMA()`
			if err := db.QueryRowContext(ctx, query, tableName).Scan(&rowSecurity, &forceSecurity); err != nil {
				t.Fatalf("failed to query pg_class for table %s: %v", tableName, err)
			}
			if !rowSecurity {
				t.Errorf("expected relrowsecurity = true for table %s", tableName)
			}
			if !forceSecurity {
				t.Errorf("expected relforcerowsecurity = true for table %s", tableName)
			}

			var policyCount int
			polQuery := `SELECT count(*) FROM pg_policies WHERE tablename = $1 AND schemaname = CURRENT_SCHEMA() AND policyname = 'hydrogo_tenant_isolation'`
			if err := db.QueryRowContext(ctx, polQuery, tableName).Scan(&policyCount); err != nil {
				t.Fatalf("failed to query pg_policies for table %s: %v", tableName, err)
			}
			if policyCount != 1 {
				t.Errorf("expected exactly 1 hydrogo_tenant_isolation policy for table %s, got %d", tableName, policyCount)
			}
		})
	}

	// 2. Test Cross-Tenant Isolation with TenantTransactor
	scopeAlpha, err := tenant.NewScope("tenant-alpha", "org-alpha")
	if err != nil {
		t.Fatal(err)
	}
	scopeBeta, err := tenant.NewScope("tenant-beta", "org-beta")
	if err != nil {
		t.Fatal(err)
	}

	ctxAlpha := tenant.WithScope(ctx, scopeAlpha)
	ctxBeta := tenant.WithScope(ctx, scopeBeta)

	transactor := database.NewTenantTransactor(db, database.Postgres, true)

	// Step A: Insert records in tenant-alpha
	err = transactor.Within(ctxAlpha, func(tx database.Executor) error {
		// Insert Page
		_, err := tx.ExecContext(ctxAlpha, `INSERT INTO pages
			(id, title, version, locale, slug, body, status, revision, publication_generation, tenant_id, created_at, updated_at)
			VALUES ('page-alpha', 'Alpha Home', 1, 'en', 'alpha-home', 'Alpha Content', 'published', 1, 0, 'tenant-alpha', NOW(), NOW())`)
		if err != nil {
			return fmt.Errorf("insert alpha page: %w", err)
		}

		// Insert Site Settings
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO site_settings (key, value, is_public, tenant_id, updated_at)
			VALUES ('site_name', 'Alpha Site', true, 'tenant-alpha', NOW())`)
		if err != nil {
			return fmt.Errorf("insert alpha site_settings: %w", err)
		}

		// Insert Member
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO members (id, email, password_hash, active, tenant_id, created_at, updated_at)
			VALUES ('member-alpha', 'member@alpha.com', 'hash', true, 'tenant-alpha', NOW(), NOW())`)
		if err != nil {
			return fmt.Errorf("insert alpha member: %w", err)
		}

		// Insert User
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO users (id, email, password_hash, role, active, tenant_id, created_at, updated_at)
			VALUES ('user-alpha', 'admin@alpha.com', 'hash', 'admin', true, 'tenant-alpha', NOW(), NOW())`)
		if err != nil {
			return fmt.Errorf("insert alpha user: %w", err)
		}

		// Insert Media Asset
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO media_assets (id, storage_key, original_filename, mime_type, size_bytes, visibility, status, tenant_id, created_at, updated_at)
			VALUES ('media-alpha', 'tenants/tenant-alpha/alpha.png', 'alpha.png', 'image/png', 1024, 'public', 'ready', 'tenant-alpha', NOW(), NOW())`)
		if err != nil {
			return fmt.Errorf("insert alpha media_asset: %w", err)
		}

		// Insert Commerce Stock
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO commerce_stock (id, variant_id, available, reserved, tenant_id, created_at, updated_at)
			VALUES ('stock-alpha', 'variant-alpha', 10, 0, 'tenant-alpha', NOW(), NOW())`)
		if err != nil {
			return fmt.Errorf("insert alpha commerce_stock: %w", err)
		}

		// Insert Audit Event
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO audit_events (id, action, target_id, tenant_id)
			VALUES ('audit-alpha', 'page.publish', 'page-alpha', 'tenant-alpha')`)
		if err != nil {
			return fmt.Errorf("insert alpha audit_event: %w", err)
		}

		// Insert Collection and Collection Record
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO hydro_collections (id, label, schema_json, version, tenant_id, created_at, updated_at)
			VALUES ('col-alpha', 'Articles', '{}', 1, 'tenant-alpha', '2026-01-01', '2026-01-01')`)
		if err != nil {
			return fmt.Errorf("insert alpha collection: %w", err)
		}
		_, err = tx.ExecContext(ctxAlpha, `INSERT INTO hydro_collection_records (collection_id, id, data_json, version, tenant_id, created_at, updated_at)
			VALUES ('col-alpha', 'rec-alpha', '{}', 1, 'tenant-alpha', '2026-01-01', '2026-01-01')`)
		if err != nil {
			return fmt.Errorf("insert alpha collection record: %w", err)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to populate tenant-alpha data: %v", err)
	}

	// Step B: Verify Tenant Alpha reads its own data
	err = transactor.WithinQuery(ctxAlpha, func(qe database.QueryExecutor) error {
		var pageCount, settingsCount, memberCount, userCount, mediaCount, productCount, auditCount, colRecordCount int
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM pages WHERE id = 'page-alpha'`).Scan(&pageCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM site_settings WHERE key = 'site_name'`).Scan(&settingsCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM members WHERE id = 'member-alpha'`).Scan(&memberCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM users WHERE id = 'user-alpha'`).Scan(&userCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM media_assets WHERE id = 'media-alpha'`).Scan(&mediaCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM commerce_stock WHERE id = 'stock-alpha'`).Scan(&productCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM audit_events WHERE id = 'audit-alpha'`).Scan(&auditCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxAlpha, `SELECT count(*) FROM hydro_collection_records WHERE id = 'rec-alpha'`).Scan(&colRecordCount); err != nil {
			return err
		}
		if pageCount != 1 || settingsCount != 1 || memberCount != 1 || userCount != 1 || mediaCount != 1 || productCount != 1 || auditCount != 1 || colRecordCount != 1 {
			return fmt.Errorf("expected tenant-alpha counts all 1, got (%d,%d,%d,%d,%d,%d,%d,%d)", pageCount, settingsCount, memberCount, userCount, mediaCount, productCount, auditCount, colRecordCount)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tenant-alpha read verification failed: %v", err)
	}

	// Step C: Verify Tenant Beta CANNOT read Tenant Alpha's data (SELECT returns 0 rows)
	err = transactor.WithinQuery(ctxBeta, func(qe database.QueryExecutor) error {
		var pageCount, settingsCount, memberCount, userCount, mediaCount, productCount, auditCount, colRecordCount int
		// Notice no explicit WHERE tenant_id filter here, to prove RLS policy alone blocks cross-tenant reads!
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM pages WHERE id = 'page-alpha'`).Scan(&pageCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM site_settings WHERE key = 'site_name'`).Scan(&settingsCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM members WHERE id = 'member-alpha'`).Scan(&memberCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM users WHERE id = 'user-alpha'`).Scan(&userCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM media_assets WHERE id = 'media-alpha'`).Scan(&mediaCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM commerce_stock WHERE id = 'stock-alpha'`).Scan(&productCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM audit_events WHERE id = 'audit-alpha'`).Scan(&auditCount); err != nil {
			return err
		}
		if err := qe.QueryRowContext(ctxBeta, `SELECT count(*) FROM hydro_collection_records WHERE id = 'rec-alpha'`).Scan(&colRecordCount); err != nil {
			return err
		}
		if pageCount != 0 || settingsCount != 0 || memberCount != 0 || userCount != 0 || mediaCount != 0 || productCount != 0 || auditCount != 0 || colRecordCount != 0 {
			return fmt.Errorf("security breach: tenant-beta saw tenant-alpha rows! counts: (%d,%d,%d,%d,%d,%d,%d,%d)", pageCount, settingsCount, memberCount, userCount, mediaCount, productCount, auditCount, colRecordCount)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tenant-beta isolation check failed: %v", err)
	}

	// Step D: Verify Tenant Beta CANNOT update Tenant Alpha's data
	err = transactor.Within(ctxBeta, func(tx database.Executor) error {
		res, err := tx.ExecContext(ctxBeta, `UPDATE pages SET title = 'Hacked' WHERE id = 'page-alpha'`)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 0 {
			return fmt.Errorf("security breach: tenant-beta updated %d tenant-alpha rows", rows)
		}

		resCol, err := tx.ExecContext(ctxBeta, `UPDATE hydro_collection_records SET data_json = '{"hacked":true}' WHERE id = 'rec-alpha'`)
		if err != nil {
			return err
		}
		rowsCol, err := resCol.RowsAffected()
		if err != nil {
			return err
		}
		if rowsCol != 0 {
			return fmt.Errorf("security breach: tenant-beta updated %d tenant-alpha collection rows", rowsCol)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tenant-beta update isolation check failed: %v", err)
	}

	// Step E: Verify Tenant Beta CANNOT delete Tenant Alpha's data
	err = transactor.Within(ctxBeta, func(tx database.Executor) error {
		res, err := tx.ExecContext(ctxBeta, `DELETE FROM pages WHERE id = 'page-alpha'`)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 0 {
			return fmt.Errorf("security breach: tenant-beta deleted %d tenant-alpha rows", rows)
		}

		resCol, err := tx.ExecContext(ctxBeta, `DELETE FROM hydro_collection_records WHERE id = 'rec-alpha'`)
		if err != nil {
			return err
		}
		rowsCol, err := resCol.RowsAffected()
		if err != nil {
			return err
		}
		if rowsCol != 0 {
			return fmt.Errorf("security breach: tenant-beta deleted %d tenant-alpha collection rows", rowsCol)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("tenant-beta delete isolation check failed: %v", err)
	}

	// Step F: Verify WITH CHECK prevents Tenant Beta from writing rows with tenant-alpha tenant_id
	err = transactor.Within(ctxBeta, func(tx database.Executor) error {
		_, err := tx.ExecContext(ctxBeta, `INSERT INTO pages
			(id, title, version, locale, slug, body, status, revision, publication_generation, tenant_id, created_at, updated_at)
			VALUES ('page-spoofed', 'Spoofed', 1, 'en', 'spoofed', 'Spoofed Content', 'published', 1, 0, 'tenant-alpha', NOW(), NOW())`)
		if err == nil {
			return fmt.Errorf("security breach: tenant-beta was allowed to insert row with tenant_id='tenant-alpha'")
		}

		_, err = tx.ExecContext(ctxBeta, `INSERT INTO hydro_collection_records
			(collection_id, id, data_json, version, tenant_id, created_at, updated_at)
			VALUES ('col-alpha', 'rec-spoofed', '{}', 1, 'tenant-alpha', '2026-01-01', '2026-01-01')`)
		if err == nil {
			return fmt.Errorf("security breach: tenant-beta was allowed to insert collection record with tenant_id='tenant-alpha'")
		}
		return nil
	})
	if err == nil {
		t.Fatal("WITH CHECK constraint verification unexpectedly committed")
	}

	// Step G: The application transactor still requires a named tenant scope.
	err = transactor.WithinQuery(context.Background(), func(qe database.QueryExecutor) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected transactor to reject context without tenant when RequireTenantScope is true")
	}

	// Step H: A raw connection without app.current_tenant_id cannot read named-tenant rows.
	var rawPageCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pages WHERE id = 'page-alpha'`).Scan(&rawPageCount); err != nil {
		t.Fatalf("unscoped raw query failed: %v", err)
	}
	if rawPageCount != 0 {
		t.Fatalf("unscoped raw query saw %d named-tenant rows", rawPageCount)
	}
}

// TestPostgresRLSDefaultSingleTenantScope verifies the default empty tenant
// scope remains usable without weakening isolation for named tenants.
func TestPostgresRLSDefaultSingleTenantScope(t *testing.T) {
	ctx := context.Background()
	db := openIsolatedPostgres(t, ctx)
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}
	insertPage := func(tx *sql.Tx, id, tenantID string) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO pages
			(id, title, version, locale, slug, body, status, revision, publication_generation, tenant_id, created_at, updated_at)
			VALUES ($1, $1, 1, 'en', $1, '', 'draft', 1, 0, $2, NOW(), NOW())`, id, tenantID)
		return err
	}

	// No GUC at all means the default tenant, whose id is the empty string.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertPage(tx, "rls-default-empty", ""); err != nil {
		t.Fatalf("insert default-scope page with no tenant setting: %v", err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pages WHERE id = 'rls-default-empty'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("default-scope row not visible without tenant setting: count=%d err=%v", count, err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE pages SET title = 'rls-default-updated' WHERE id = 'rls-default-empty'`); err != nil {
		t.Fatalf("update default-scope page with no tenant setting: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// An explicit empty setting has exactly the same default-scope access.
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', '', true)`); err != nil {
		t.Fatal(err)
	}
	if err := insertPage(tx, "rls-default-explicit-empty", ""); err != nil {
		t.Fatalf("insert default-scope page with empty tenant setting: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pages WHERE id = 'rls-default-explicit-empty'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("default-scope row not visible with empty tenant setting: count=%d err=%v", count, err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE pages SET title = 'rls-default-explicit-updated' WHERE id = 'rls-default-explicit-empty'`); err != nil {
		t.Fatalf("update default-scope page with empty tenant setting: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Seed two named tenants, each under its own tenant context.
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', 'tenant-alpha', true)`); err != nil {
		t.Fatal(err)
	}
	if err := insertPage(tx, "rls-alpha", "tenant-alpha"); err != nil {
		t.Fatalf("insert alpha page: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', 'tenant-beta', true)`); err != nil {
		t.Fatal(err)
	}
	if err := insertPage(tx, "rls-beta", "tenant-beta"); err != nil {
		t.Fatalf("insert beta page: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// With no setting and with an explicit empty setting, the empty-scope row
	// is visible but named-tenant rows remain hidden.
	for _, setEmpty := range []bool{false, true} {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if setEmpty {
			if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', '', true)`); err != nil {
				t.Fatal(err)
			}
		}
		for _, id := range []string{"rls-alpha", "rls-beta"} {
			if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM pages WHERE id = $1`, id).Scan(&count); err != nil || count != 0 {
				t.Fatalf("empty context set=%t saw %s: count=%d err=%v", setEmpty, id, count, err)
			}
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}

	alpha, err := tenant.NewScope("tenant-alpha", "org-alpha")
	if err != nil {
		t.Fatal(err)
	}
	transactor := database.NewTenantTransactor(db, database.Postgres, true)
	if err := transactor.WithinQuery(tenant.WithScope(ctx, alpha), func(qe database.QueryExecutor) error {
		for _, id := range []string{"rls-default-empty", "rls-beta"} {
			if err := qe.QueryRowContext(ctx, `SELECT count(*) FROM pages WHERE id = $1`, id).Scan(&count); err != nil {
				return err
			} else if count != 0 {
				return fmt.Errorf("tenant-alpha saw row %s", id)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Raw queries with no tenant context cannot see a named tenant's row.
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pages WHERE id = 'rls-alpha'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("unscoped query saw alpha row: count=%d err=%v", count, err)
	}

	// WITH CHECK rejects both another named tenant and the default scope.
	for i, tenantID := range []string{"tenant-beta", ""} {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_tenant_id', 'tenant-alpha', true)`); err != nil {
			t.Fatal(err)
		}
		err = insertPage(tx, fmt.Sprintf("rls-spoof-%d", i), tenantID)
		_ = tx.Rollback()
		if err == nil {
			t.Fatalf("tenant-alpha inserted row with tenant_id=%q", tenantID)
		}
	}
}

package tenant

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

var ErrProvisionInput = errors.New("invalid tenant provisioning input")

var rlsTablePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

type ProvisionRequest struct {
	TenantID          string
	AdminEmail        string
	AdminPasswordHash string
	Locale            string
	Timezone          string
}

type ProvisionReport struct {
	TenantID        string `json:"tenant_id"`
	State           string `json:"state"`
	AdminCreated    bool   `json:"admin_created"`
	DefaultsApplied int    `json:"defaults_applied"`
}

type ActiveBackfillReport struct {
	DryRun           bool  `json:"dry_run"`
	TenantsProcessed int   `json:"tenants_processed"`
	DefaultsApplied  int64 `json:"defaults_applied"`
}

var bootstrapDefaults = []struct {
	Key, Value string
	Public     bool
}{
	{"site.default_locale", "tr", true},
	{"site.timezone", "Europe/Istanbul", false},
	// Media folders have no standalone domain entity. This explicit setting is
	// the bootstrap contract used by a later media-folder module.
	{"media.default_folder", "uploads", false},
}

func validateProvisionRequest(request ProvisionRequest) error {
	if _, err := NewScope(strings.TrimSpace(request.TenantID), strings.TrimSpace(request.TenantID)); err != nil {
		return err
	}
	if strings.TrimSpace(request.AdminEmail) == "" || len(request.AdminEmail) > 320 || strings.TrimSpace(request.AdminPasswordHash) == "" {
		return ErrProvisionInput
	}
	if !validLocale(request.Locale) || !validTimezone(request.Timezone) {
		return ErrProvisionInput
	}
	return nil
}

func validLocale(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 2 || len(value) > 32 {
		return false
	}
	for _, r := range value {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == '-') {
			return false
		}
	}
	return true
}

func validTimezone(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r == '_' || r == '/' || r == '-' || r == '+') {
			return false
		}
	}
	return true
}

// VerifyProvisioningRLS makes RLS a PostgreSQL provisioning precondition. The
// SQLite contract has no database-side RLS and remains useful for local tests.
func VerifyProvisioningRLS(ctx context.Context, db *sql.DB, dialect string) error {
	return VerifyProvisioningRLSForTables(ctx, db, dialect, []string{"pages", "users", "site_settings", "media_assets"})
}

// VerifyProvisioningRLSForTables verifies the database-side defense-in-depth
// contract for Core and generated tenant-scoped tables. Table identifiers are
// validated as data before being used as query parameters; Editions must pass
// their complete tenant table set before provisioning or enabling shared mode.
func VerifyProvisioningRLSForTables(ctx context.Context, db *sql.DB, dialect string, tables []string) error {
	if db == nil {
		return errors.New("database is required")
	}
	if len(tables) == 0 || len(tables) > 256 {
		return ErrProvisionInput
	}
	validated := make([]string, 0, len(tables))
	seen := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		table = strings.TrimSpace(table)
		if !rlsTablePattern.MatchString(table) {
			return fmt.Errorf("invalid RLS table %q", table)
		}
		if _, ok := seen[table]; ok {
			return fmt.Errorf("duplicate RLS table %q", table)
		}
		seen[table] = struct{}{}
		validated = append(validated, table)
	}
	if dialect != "postgres" {
		return nil
	}
	sort.Strings(validated)
	for _, table := range validated {
		var enabled, forced, policy bool
		err := db.QueryRowContext(ctx, `SELECT c.relrowsecurity, c.relforcerowsecurity,
 EXISTS (SELECT 1 FROM pg_policies p WHERE p.schemaname = current_schema() AND p.tablename = c.relname AND p.policyname = 'hydrogo_tenant_isolation')
 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = current_schema() AND c.relname = $1`, table).Scan(&enabled, &forced, &policy)
		if err != nil {
			return fmt.Errorf("verify RLS for %s: %w", table, err)
		}
		if !enabled || !forced || !policy {
			return fmt.Errorf("RLS policy is not active for %s", table)
		}
	}
	return nil
}

func Provision(ctx context.Context, db *sql.DB, dialect string, request ProvisionRequest) (*ProvisionReport, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	if err := validateProvisionRequest(request); err != nil {
		return nil, err
	}
	request.TenantID, request.AdminEmail = strings.TrimSpace(request.TenantID), strings.ToLower(strings.TrimSpace(request.AdminEmail))
	request.Locale, request.Timezone = strings.TrimSpace(request.Locale), strings.TrimSpace(request.Timezone)
	if err := recordSystemAudit(ctx, db, dialect, request.TenantID, "tenant.provision.started"); err != nil {
		return nil, err
	}
	report, err := provisionTransaction(ctx, db, dialect, request)
	if err != nil {
		_ = recordSystemAudit(ctx, db, dialect, request.TenantID, "tenant.provision.failed")
		return nil, err
	}
	for _, action := range provisionAuditActions(report) {
		if err := recordSystemAudit(ctx, db, dialect, request.TenantID, action); err != nil {
			return nil, err
		}
	}
	if err := recordSystemAudit(ctx, db, dialect, request.TenantID, "tenant.provision.completed"); err != nil {
		return nil, err
	}
	return report, nil
}

func provisionAuditActions(report *ProvisionReport) []string {
	actions := []string{
		"tenant.provision.registry.applied",
		"tenant.provision.defaults.applied",
	}
	if report.AdminCreated {
		actions = append(actions, "tenant.provision.admin.created")
	} else {
		actions = append(actions, "tenant.provision.admin.reused")
	}
	return append(actions, "tenant.provision.activated")
}

func provisionTransaction(ctx context.Context, db *sql.DB, dialect string, request ProvisionRequest) (*ProvisionReport, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if dialect == "postgres" {
		scope, _ := NewScope(request.TenantID, request.TenantID)
		if err := BindPostgresTransaction(ctx, tx, scope); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	if err := upsertTenant(ctx, tx, dialect, request.TenantID, request.Locale, request.Timezone, now); err != nil {
		return nil, err
	}
	defaults, err := ensureDefaults(ctx, tx, dialect, request.TenantID, request.Locale, request.Timezone, now)
	if err != nil {
		return nil, err
	}
	created, err := ensureAdmin(ctx, tx, dialect, request, now)
	if err != nil {
		return nil, err
	}
	if err := setTenantState(ctx, tx, dialect, request.TenantID, "active", now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ProvisionReport{TenantID: request.TenantID, State: "active", AdminCreated: created, DefaultsApplied: defaults}, nil
}

func placeholder(dialect string, n int) string {
	if dialect == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func upsertTenant(ctx context.Context, tx *sql.Tx, dialect string, id, locale, timezone string, now time.Time) error {
	query := `INSERT INTO tenants (tenant_id, state, default_locale, timezone, created_at, updated_at) VALUES (` + placeholder(dialect, 1) + `, ` + placeholder(dialect, 2) + `, ` + placeholder(dialect, 3) + `, ` + placeholder(dialect, 4) + `, ` + placeholder(dialect, 5) + `, ` + placeholder(dialect, 6) + `) ON CONFLICT(tenant_id) DO UPDATE SET updated_at = excluded.updated_at`
	if dialect == "mysql" {
		query = `INSERT INTO tenants (tenant_id, state, default_locale, timezone, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`
	}
	args := []any{id, "provisioning", locale, timezone, dbTime(dialect, now), dbTime(dialect, now)}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}
func setTenantState(ctx context.Context, tx *sql.Tx, dialect string, id, state string, now time.Time) error {
	_, err := tx.ExecContext(ctx, `UPDATE tenants SET state = `+placeholder(dialect, 1)+`, updated_at = `+placeholder(dialect, 2)+` WHERE tenant_id = `+placeholder(dialect, 3), state, dbTime(dialect, now), id)
	return err
}
func dbTime(dialect string, value time.Time) any {
	if dialect == "sqlite" {
		return value.Format(time.RFC3339Nano)
	}
	return value
}

func ensureDefaults(ctx context.Context, tx *sql.Tx, dialect string, tenantID, locale, timezone string, now time.Time) (int, error) {
	defaults := append([]struct {
		Key, Value string
		Public     bool
	}(nil), bootstrapDefaults...)
	defaults[0].Value, defaults[1].Value = locale, timezone
	inserted := 0
	for _, item := range defaults {
		query := `INSERT INTO site_settings (tenant_id, key, value, is_public, updated_at) VALUES (` + placeholder(dialect, 1) + `, ` + placeholder(dialect, 2) + `, ` + placeholder(dialect, 3) + `, ` + placeholder(dialect, 4) + `, ` + placeholder(dialect, 5) + `) ON CONFLICT(tenant_id, key) DO NOTHING`
		if dialect == "mysql" {
			query = "INSERT IGNORE INTO site_settings (tenant_id, `key`, `value`, is_public, updated_at) VALUES (?, ?, ?, ?, ?)"
		}
		public := any(item.Public)
		if dialect == "sqlite" || dialect == "mysql" {
			if item.Public {
				public = 1
			} else {
				public = 0
			}
		}
		result, err := tx.ExecContext(ctx, query, tenantID, item.Key, item.Value, public, dbTime(dialect, now))
		if err != nil {
			return 0, err
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		inserted += int(rows)
	}
	return inserted, nil
}

func ensureAdmin(ctx context.Context, tx *sql.Tx, dialect string, request ProvisionRequest, now time.Time) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE tenant_id = ` + placeholder(dialect, 1) + ` AND email = ` + placeholder(dialect, 2)
	if err := tx.QueryRowContext(ctx, query, request.TenantID, request.AdminEmail).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	id, err := randomID()
	if err != nil {
		return false, err
	}
	query = `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at, tenant_id) VALUES (` + placeholder(dialect, 1) + `, ` + placeholder(dialect, 2) + `, ` + placeholder(dialect, 3) + `, ` + placeholder(dialect, 4) + `, ` + placeholder(dialect, 5) + `, ` + placeholder(dialect, 6) + `, ` + placeholder(dialect, 7) + `, ` + placeholder(dialect, 8) + `)`
	active := any(true)
	if dialect == "sqlite" {
		active = 1
	}
	_, err = tx.ExecContext(ctx, query, id, request.AdminEmail, request.AdminPasswordHash, "admin", active, dbTime(dialect, now), dbTime(dialect, now), request.TenantID)
	return err == nil, err
}

func recordSystemAudit(ctx context.Context, db *sql.DB, dialect string, tenantID, action string) error {
	id, err := randomID()
	if err != nil {
		return err
	}
	query := `INSERT INTO tenant_system_audit (id, tenant_id, action, created_at) VALUES (` + placeholder(dialect, 1) + `, ` + placeholder(dialect, 2) + `, ` + placeholder(dialect, 3) + `, ` + placeholder(dialect, 4) + `)`
	_, err = db.ExecContext(ctx, query, id, tenantID, action, dbTime(dialect, time.Now().UTC()))
	return err
}

func randomID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// BackfillActiveTenants applies each missing bootstrap setting in a separate,
// short transaction. It avoids one global maintenance transaction and is safe
// to resume after interruption.
func BackfillActiveTenants(ctx context.Context, db *sql.DB, dialect string, dryRun bool) (*ActiveBackfillReport, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	rows, err := db.QueryContext(ctx, `SELECT tenant_id, default_locale, timezone FROM tenants WHERE state = 'active' ORDER BY tenant_id`)
	if err != nil {
		return nil, err
	}
	type activeTenant struct{ id, locale, timezone string }
	var active []activeTenant
	for rows.Next() {
		var current activeTenant
		if err := rows.Scan(&current.id, &current.locale, &current.timezone); err != nil {
			rows.Close()
			return nil, err
		}
		active = append(active, current)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	report := &ActiveBackfillReport{DryRun: dryRun}
	for _, current := range active {
		report.TenantsProcessed++
		if dryRun {
			count, err := countMissingDefaults(ctx, db, dialect, current.id)
			if err != nil {
				return nil, err
			}
			report.DefaultsApplied += count
			continue
		}
		if err := backfillOneTenant(ctx, db, dialect, current.id, current.locale, current.timezone, report); err != nil {
			return nil, err
		}
	}
	return report, nil
}
func countMissingDefaults(ctx context.Context, db *sql.DB, dialect string, tenantID string) (int64, error) {
	var missing int64
	for _, item := range bootstrapDefaults {
		var exists int
		query := `SELECT COUNT(*) FROM site_settings WHERE tenant_id = ` + placeholder(dialect, 1) + ` AND key = ` + placeholder(dialect, 2)
		if err := db.QueryRowContext(ctx, query, tenantID, item.Key).Scan(&exists); err != nil {
			return 0, err
		}
		if exists == 0 {
			missing++
		}
	}
	return missing, nil
}
func backfillOneTenant(ctx context.Context, db *sql.DB, dialect string, id, locale, timezone string, report *ActiveBackfillReport) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if dialect == "postgres" {
		scope, _ := NewScope(id, id)
		if err := BindPostgresTransaction(ctx, tx, scope); err != nil {
			return err
		}
	}
	applied, err := ensureDefaults(ctx, tx, dialect, id, locale, timezone, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	report.DefaultsApplied += int64(applied)
	if err := recordSystemAudit(ctx, db, dialect, id, "tenant.backfill.applied"); err != nil {
		return err
	}
	return nil
}

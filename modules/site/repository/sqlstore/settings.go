package sqlstore

import (
	"context"
	"database/sql"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (s Store) UpsertSetting(ctx context.Context, tx database.Executor, value site.Setting) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	updatedAt := value.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	var now any = updatedAt
	if s.Dialect == database.SQLite {
		now = updatedAt.Format(time.RFC3339Nano)
	}
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	keyColumn := "key"
	if s.Dialect == database.MySQL {
		keyColumn = "`key`"
	}
	query := `INSERT INTO site_settings (tenant_id,` + keyColumn + `,value,is_public,updated_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `) ON CONFLICT (tenant_id,` + keyColumn + `) DO UPDATE SET value=EXCLUDED.value,is_public=EXCLUDED.is_public,updated_at=EXCLUDED.updated_at`
	publicArg := any(value.Public)
	if s.Dialect != database.Postgres {
		publicArg = boolInt(value.Public)
	}
	args := []any{scopeID, value.Key, value.Value, publicArg, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO site_settings (tenant_id,` + keyColumn + `,value,is_public,updated_at) VALUES (?,?,?,?,?) ON CONFLICT(tenant_id,` + keyColumn + `) DO UPDATE SET value=excluded.value,is_public=excluded.is_public,updated_at=excluded.updated_at`
	}
	if s.Dialect == database.MySQL {
		query = `INSERT INTO site_settings (tenant_id,` + keyColumn + `,value,is_public,updated_at) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE value=VALUES(value),is_public=VALUES(is_public),updated_at=VALUES(updated_at)`
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) ListPublicSettings(ctx context.Context, db database.Executor) ([]site.Setting, error) {
	filter, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return nil, err
	}
	keyColumn := "key"
	if s.Dialect == database.MySQL {
		keyColumn = "`key`"
	}
	query := `SELECT ` + keyColumn + `,value,is_public,updated_at FROM site_settings WHERE is_public = TRUE` + filter + ` ORDER BY ` + keyColumn + ` LIMIT 1000`
	if s.Dialect != database.Postgres {
		query = `SELECT ` + keyColumn + `,value,is_public,updated_at FROM site_settings WHERE is_public = 1` + filter + ` ORDER BY ` + keyColumn + ` LIMIT 1000`
	}
	args := []any{}
	if filter != "" {
		args = append(args, scopeID)
	}
	rows, err := queryContext(ctx, db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.Setting, 0)
	for rows.Next() {
		var value site.Setting
		var public any
		var updated any
		if err := rows.Scan(&value.Key, &value.Value, &public, &updated); err != nil {
			return nil, err
		}
		value.Public = parseBool(public)
		value.UpdatedAt = parseTime(updated)
		result = append(result, value)
	}
	return result, rows.Err()
}

// ListSettings reads every setting, private ones included, within the current
// tenant scope. Callers must not expose private values publicly.
func (s Store) ListSettings(ctx context.Context, db database.Executor) ([]site.Setting, error) {
	_, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return nil, err
	}
	keyColumn := "key"
	if s.Dialect == database.MySQL {
		keyColumn = "`key`"
	}
	query := `SELECT ` + keyColumn + `,value,is_public,updated_at FROM site_settings WHERE tenant_id = ` + s.p(1) + ` ORDER BY ` + keyColumn + ` LIMIT 1000`
	rows, err := queryContext(ctx, db, query, scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.Setting, 0)
	for rows.Next() {
		var value site.Setting
		var public, updated any
		if err := rows.Scan(&value.Key, &value.Value, &public, &updated); err != nil {
			return nil, err
		}
		value.Public = parseBool(public)
		value.UpdatedAt = parseTime(updated)
		result = append(result, value)
	}
	return result, rows.Err()
}

// GetSetting reads a single setting, including private settings, within the
// current tenant scope. Callers must not expose private values publicly.
func (s Store) GetSetting(ctx context.Context, db database.Executor, key string) (site.Setting, bool, error) {
	_, scopeID, err := s.tenantFilter(ctx, 1)
	if err != nil {
		return site.Setting{}, false, err
	}
	keyColumn := "key"
	if s.Dialect == database.MySQL {
		keyColumn = "`key`"
	}
	args := []any{key, scopeID}
	query := `SELECT ` + keyColumn + `,value,is_public,updated_at FROM site_settings WHERE ` + keyColumn + `=` + s.p(1) + ` AND tenant_id=` + s.p(2)
	var rows *sql.Rows
	rows, err = queryContext(ctx, db, query, args...)
	if err != nil {
		return site.Setting{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return site.Setting{}, false, rows.Err()
	}
	var value site.Setting
	var isPublic any
	var updated any
	if err := rows.Scan(&value.Key, &value.Value, &isPublic, &updated); err != nil {
		return site.Setting{}, false, err
	}
	value.Public = parseBool(isPublic)
	value.UpdatedAt = parseTime(updated)
	return value, true, nil
}

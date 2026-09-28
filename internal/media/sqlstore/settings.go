package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
)

const mediaSettingsKey = "media.admin_settings"

func (s Store) LoadAdminSettings(ctx context.Context, db database.Executor) (media.AdminSettings, bool, error) {
	if raw, ok := db.(*sql.DB); ok {
		var value media.AdminSettings
		var found bool
		err := database.NewTenantTransactor(raw, s.Dialect, s.RequireTenantScope).WithinQuery(ctx, func(tx database.QueryExecutor) error {
			var e error
			value, found, e = s.LoadAdminSettings(ctx, tx)
			return e
		})
		return value, found, err
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return media.AdminSettings{}, false, err
	}
	if err = database.RequireExecutor(db); err != nil {
		return media.AdminSettings{}, false, err
	}
	query := "SELECT value FROM site_settings WHERE "
	if s.Dialect == database.MySQL {
		query += "`key`"
	} else {
		query += "key"
	}
	query += "=" + s.p(1) + " AND tenant_id=" + s.p(2)
	tenantID := ""
	if scoped {
		tenantID = scope.TenantID
	}
	var raw string
	if err := db.QueryRowContext(ctx, query, mediaSettingsKey, tenantID).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return media.AdminSettings{}, false, nil
		}
		return media.AdminSettings{}, false, err
	}
	var value media.AdminSettings
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return media.AdminSettings{}, false, err
	}
	return value, true, nil
}

func (s Store) SaveAdminSettings(ctx context.Context, db database.Executor, value media.AdminSettings) error {
	if raw, ok := db.(*sql.DB); ok {
		return database.NewTenantTransactor(raw, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error { return s.SaveAdminSettings(ctx, tx, value) })
	}
	scope, scoped, err := s.scope(ctx)
	if err != nil {
		return err
	}
	if err = database.RequireExecutor(db); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tenantID := ""
	if scoped {
		tenantID = scope.TenantID
	}
	key := "key"
	if s.Dialect == database.MySQL {
		key = "`key`"
	}
	query := "INSERT INTO site_settings (tenant_id," + key + ",value,is_public,updated_at) VALUES (" + s.p(1) + "," + s.p(2) + "," + s.p(3) + "," + s.p(4) + "," + s.p(5) + ") ON CONFLICT (tenant_id," + key + ") DO UPDATE SET value=EXCLUDED.value,is_public=EXCLUDED.is_public,updated_at=EXCLUDED.updated_at"
	args := []any{tenantID, mediaSettingsKey, string(encoded), false, time.Now().UTC()}
	if s.Dialect == database.SQLite {
		query = "INSERT INTO site_settings (tenant_id," + key + ",value,is_public,updated_at) VALUES (?,?,?,?,?) ON CONFLICT(tenant_id," + key + ") DO UPDATE SET value=excluded.value,is_public=excluded.is_public,updated_at=excluded.updated_at;"
		args[3] = 0
		args[4] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if s.Dialect == database.MySQL {
		query = "INSERT INTO site_settings (tenant_id," + key + ",value,is_public,updated_at) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE value=VALUES(value),is_public=VALUES(is_public),updated_at=VALUES(updated_at)"
		args[3] = 0
		args[4] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err = db.ExecContext(ctx, query, args...)
	return err
}

package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (s Store) PutRedirect(ctx context.Context, tx database.Executor, value site.Redirect) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	createdAt := value.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	var now any = createdAt
	if s.Dialect == database.SQLite {
		now = createdAt.Format(time.RFC3339Nano)
	}
	scopeID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		scopeID = scope.TenantID
	}
	query := `INSERT INTO page_redirects (tenant_id,source_path,target_path,status,created_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `) ON CONFLICT (tenant_id,source_path) DO UPDATE SET target_path=EXCLUDED.target_path,status=EXCLUDED.status`
	args := []any{scopeID, value.SourcePath, value.TargetPath, value.Status, now}
	if s.Dialect == database.SQLite {
		query = `INSERT INTO page_redirects (tenant_id,source_path,target_path,status,created_at) VALUES (?,?,?,?,?) ON CONFLICT(tenant_id,source_path) DO UPDATE SET target_path=excluded.target_path,status=excluded.status`
	} else if s.Dialect == database.MySQL {
		query = `INSERT INTO page_redirects (tenant_id,source_path,target_path,status,created_at) VALUES (?,?,?,?,?) ON DUPLICATE KEY UPDATE target_path=VALUES(target_path),status=VALUES(status)`
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) GetRedirect(ctx context.Context, db database.Executor, source string) (site.Redirect, error) {
	if err := database.RequireExecutor(db); err != nil {
		return site.Redirect{}, err
	}
	filter, scopeID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return site.Redirect{}, err
	}
	query := `SELECT source_path,target_path,status,created_at FROM page_redirects WHERE source_path = $1` + filter
	if s.Dialect != database.Postgres {
		query = `SELECT source_path,target_path,status,created_at FROM page_redirects WHERE source_path = ?` + filter
	}
	var value site.Redirect
	var created any
	args := []any{source}
	if filter != "" {
		args = append(args, scopeID)
	}
	err = db.QueryRowContext(ctx, query, args...).Scan(&value.SourcePath, &value.TargetPath, &value.Status, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return site.Redirect{}, sql.ErrNoRows
	}
	value.CreatedAt = parseTime(created)
	return value, err
}

func (s Store) UpsertRedirectRule(ctx context.Context, tx database.Executor, value site.RedirectRule) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return err
		}
		tenantID = scope.TenantID
	}
	created := value.CreatedAt.UTC()
	if created.IsZero() {
		created = time.Now().UTC()
	}
	var createdArg any = created
	if s.Dialect == database.SQLite {
		createdArg = created.Format(time.RFC3339Nano)
	}
	query := `INSERT INTO seo_redirect_rules(tenant_id,id,pattern,match_kind,target_path,status,hits,created_at) VALUES (` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,` + s.p(4) + `,` + s.p(5) + `,` + s.p(6) + `,0,` + s.p(7) + `) ON CONFLICT(tenant_id,id) DO UPDATE SET pattern=EXCLUDED.pattern,match_kind=EXCLUDED.match_kind,target_path=EXCLUDED.target_path,status=EXCLUDED.status`
	if s.Dialect == database.SQLite {
		query = `INSERT INTO seo_redirect_rules(tenant_id,id,pattern,match_kind,target_path,status,hits,created_at) VALUES (?,?,?,?,?,?,0,?) ON CONFLICT(tenant_id,id) DO UPDATE SET pattern=excluded.pattern,match_kind=excluded.match_kind,target_path=excluded.target_path,status=excluded.status`
	} else if s.Dialect == database.MySQL {
		query = `INSERT INTO seo_redirect_rules(tenant_id,id,pattern,match_kind,target_path,status,hits,created_at) VALUES (?,?,?,?,?,?,0,?) ON DUPLICATE KEY UPDATE pattern=VALUES(pattern),match_kind=VALUES(match_kind),target_path=VALUES(target_path),status=VALUES(status)`
	}
	_, err := tx.ExecContext(ctx, query, tenantID, value.ID, value.Pattern, value.Kind, value.TargetPath, value.Status, createdArg)
	return err
}

func (s Store) DeleteRedirectRule(ctx context.Context, tx database.Executor, id string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	if strings.HasPrefix(id, "legacy:") {
		return s.deleteLegacyRedirect(ctx, tx, strings.TrimPrefix(id, "legacy:"))
	}
	filter, tenantID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return err
	}
	query := `DELETE FROM seo_redirect_rules WHERE id = ` + s.p(1) + filter
	result, err := tx.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) ListRedirectRules(ctx context.Context, db database.QueryExecutor, limit int) ([]site.RedirectRule, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return nil, err
		}
		tenantID = scope.TenantID
	}
	query := `SELECT id,tenant_id,pattern,match_kind,target_path,status,hits,created_at FROM seo_redirect_rules WHERE tenant_id = ` + s.p(1) + ` ORDER BY match_kind,pattern,id LIMIT ` + s.p(2)
	if s.Dialect != database.Postgres {
		query = `SELECT id,tenant_id,pattern,match_kind,target_path,status,hits,created_at FROM seo_redirect_rules WHERE tenant_id = ? ORDER BY match_kind,pattern,id LIMIT ?`
	}
	rows, err := db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]site.RedirectRule, 0, limit)
	for rows.Next() {
		var value site.RedirectRule
		var created any
		if err := rows.Scan(&value.ID, &value.TenantID, &value.Pattern, &value.Kind, &value.TargetPath, &value.Status, &value.Hits, &created); err != nil {
			return nil, err
		}
		value.CreatedAt = parseTime(created)
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	legacyQuery := `SELECT source_path,tenant_id,target_path,status,created_at FROM page_redirects WHERE tenant_id = ` + s.p(1) + ` ORDER BY source_path LIMIT ` + s.p(2)
	if s.Dialect != database.Postgres {
		legacyQuery = `SELECT source_path,tenant_id,target_path,status,created_at FROM page_redirects WHERE tenant_id = ? ORDER BY source_path LIMIT ?`
	}
	legacyRows, err := db.QueryContext(ctx, legacyQuery, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer legacyRows.Close()
	for legacyRows.Next() {
		var rule site.RedirectRule
		var created any
		if err := legacyRows.Scan(&rule.Pattern, &rule.TenantID, &rule.TargetPath, &rule.Status, &created); err != nil {
			return nil, err
		}
		rule.ID = "legacy:" + rule.Pattern
		rule.Kind = "exact"
		rule.CreatedAt = parseTime(created)
		result = append(result, rule)
	}
	return result, legacyRows.Err()
}

func (s Store) deleteLegacyRedirect(ctx context.Context, tx database.Executor, source string) error {
	filter, tenantID, err := s.tenantFilter(ctx, 2)
	if err != nil {
		return err
	}
	query := `DELETE FROM page_redirects WHERE source_path = ` + s.p(1) + filter
	result, err := tx.ExecContext(ctx, query, source, tenantID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return sql.ErrNoRows
	}
	return nil
}

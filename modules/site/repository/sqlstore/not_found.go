package sqlstore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

const maxNotFoundPathsPerTenant = 5000

func (s Store) RecordNotFound(ctx context.Context, tx database.Executor, path string, at time.Time) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	path, err := canonical.Path(path)
	if err != nil || path == "" || len(path) > 2047 {
		return site.ErrInvalidRedirect
	}
	path = "/" + path
	tenantID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		tenantID = scope.TenantID
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()
	var stamp any = at
	if s.Dialect == database.SQLite {
		stamp = at.Format(time.RFC3339Nano)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(path)))
	query := `INSERT INTO site_not_found_hits(tenant_id,path_hash,path,hit_count,first_seen,last_seen) SELECT ` + s.p(1) + `,` + s.p(2) + `,` + s.p(3) + `,1,` + s.p(4) + `,` + s.p(5) + ` WHERE EXISTS(SELECT 1 FROM site_not_found_hits WHERE tenant_id=` + s.p(1) + ` AND path_hash=` + s.p(2) + `) OR (SELECT COUNT(*) FROM site_not_found_hits WHERE tenant_id=` + s.p(1) + `) < ` + fmt.Sprint(maxNotFoundPathsPerTenant) + ` ON CONFLICT(tenant_id,path_hash) DO UPDATE SET path=excluded.path,hit_count=site_not_found_hits.hit_count+1,last_seen=excluded.last_seen`
	if s.Dialect == database.Postgres {
		query = `INSERT INTO site_not_found_hits(tenant_id,path_hash,path,hit_count,first_seen,last_seen) SELECT $1,$2,$3,1,$4,$5 WHERE EXISTS(SELECT 1 FROM site_not_found_hits WHERE tenant_id=$1 AND path_hash=$2) OR (SELECT COUNT(*) FROM site_not_found_hits WHERE tenant_id=$1) < ` + fmt.Sprint(maxNotFoundPathsPerTenant) + ` ON CONFLICT(tenant_id,path_hash) DO UPDATE SET path=EXCLUDED.path,hit_count=site_not_found_hits.hit_count+1,last_seen=EXCLUDED.last_seen`
	}
	if s.Dialect == database.MySQL {
		query = `INSERT INTO site_not_found_hits(tenant_id,path_hash,path,hit_count,first_seen,last_seen) SELECT ?,?,?,1,?,? WHERE EXISTS(SELECT 1 FROM site_not_found_hits WHERE tenant_id=? AND path_hash=?) OR (SELECT COUNT(*) FROM site_not_found_hits WHERE tenant_id=?) < ` + fmt.Sprint(maxNotFoundPathsPerTenant) + ` ON DUPLICATE KEY UPDATE path=VALUES(path),hit_count=hit_count+1,last_seen=VALUES(last_seen)`
	}
	args := []any{tenantID, hash, path, stamp, stamp, tenantID, hash, tenantID}
	if s.Dialect == database.Postgres {
		args = []any{tenantID, hash, path, stamp, stamp}
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s Store) ListNotFound(ctx context.Context, db database.QueryExecutor, limit int) ([]site.NotFoundPath, error) {
	if err := database.RequireExecutor(db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 500 {
		return nil, site.ErrInvalidRedirect
	}
	query := `SELECT path,hit_count,first_seen,last_seen FROM site_not_found_hits WHERE tenant_id=` + s.p(1) + ` ORDER BY hit_count DESC,last_seen DESC,path LIMIT ` + s.p(2)
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return nil, err
		}
		tenantID = scope.TenantID
	}
	rows, err := db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]site.NotFoundPath, 0, limit)
	for rows.Next() {
		var item site.NotFoundPath
		var first, last any
		if err := rows.Scan(&item.Path, &item.Hits, &first, &last); err != nil {
			return nil, err
		}
		item.FirstSeen = parseTime(first)
		item.LastSeen = parseTime(last)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Store) DeleteNotFound(ctx context.Context, tx database.Executor, path string) error {
	if err := database.RequireExecutor(tx); err != nil {
		return err
	}
	path, err := canonical.Path(path)
	if err != nil || path == "" {
		return site.ErrInvalidRedirect
	}
	path = "/" + path
	tenantID := ""
	if s.RequireTenantScope {
		scope, scopeErr := tenant.Require(ctx)
		if scopeErr != nil {
			return scopeErr
		}
		tenantID = scope.TenantID
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(path)))
	_, err = tx.ExecContext(ctx, `DELETE FROM site_not_found_hits WHERE tenant_id=`+s.p(1)+` AND path_hash=`+s.p(2), tenantID, hash)
	return err
}

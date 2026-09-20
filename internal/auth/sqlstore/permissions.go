package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func (s *UserStore) LoadPermissionOverrides(ctx context.Context, userID string) (map[string]bool, error) {
	if err := requireUserDB(s); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT permission, allowed FROM admin_user_permissions WHERE user_id = $1 AND tenant_id = $2`
	args := []any{userID, tenantID}
	if s.dialect == database.SQLite {
		query = `SELECT permission, allowed FROM admin_user_permissions WHERE user_id = ? AND tenant_id = ?`
	}
	result := make(map[string]bool)
	err = s.within(ctx, func(exec database.Executor) error {
		queryExec, ok := exec.(database.QueryExecutor)
		if !ok {
			return errors.New("query executor is required")
		}
		rows, queryErr := queryExec.QueryContext(ctx, query, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()
		for rows.Next() {
			var permission string
			var allowed any
			if scanErr := rows.Scan(&permission, &allowed); scanErr != nil {
				return scanErr
			}
			result[permission] = parseActive(allowed)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load user permissions: %w", err)
	}
	return result, nil
}

func (s *UserStore) ReplacePermissionOverrides(ctx context.Context, userID string, overrides map[string]bool) error {
	if err := requireUserDB(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	return s.within(ctx, func(exec database.Executor) error {
		deleteQuery := `DELETE FROM admin_user_permissions WHERE user_id = $1 AND tenant_id = $2`
		deleteArgs := []any{userID, tenantID}
		if s.dialect == database.SQLite {
			deleteQuery = `DELETE FROM admin_user_permissions WHERE user_id = ? AND tenant_id = ?`
		}
		if _, err := exec.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
			return err
		}
		now := time.Now().UTC()
		for permission, allowed := range overrides {
			query := `INSERT INTO admin_user_permissions (user_id, permission, allowed, tenant_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
			args := []any{userID, permission, allowed, tenantID, now, now}
			if s.dialect == database.SQLite {
				query = `INSERT INTO admin_user_permissions (user_id, permission, allowed, tenant_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
				args[2] = boolToSQLite(allowed)
				args[4], args[5] = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
			}
			if _, err := exec.ExecContext(ctx, query, args...); err != nil {
				return err
			}
		}
		return nil
	})
}

var _ auth.PermissionStore = (*UserStore)(nil)

package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func (s *UserStore) List(ctx context.Context) ([]*auth.User, error) {
	if err := requireUserDB(s); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at, email_verified_at FROM users ORDER BY email`
	args := []any{}
	if s.RequireTenantScope {
		query = `SELECT id, email, password_hash, role, active, created_at, updated_at, email_verified_at FROM users WHERE tenant_id = $1 ORDER BY email`
		args = append(args, tenantID)
		if s.dialect == database.SQLite {
			query = `SELECT id, email, password_hash, role, active, created_at, updated_at, email_verified_at FROM users WHERE tenant_id = ? ORDER BY email`
		}
	}
	var users []*auth.User
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
			u := &auth.User{}
			var role string
			var active any
			var created, updated, verified any
			if scanErr := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &role, &active, &created, &updated, &verified); scanErr != nil {
				return scanErr
			}
			u.Role = auth.Role(role)
			u.Active = parseActive(active)
			u.CreatedAt = parseTime(created)
			u.UpdatedAt = parseTime(updated)
			u.EmailVerifiedAt = parseOptionalTime(verified)
			users = append(users, u)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	for _, user := range users {
		user.PermissionOverrides, err = s.LoadPermissionOverrides(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load user permissions: %w", err)
		}
	}
	return users, nil
}

func (s *UserStore) UpdateRole(ctx context.Context, userID string, role auth.Role) error {
	if role != auth.RoleAdmin && role != auth.RoleEditor && role != auth.RoleViewer {
		return errors.New("invalid user role")
	}
	if err := requireUserDB(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE users SET role = $1, updated_at = $2 WHERE id = $3`
	args := []any{string(role), time.Now().UTC(), userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $4`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `UPDATE users SET role = ?, updated_at = ? WHERE id = ?`
		args = []any{string(role), time.Now().UTC().Format(time.RFC3339Nano), userID}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	var changed int64
	err = s.within(ctx, func(exec database.Executor) error {
		result, execErr := exec.ExecContext(ctx, query, args...)
		if execErr != nil {
			return execErr
		}
		var rowsErr error
		changed, rowsErr = result.RowsAffected()
		return rowsErr
	})
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	if changed != 1 {
		return auth.ErrUserNotFound
	}
	return nil
}

var _ auth.UserAdminRepository = (*UserStore)(nil)

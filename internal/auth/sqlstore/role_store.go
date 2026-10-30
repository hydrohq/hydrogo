package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// RoleStore provides persistence for tenant-isolated custom roles and granular permissions.
type RoleStore struct {
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s RoleStore) p(position int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func (s RoleStore) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func (s RoleStore) ListRoles(ctx context.Context) ([]auth.CustomRole, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, name, description, created_at, updated_at FROM hydro_roles WHERE tenant_id = ` + s.p(1) + ` ORDER BY name ASC`
	rows, err := s.DB.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []auth.CustomRole
	for rows.Next() {
		var r auth.CustomRole
		var created, updated string
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &created, &updated); err != nil {
			return nil, err
		}
		r.TenantID = tenantID
		r.CreatedAt = parseTime(created)
		r.UpdatedAt = parseTime(updated)
		roles = append(roles, r)
	}

	// Load permissions for each role
	for i := range roles {
		perms, err := s.loadPermissions(ctx, tenantID, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}

	return roles, nil
}

func (s RoleStore) GetRole(ctx context.Context, id string) (auth.CustomRole, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return auth.CustomRole{}, err
	}

	query := `SELECT id, name, description, created_at, updated_at FROM hydro_roles WHERE tenant_id = ` + s.p(1) + ` AND id = ` + s.p(2)
	var r auth.CustomRole
	var created, updated string
	if err := s.DB.QueryRowContext(ctx, query, tenantID, id).Scan(&r.ID, &r.Name, &r.Description, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.CustomRole{}, auth.ErrRoleNotFound
		}
		return auth.CustomRole{}, err
	}

	r.TenantID = tenantID
	r.CreatedAt = parseTime(created)
	r.UpdatedAt = parseTime(updated)

	perms, err := s.loadPermissions(ctx, tenantID, id)
	if err != nil {
		return auth.CustomRole{}, err
	}
	r.Permissions = perms

	return r, nil
}

func (s RoleStore) SaveRole(ctx context.Context, role auth.CustomRole) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(strings.ToLower(role.ID))
	name := strings.TrimSpace(role.Name)
	if !auth.ValidCustomRoleName(id) || name == "" {
		return auth.ErrRoleInvalid
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Upsert role
	var insertRole string
	if s.Dialect == database.Postgres {
		insertRole = `INSERT INTO hydro_roles (tenant_id, id, name, description, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6) 
			ON CONFLICT (tenant_id, id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at`
	} else if s.Dialect == database.MySQL {
		insertRole = `INSERT INTO hydro_roles (tenant_id, id, name, description, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?) 
			ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description), updated_at = VALUES(updated_at)`
	} else {
		insertRole = `INSERT INTO hydro_roles (tenant_id, id, name, description, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?) 
			ON CONFLICT(tenant_id, id) DO UPDATE SET name = excluded.name, description = excluded.description, updated_at = excluded.updated_at`
	}

	if _, err := tx.ExecContext(ctx, insertRole, tenantID, id, name, role.Description, nowStr, nowStr); err != nil {
		return err
	}

	// 2. Clear old permissions
	deletePerms := `DELETE FROM hydro_role_permissions WHERE tenant_id = ` + s.p(1) + ` AND role_id = ` + s.p(2)
	if _, err := tx.ExecContext(ctx, deletePerms, tenantID, id); err != nil {
		return err
	}

	// 3. Insert new permissions
	for _, perm := range role.Permissions {
		res := strings.TrimSpace(perm.Resource)
		act := strings.TrimSpace(perm.Action)
		if res == "" || act == "" {
			continue
		}

		insertPerm := `INSERT INTO hydro_role_permissions (tenant_id, role_id, resource, action) VALUES (` + s.p(1) + `, ` + s.p(2) + `, ` + s.p(3) + `, ` + s.p(4) + `)`
		if _, err := tx.ExecContext(ctx, insertPerm, tenantID, id, res, act); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s RoleStore) DeleteRole(ctx context.Context, id string) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}

	query := `DELETE FROM hydro_roles WHERE tenant_id = ` + s.p(1) + ` AND id = ` + s.p(2)
	res, err := s.DB.ExecContext(ctx, query, tenantID, id)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return auth.ErrRoleNotFound
	}

	return nil
}

func (s RoleStore) loadPermissions(ctx context.Context, tenantID, roleID string) ([]auth.RolePermission, error) {
	query := `SELECT resource, action FROM hydro_role_permissions WHERE tenant_id = ` + s.p(1) + ` AND role_id = ` + s.p(2) + ` ORDER BY resource ASC, action ASC`
	rows, err := s.DB.QueryContext(ctx, query, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []auth.RolePermission
	for rows.Next() {
		var p auth.RolePermission
		if err := rows.Scan(&p.Resource, &p.Action); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}

	return perms, nil
}

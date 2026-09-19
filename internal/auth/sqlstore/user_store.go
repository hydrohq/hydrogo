package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type UserStore struct {
	db                 *sql.DB
	dialect            database.Dialect
	RequireTenantScope bool
}

func (s *UserStore) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func NewUserStore(db *sql.DB, dialect database.Dialect) *UserStore {
	return &UserStore{db: db, dialect: dialect}
}

func (s *UserStore) Create(ctx context.Context, u *auth.User) error {
	if err := requireUserDB(s); err != nil {
		return err
	}
	if u == nil {
		return errors.New("user is required")
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	query := `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at`
	args := []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), u.Active, u.CreatedAt, u.UpdatedAt}
	if s.RequireTenantScope {
		query += `, tenant_id`
		args = append(args, tenantID)
	}
	query += `) VALUES (`
	for i := range args {
		if i > 0 {
			query += `, `
		}
		query += fmt.Sprintf("$%d", i+1)
	}
	query += `)`
	if s.dialect == database.SQLite {
		query = `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at`
		args = []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), boolToSQLite(u.Active), u.CreatedAt.Format(time.RFC3339Nano), u.UpdatedAt.Format(time.RFC3339Nano)}
		if s.RequireTenantScope {
			query += `, tenant_id`
			args = append(args, tenantID)
		}
		query += `) VALUES (`
		for i := range args {
			if i > 0 {
				query += `, `
			}
			query += `?`
		}
		query += `)`
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (s *UserStore) CreateUserTx(ctx context.Context, tx database.Executor, u *auth.User) error {
	if err := requireUserStore(s); err != nil {
		return err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	if u == nil {
		return errors.New("user is required")
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	query := `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at`
	args := []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), u.Active, u.CreatedAt, u.UpdatedAt}
	if s.RequireTenantScope {
		query += `, tenant_id`
		args = append(args, tenantID)
	}
	query += `) VALUES (`
	for i := range args {
		if i > 0 {
			query += `,`
		}
		query += fmt.Sprintf("$%d", i+1)
	}
	query += `)`
	if s.dialect == database.SQLite {
		query = `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at`
		active := 0
		if u.Active {
			active = 1
		}
		args = []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), active, u.CreatedAt.Format(time.RFC3339Nano), u.UpdatedAt.Format(time.RFC3339Nano)}
		if s.RequireTenantScope {
			query += `, tenant_id`
			args = append(args, tenantID)
		}
		query += `) VALUES (`
		for i := range args {
			if i > 0 {
				query += `,`
			}
			query += `?`
		}
		query += `)`
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*auth.User, error) {
	if err := requireUserDB(s); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE id = $1`
	args := []any{id}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE id = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}

	var u auth.User
	var roleStr string
	var activeVal any
	var createdAtVal, updatedAtVal any

	err = s.db.QueryRowContext(ctx, query, args...).Scan(&u.ID, &u.Email, &u.PasswordHash, &roleStr, &activeVal, &createdAtVal, &updatedAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	u.Role = auth.Role(roleStr)
	u.Active = parseActive(activeVal)
	u.CreatedAt = parseTime(createdAtVal)
	u.UpdatedAt = parseTime(updatedAtVal)

	return &u, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	if err := requireUserDB(s); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE email = $1`
	args := []any{auth.NormalizeEmail(email)}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE email = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}

	var u auth.User
	var roleStr string
	var activeVal any
	var createdAtVal, updatedAtVal any

	err = s.db.QueryRowContext(ctx, query, args...).Scan(&u.ID, &u.Email, &u.PasswordHash, &roleStr, &activeVal, &createdAtVal, &updatedAtVal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	u.Role = auth.Role(roleStr)
	u.Active = parseActive(activeVal)
	u.CreatedAt = parseTime(createdAtVal)
	u.UpdatedAt = parseTime(updatedAtVal)

	return &u, nil
}

func (s *UserStore) Count(ctx context.Context) (int64, error) {
	if err := requireUserDB(s); err != nil {
		return 0, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return 0, err
	}
	query := "SELECT COUNT(*) FROM users"
	args := []any{}
	if s.RequireTenantScope {
		query += " WHERE tenant_id = $1"
		args = append(args, tenantID)
		if s.dialect == database.SQLite {
			query = "SELECT COUNT(*) FROM users WHERE tenant_id = ?"
		}
	}
	var count int64
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (s *UserStore) SetActive(ctx context.Context, id string, active bool) error {
	if err := requireUserDB(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE users SET active = $1, updated_at = $2 WHERE id = $3`
	args := []any{active, time.Now().UTC(), id}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $4`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `UPDATE users SET active = ?, updated_at = ? WHERE id = ?`
		args = []any{boolToSQLite(active), time.Now().UTC().Format(time.RFC3339Nano), id}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user active state: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return auth.ErrUserNotFound
	}
	return nil
}

func boolToSQLite(value bool) int {
	if value {
		return 1
	}
	return 0
}

func parseActive(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int64:
		return val == 1
	case int:
		return val == 1
	case []byte:
		return string(val) == "1" || string(val) == "true" || string(val) == "t"
	case string:
		return val == "1" || val == "true" || val == "t"
	default:
		return false
	}
}

func parseTime(v any) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val.UTC()
	case string:
		t, err := time.Parse(time.RFC3339Nano, val)
		if err == nil {
			return t.UTC()
		}
		t, err = time.Parse(time.RFC3339, val)
		if err == nil {
			return t.UTC()
		}
		return time.Time{}
	case []byte:
		return parseTime(string(val))
	default:
		return time.Time{}
	}
}

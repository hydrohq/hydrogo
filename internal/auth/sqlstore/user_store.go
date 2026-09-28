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

// within keeps direct user lookups on the same transaction boundary as the
// rest of the shared-mode application. This is required for PostgreSQL RLS:
// the tenant GUC and the query must use the same physical transaction.
func (s *UserStore) within(ctx context.Context, fn func(database.Executor) error) error {
	if err := requireUserDB(s); err != nil {
		return err
	}
	return database.NewTenantTransactor(s.db, s.dialect, s.RequireTenantScope).Within(ctx, fn)
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

	active := any(u.Active)
	if s.dialect != database.Postgres {
		active = boolToSQLite(u.Active)
	}
	var emailVerified any
	if u.EmailVerifiedAt != nil {
		emailVerified = s.dbTime(*u.EmailVerifiedAt)
	}
	args := []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), active, s.dbTime(u.CreatedAt), s.dbTime(u.UpdatedAt), emailVerified}
	columns := `id, email, password_hash, role, active, created_at, updated_at, email_verified_at`
	if s.RequireTenantScope {
		columns += `, tenant_id`
		args = append(args, tenantID)
	}
	query := `INSERT INTO users (` + columns + `) VALUES (` + s.bindList(len(args)) + `)`

	return s.within(ctx, func(exec database.Executor) error {
		_, execErr := exec.ExecContext(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("failed to insert user: %w", execErr)
		}
		return nil
	})
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
	active := any(u.Active)
	if s.dialect != database.Postgres {
		active = boolToSQLite(u.Active)
	}
	var emailVerified any
	if u.EmailVerifiedAt != nil {
		emailVerified = s.dbTime(*u.EmailVerifiedAt)
	}
	args := []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), active, s.dbTime(u.CreatedAt), s.dbTime(u.UpdatedAt), emailVerified}
	columns := `id, email, password_hash, role, active, created_at, updated_at, email_verified_at`
	if s.RequireTenantScope {
		columns += `, tenant_id`
		args = append(args, tenantID)
	}
	query := `INSERT INTO users (` + columns + `) VALUES (` + s.bindList(len(args)) + `)`
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
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at, email_verified_at FROM users WHERE id = ` + s.bind(1)
	args := []any{id}
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.bind(2)
		args = append(args, tenantID)
	}

	var u auth.User
	var roleStr string
	var activeVal any
	var createdAtVal, updatedAtVal, verifiedAtVal any

	err = s.within(ctx, func(exec database.Executor) error {
		return exec.QueryRowContext(ctx, query, args...).Scan(&u.ID, &u.Email, &u.PasswordHash, &roleStr, &activeVal, &createdAtVal, &updatedAtVal, &verifiedAtVal)
	})
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
	u.EmailVerifiedAt = parseOptionalTime(verifiedAtVal)

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
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at, email_verified_at FROM users WHERE email = ` + s.bind(1)
	args := []any{auth.NormalizeEmail(email)}
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.bind(2)
		args = append(args, tenantID)
	}

	var u auth.User
	var roleStr string
	var activeVal any
	var createdAtVal, updatedAtVal, verifiedAtVal any

	err = s.within(ctx, func(exec database.Executor) error {
		return exec.QueryRowContext(ctx, query, args...).Scan(&u.ID, &u.Email, &u.PasswordHash, &roleStr, &activeVal, &createdAtVal, &updatedAtVal, &verifiedAtVal)
	})
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
	u.EmailVerifiedAt = parseOptionalTime(verifiedAtVal)

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
		query += " WHERE tenant_id = " + s.bind(1)
		args = append(args, tenantID)
	}
	var count int64
	err = s.within(ctx, func(exec database.Executor) error {
		return exec.QueryRowContext(ctx, query, args...).Scan(&count)
	})
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
	activeArg := any(active)
	if s.dialect != database.Postgres {
		activeArg = boolToSQLite(active)
	}
	query := `UPDATE users SET active = ` + s.bind(1) + `, updated_at = ` + s.bind(2) + ` WHERE id = ` + s.bind(3)
	args := []any{activeArg, s.dbTime(time.Now().UTC()), id}
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.bind(4)
		args = append(args, tenantID)
	}
	var result sql.Result
	err = s.within(ctx, func(exec database.Executor) error {
		var execErr error
		result, execErr = exec.ExecContext(ctx, query, args...)
		return execErr
	})
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

func (s *UserStore) bind(position int) string {
	if s.dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func (s *UserStore) bindList(count int) string {
	values := make([]string, count)
	for i := range values {
		values[i] = s.bind(i + 1)
	}
	return strings.Join(values, ",")
}

func (s *UserStore) dbTime(value time.Time) any {
	value = value.UTC()
	if s.dialect == database.SQLite {
		return value.Format(time.RFC3339Nano)
	}
	return value
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

func parseOptionalTime(v any) *time.Time {
	if v == nil {
		return nil
	}
	parsed := parseTime(v)
	if parsed.IsZero() {
		return nil
	}
	return &parsed
}

func (s *UserStore) SetEmailVerifiedTx(ctx context.Context, tx database.Executor, userID string, verifiedAt time.Time) error {
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE users SET email_verified_at = COALESCE(email_verified_at, ` + s.bind(1) + `), updated_at = ` + s.bind(2) + ` WHERE id = ` + s.bind(3)
	args := []any{s.dbTime(verifiedAt), s.dbTime(verifiedAt), userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.bind(4)
		args = append(args, tenantID)
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err == nil && count != 1 {
		return auth.ErrUserNotFound
	}
	return nil
}

func (s *UserStore) LockEmailVerificationUserTx(ctx context.Context, tx database.Executor, userID string) (string, *time.Time, error) {
	if err := requireUserStore(s); err != nil {
		return "", nil, err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return "", nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return "", nil, err
	}
	if s.dialect == database.SQLite {
		query := `UPDATE users SET updated_at=updated_at WHERE id=` + s.bind(1)
		args := []any{userID}
		if s.RequireTenantScope {
			query += ` AND tenant_id=` + s.bind(2)
			args = append(args, tenantID)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return "", nil, err
		}
	}
	query := `SELECT email,email_verified_at FROM users WHERE id=` + s.bind(1)
	args := []any{userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id=` + s.bind(2)
		args = append(args, tenantID)
	}
	if s.dialect != database.SQLite {
		query += ` FOR UPDATE`
	}
	var email string
	var verified any
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&email, &verified); err != nil {
		return "", nil, auth.ErrUserNotFound
	}
	return email, parseOptionalTime(verified), nil
}

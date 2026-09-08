package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type UserStore struct {
	db      *sql.DB
	dialect database.Dialect
}

func NewUserStore(db *sql.DB, dialect database.Dialect) *UserStore {
	return &UserStore{db: db, dialect: dialect}
}

func (s *UserStore) Create(ctx context.Context, u *auth.User) error {
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	query := `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if s.dialect == database.SQLite {
		query = `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at)
                 VALUES (?, ?, ?, ?, ?, ?, ?)`
	}

	var activeVal any = u.Active
	var createdAtVal any = u.CreatedAt
	var updatedAtVal any = u.UpdatedAt
	if s.dialect == database.SQLite {
		if u.Active {
			activeVal = 1
		} else {
			activeVal = 0
		}
		createdAtVal = u.CreatedAt.Format(time.RFC3339Nano)
		updatedAtVal = u.UpdatedAt.Format(time.RFC3339Nano)
	}

	_, err := s.db.ExecContext(ctx, query, u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), activeVal, createdAtVal, updatedAtVal)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (s *UserStore) CreateUserTx(ctx context.Context, tx database.Executor, u *auth.User) error {
	now := time.Now().UTC()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	query := `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	args := []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), u.Active, u.CreatedAt, u.UpdatedAt}
	if s.dialect == database.SQLite {
		query = `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at) VALUES (?,?,?,?,?,?,?)`
		active := 0
		if u.Active {
			active = 1
		}
		args = []any{u.ID, auth.NormalizeEmail(u.Email), u.PasswordHash, string(u.Role), active, u.CreatedAt.Format(time.RFC3339Nano), u.UpdatedAt.Format(time.RFC3339Nano)}
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*auth.User, error) {
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE id = $1`
	if s.dialect == database.SQLite {
		query = `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE id = ?`
	}

	var u auth.User
	var roleStr string
	var activeVal any
	var createdAtVal, updatedAtVal any

	err := s.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &roleStr, &activeVal, &createdAtVal, &updatedAtVal)
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
	query := `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE email = $1`
	if s.dialect == database.SQLite {
		query = `SELECT id, email, password_hash, role, active, created_at, updated_at FROM users WHERE email = ?`
	}

	var u auth.User
	var roleStr string
	var activeVal any
	var createdAtVal, updatedAtVal any

	err := s.db.QueryRowContext(ctx, query, auth.NormalizeEmail(email)).Scan(&u.ID, &u.Email, &u.PasswordHash, &roleStr, &activeVal, &createdAtVal, &updatedAtVal)
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
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (s *UserStore) SetActive(ctx context.Context, id string, active bool) error {
	query := `UPDATE users SET active = $1, updated_at = $2 WHERE id = $3`
	args := []any{active, time.Now().UTC(), id}
	if s.dialect == database.SQLite {
		query = `UPDATE users SET active = ?, updated_at = ? WHERE id = ?`
		args = []any{boolToSQLite(active), time.Now().UTC().Format(time.RFC3339Nano), id}
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

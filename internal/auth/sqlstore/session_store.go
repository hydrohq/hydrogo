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

type SessionStore struct {
	db                 *sql.DB
	dialect            database.Dialect
	RequireTenantScope bool
}

func (s *SessionStore) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func NewSessionStore(db *sql.DB, dialect database.Dialect) *SessionStore {
	return &SessionStore{db: db, dialect: dialect}
}

func (s *SessionStore) Create(ctx context.Context, sess *auth.Session) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	if sess == nil {
		return errors.New("session is required")
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	if sess.LastSeenAt.IsZero() {
		sess.LastSeenAt = now
	}

	query := `INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address`
	args := []any{sess.ID, sess.UserID, sess.TokenHash, sess.ExpiresAt, sess.CreatedAt, sess.LastSeenAt, sess.UserAgent, sess.IPAddress}
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
		query = `INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address`
		args = []any{sess.ID, sess.UserID, sess.TokenHash, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano), sess.LastSeenAt.Format(time.RFC3339Nano), sess.UserAgent, sess.IPAddress}
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
		return fmt.Errorf("failed to insert session: %w", err)
	}
	return nil
}

func (s *SessionStore) GetByTokenHash(ctx context.Context, tokenHash string) (*auth.Session, error) {
	if err := requireSessionStore(s); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address
              FROM sessions WHERE token_hash = $1`
	args := []any{tokenHash}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address
                 FROM sessions WHERE token_hash = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}

	var sess auth.Session
	var expiresAtVal, createdAtVal, lastSeenAtVal any

	err = s.db.QueryRowContext(ctx, query, args...).Scan(
		&sess.ID, &sess.UserID, &sess.TokenHash,
		&expiresAtVal, &createdAtVal, &lastSeenAtVal,
		&sess.UserAgent, &sess.IPAddress,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	sess.ExpiresAt = parseTime(expiresAtVal)
	sess.CreatedAt = parseTime(createdAtVal)
	sess.LastSeenAt = parseTime(lastSeenAtVal)

	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, auth.ErrSessionExpired
	}

	return &sess, nil
}

func (s *SessionStore) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE token_hash = $1`
	args := []any{tokenHash}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE token_hash = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *SessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE user_id = $1`
	args := []any{userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE user_id = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (s *SessionStore) DeleteByUserIDTx(ctx context.Context, tx database.Executor, userID string) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE user_id = $1`
	args := []any{userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE user_id = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s *SessionStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	if err := requireSessionStore(s); err != nil {
		return 0, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return 0, err
	}
	query := `DELETE FROM sessions WHERE expires_at <= $1`
	var arg any = now.UTC()
	args := []any{arg}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE expires_at <= ?`
		arg = now.UTC().Format(time.RFC3339Nano)
		args = []any{arg}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return res.RowsAffected()
}

func (s *SessionStore) Touch(ctx context.Context, tokenHash string, lastSeen time.Time) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE sessions SET last_seen_at = $1 WHERE token_hash = $2`
	var arg any = lastSeen.UTC()
	args := []any{arg, tokenHash}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $3`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`
		arg = lastSeen.UTC().Format(time.RFC3339Nano)
		args = []any{arg, tokenHash}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

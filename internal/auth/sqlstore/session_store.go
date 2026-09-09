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

type SessionStore struct {
	db      *sql.DB
	dialect database.Dialect
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
	now := time.Now().UTC()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	if sess.LastSeenAt.IsZero() {
		sess.LastSeenAt = now
	}

	query := `INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	if s.dialect == database.SQLite {
		query = `INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address)
                 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	}

	var expiresAtVal any = sess.ExpiresAt
	var createdAtVal any = sess.CreatedAt
	var lastSeenAtVal any = sess.LastSeenAt
	if s.dialect == database.SQLite {
		expiresAtVal = sess.ExpiresAt.Format(time.RFC3339Nano)
		createdAtVal = sess.CreatedAt.Format(time.RFC3339Nano)
		lastSeenAtVal = sess.LastSeenAt.Format(time.RFC3339Nano)
	}

	_, err := s.db.ExecContext(ctx, query, sess.ID, sess.UserID, sess.TokenHash, expiresAtVal, createdAtVal, lastSeenAtVal, sess.UserAgent, sess.IPAddress)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}
	return nil
}

func (s *SessionStore) GetByTokenHash(ctx context.Context, tokenHash string) (*auth.Session, error) {
	if err := requireSessionStore(s); err != nil {
		return nil, err
	}
	query := `SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address
              FROM sessions WHERE token_hash = $1`
	if s.dialect == database.SQLite {
		query = `SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address
                 FROM sessions WHERE token_hash = ?`
	}

	var sess auth.Session
	var expiresAtVal, createdAtVal, lastSeenAtVal any

	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
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
	query := `DELETE FROM sessions WHERE token_hash = $1`
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE token_hash = ?`
	}
	_, err := s.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *SessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE user_id = $1`
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE user_id = ?`
	}
	_, err := s.db.ExecContext(ctx, query, userID)
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
	query := `DELETE FROM sessions WHERE user_id = $1`
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE user_id = ?`
	}
	_, err := tx.ExecContext(ctx, query, userID)
	return err
}

func (s *SessionStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	if err := requireSessionStore(s); err != nil {
		return 0, err
	}
	query := `DELETE FROM sessions WHERE expires_at <= $1`
	var arg any = now.UTC()
	if s.dialect == database.SQLite {
		query = `DELETE FROM sessions WHERE expires_at <= ?`
		arg = now.UTC().Format(time.RFC3339Nano)
	}
	res, err := s.db.ExecContext(ctx, query, arg)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return res.RowsAffected()
}

func (s *SessionStore) Touch(ctx context.Context, tokenHash string, lastSeen time.Time) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	query := `UPDATE sessions SET last_seen_at = $1 WHERE token_hash = $2`
	var arg any = lastSeen.UTC()
	if s.dialect == database.SQLite {
		query = `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`
		arg = lastSeen.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, query, arg, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

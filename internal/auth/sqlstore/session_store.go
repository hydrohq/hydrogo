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

// within ensures the tenant GUC and the session query share one PostgreSQL
// transaction/physical connection.
func (s *SessionStore) within(ctx context.Context, fn func(database.Executor) error) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	return database.NewTenantTransactor(s.db, s.dialect, s.RequireTenantScope).Within(ctx, fn)
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address`
		args = []any{sess.ID, sess.UserID, sess.TokenHash, sess.ExpiresAt, sess.CreatedAt, sess.LastSeenAt, sess.UserAgent, sess.IPAddress}
		if s.dialect == database.SQLite {
			args = []any{sess.ID, sess.UserID, sess.TokenHash, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano), sess.LastSeenAt.Format(time.RFC3339Nano), sess.UserAgent, sess.IPAddress}
		}
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

	return s.within(ctx, func(exec database.Executor) error {
		_, execErr := exec.ExecContext(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("failed to insert session: %w", execErr)
		}
		return nil
	})
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `SELECT id, user_id, token_hash, expires_at, created_at, last_seen_at, user_agent, ip_address
                 FROM sessions WHERE token_hash = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}

	var sess auth.Session
	var expiresAtVal, createdAtVal, lastSeenAtVal any

	err = s.within(ctx, func(exec database.Executor) error {
		return exec.QueryRowContext(ctx, query, args...).Scan(
			&sess.ID, &sess.UserID, &sess.TokenHash,
			&expiresAtVal, &createdAtVal, &lastSeenAtVal,
			&sess.UserAgent, &sess.IPAddress,
		)
	})
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `DELETE FROM sessions WHERE token_hash = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, execErr := exec.ExecContext(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("failed to delete session: %w", execErr)
		}
		return nil
	})
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `DELETE FROM sessions WHERE user_id = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, execErr := exec.ExecContext(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("failed to delete user sessions: %w", execErr)
		}
		return nil
	})
}

func (s *SessionStore) DeleteOtherByUserID(ctx context.Context, userID, currentTokenHash string) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2`
	args := []any{userID, currentTokenHash}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $3`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `DELETE FROM sessions WHERE user_id = ? AND token_hash <> ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, err := exec.ExecContext(ctx, query, args...)
		return err
	})
}

func (s *SessionStore) DeleteByIDAndUserID(ctx context.Context, userID, sessionID string) error {
	if err := requireSessionStore(s); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM sessions WHERE user_id = $1 AND id = $2`
	args := []any{userID, sessionID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $3`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `DELETE FROM sessions WHERE user_id = ? AND id = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		result, err := exec.ExecContext(ctx, query, args...)
		if err != nil {
			return err
		}
		if affected, err := result.RowsAffected(); err == nil && affected == 0 {
			return auth.ErrSessionNotFound
		}
		return nil
	})
}

func (s *SessionStore) ListByUserID(ctx context.Context, userID string, now time.Time) ([]auth.Session, error) {
	if err := requireSessionStore(s); err != nil {
		return nil, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	args := []any{userID, now.UTC()}
	var q string
	switch s.dialect {
	case database.SQLite:
		args[1] = now.UTC().Format(time.RFC3339Nano)
		q = "SELECT id,user_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM sessions WHERE user_id=?1 AND expires_at>=?2"
		if s.RequireTenantScope {
			q += " AND tenant_id=?3"
			args = append(args, tenantID)
		}
	case database.MySQL:
		q = "SELECT id,user_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM sessions WHERE user_id=? AND expires_at>=?"
		if s.RequireTenantScope {
			q += " AND tenant_id=?"
			args = append(args, tenantID)
		}
	default:
		q = "SELECT id,user_id,token_hash,expires_at,created_at,last_seen_at,user_agent,ip_address FROM sessions WHERE user_id=$1 AND expires_at>=$2"
		if s.RequireTenantScope {
			q += " AND tenant_id=$3"
			args = append(args, tenantID)
		}
	}
	q += " ORDER BY last_seen_at DESC,id ASC LIMIT 100"
	var sessions []auth.Session
	err = s.within(ctx, func(exec database.Executor) error {
		queryer, ok := exec.(database.QueryExecutor)
		if !ok {
			return errors.New("session query is unsupported")
		}
		rows, e := queryer.QueryContext(ctx, q, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var x auth.Session
			var expires, created, last any
			if e := rows.Scan(&x.ID, &x.UserID, &x.TokenHash, &expires, &created, &last, &x.UserAgent, &x.IPAddress); e != nil {
				return e
			}
			x.ExpiresAt = parseTime(expires)
			x.CreatedAt = parseTime(created)
			x.LastSeenAt = parseTime(last)
			sessions = append(sessions, x)
		}
		return rows.Err()
	})
	return sessions, err
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `DELETE FROM sessions WHERE expires_at <= ?`
		if s.dialect == database.SQLite {
			arg = now.UTC().Format(time.RFC3339Nano)
		}
		args = []any{arg}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	var res sql.Result
	err = s.within(ctx, func(exec database.Executor) error {
		var execErr error
		res, execErr = exec.ExecContext(ctx, query, args...)
		return execErr
	})
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
	if s.dialect == database.SQLite || s.dialect == database.MySQL {
		query = `UPDATE sessions SET last_seen_at = ? WHERE token_hash = ?`
		if s.dialect == database.SQLite {
			arg = lastSeen.UTC().Format(time.RFC3339Nano)
		}
		args = []any{arg, tokenHash}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, execErr := exec.ExecContext(ctx, query, args...)
		if execErr != nil {
			return fmt.Errorf("failed to touch session: %w", execErr)
		}
		return nil
	})
}

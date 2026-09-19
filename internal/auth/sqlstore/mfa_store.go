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

func (s *UserStore) PutFactor(ctx context.Context, tx database.Executor, userID string, ciphertext []byte, enabled bool) error {
	if err := requireUserStore(s); err != nil {
		return err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	query := `INSERT INTO user_mfa (user_id, secret_ciphertext, enabled, created_at, updated_at, tenant_id) VALUES ($1, $2, $3, $4, $4, $5) ON CONFLICT (user_id) DO UPDATE SET secret_ciphertext = EXCLUDED.secret_ciphertext, enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at, tenant_id = EXCLUDED.tenant_id`
	args := []any{userID, ciphertext, enabled, now}
	if !s.RequireTenantScope {
		query = `INSERT INTO user_mfa (user_id, secret_ciphertext, enabled, created_at, updated_at) VALUES ($1, $2, $3, $4, $4) ON CONFLICT (user_id) DO UPDATE SET secret_ciphertext = EXCLUDED.secret_ciphertext, enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at`
	} else {
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `INSERT INTO user_mfa (user_id, secret_ciphertext, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET secret_ciphertext = excluded.secret_ciphertext, enabled = excluded.enabled, updated_at = excluded.updated_at`
		args = []any{userID, ciphertext, boolToSQLite(enabled), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}
		if s.RequireTenantScope {
			query = `INSERT INTO user_mfa (user_id, secret_ciphertext, enabled, created_at, updated_at, tenant_id) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(user_id) DO UPDATE SET secret_ciphertext = excluded.secret_ciphertext, enabled = excluded.enabled, updated_at = excluded.updated_at, tenant_id = excluded.tenant_id`
			args = append(args, tenantID)
		}
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) GetFactor(ctx context.Context, db database.Executor, userID string) ([]byte, bool, error) {
	if err := requireUserStore(s); err != nil {
		return nil, false, err
	}
	if err := requireAuthExecutor(db); err != nil {
		return nil, false, err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, false, err
	}
	query := `SELECT secret_ciphertext, enabled FROM user_mfa WHERE user_id = $1`
	args := []any{userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $2`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `SELECT secret_ciphertext, enabled FROM user_mfa WHERE user_id = ?`
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
		}
	}
	var secret []byte
	var enabled any
	if err := db.QueryRowContext(ctx, query, args...).Scan(&secret, &enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, auth.ErrMFAUnavailable
		}
		return nil, false, err
	}
	return secret, parseActive(enabled), nil
}

func (s *UserStore) PutRecoveryCode(ctx context.Context, tx database.Executor, userID, id, codeHash string) error {
	if err := requireUserStore(s); err != nil {
		return err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `INSERT INTO mfa_recovery_codes (id, user_id, code_hash, tenant_id) VALUES ($1, $2, $3, $4)`
	args := []any{id, userID, codeHash, tenantID}
	if !s.RequireTenantScope {
		query = `INSERT INTO mfa_recovery_codes (id, user_id, code_hash) VALUES ($1, $2, $3)`
		args = args[:3]
	}
	if s.dialect == database.SQLite {
		query = `INSERT INTO mfa_recovery_codes (id, user_id, code_hash) VALUES (?, ?, ?)`
		args = []any{id, userID, codeHash}
		if s.RequireTenantScope {
			query = `INSERT INTO mfa_recovery_codes (id, user_id, code_hash, tenant_id) VALUES (?, ?, ?, ?)`
			args = append(args, tenantID)
		}
	}
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) ConsumeRecoveryCode(ctx context.Context, tx database.Executor, userID, codeHash string, usedAt time.Time) error {
	if err := requireUserStore(s); err != nil {
		return err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE mfa_recovery_codes SET used_at = $1 WHERE user_id = $2 AND code_hash = $3 AND used_at IS NULL`
	args := []any{usedAt, userID, codeHash}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $4`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `UPDATE mfa_recovery_codes SET used_at = ? WHERE user_id = ? AND code_hash = ? AND used_at IS NULL`
		args = []any{usedAt.Format(time.RFC3339Nano), userID, codeHash}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return fmt.Errorf("recovery code: %w", auth.ErrMFAInvalidCode)
	}
	return nil
}

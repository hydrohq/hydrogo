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

func (s *UserStore) UpdatePassword(ctx context.Context, tx database.Executor, userID, passwordHash string) error {
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
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	args := []any{passwordHash, time.Now().UTC(), userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $4`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
		args = []any{passwordHash, time.Now().UTC().Format(time.RFC3339Nano), userID}
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
		return auth.ErrUserNotFound
	}
	return nil
}

func (s *UserStore) UpdateEmail(ctx context.Context, tx database.Executor, userID, email string) error {
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
	query := `UPDATE users SET email = $1, email_verified_at = NULL, updated_at = $2 WHERE id = $3`
	args := []any{auth.NormalizeEmail(email), time.Now().UTC(), userID}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $4`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `UPDATE users SET email = ?, email_verified_at = NULL, updated_at = ? WHERE id = ?`
		args = []any{auth.NormalizeEmail(email), time.Now().UTC().Format(time.RFC3339Nano), userID}
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
		return auth.ErrUserNotFound
	}
	deleteTokens := `DELETE FROM auth_recovery_tokens WHERE user_id = ` + s.bind(1) + ` AND purpose = ` + s.bind(2)
	deleteArgs := []any{userID, auth.EmailVerificationPurpose}
	if s.RequireTenantScope {
		deleteTokens += ` AND tenant_id = ` + s.bind(3)
		deleteArgs = append(deleteArgs, tenantID)
	}
	if _, err := tx.ExecContext(ctx, deleteTokens, deleteArgs...); err != nil {
		return err
	}
	return nil
}

func (s *UserStore) CreateRecoveryToken(ctx context.Context, tx database.Executor, id, userID, purpose, tokenHash string, expiresAt, createdAt time.Time) error {
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
	query := `INSERT INTO auth_recovery_tokens (id, user_id, purpose, token_hash, expires_at, created_at`
	args := []any{id, userID, purpose, tokenHash, expiresAt, createdAt}
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
		query = `INSERT INTO auth_recovery_tokens (id, user_id, purpose, token_hash, expires_at, created_at`
		args = []any{id, userID, purpose, tokenHash, expiresAt.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano)}
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
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) ConsumeRecoveryToken(ctx context.Context, tx database.Executor, purpose, tokenHash string, now time.Time) (string, error) {
	if err := requireUserStore(s); err != nil {
		return "", err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return "", err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return "", err
	}
	query := `SELECT user_id FROM auth_recovery_tokens WHERE purpose = $1 AND token_hash = $2 AND consumed_at IS NULL AND expires_at > $3`
	argNow := any(now)
	args := []any{purpose, tokenHash, argNow}
	if s.RequireTenantScope {
		query += ` AND tenant_id = $4`
		args = append(args, tenantID)
	}
	if s.dialect == database.SQLite {
		query = `SELECT user_id FROM auth_recovery_tokens WHERE purpose = ? AND token_hash = ? AND consumed_at IS NULL AND expires_at > ?`
		argNow = now.Format(time.RFC3339Nano)
		args = []any{purpose, tokenHash, argNow}
		if s.RequireTenantScope {
			query += ` AND tenant_id = ?`
			args = append(args, tenantID)
		}
	}
	var userID string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", auth.ErrRecoveryTokenInvalid
		}
		return "", err
	}
	update := `UPDATE auth_recovery_tokens SET consumed_at = $1 WHERE purpose = $2 AND token_hash = $3 AND consumed_at IS NULL`
	updateArgs := []any{now, purpose, tokenHash}
	if s.RequireTenantScope {
		update += ` AND tenant_id = $4`
		updateArgs = append(updateArgs, tenantID)
	}
	if s.dialect == database.SQLite {
		update = `UPDATE auth_recovery_tokens SET consumed_at = ? WHERE purpose = ? AND token_hash = ? AND consumed_at IS NULL`
		updateArgs = []any{now.Format(time.RFC3339Nano), purpose, tokenHash}
		if s.RequireTenantScope {
			update += ` AND tenant_id = ?`
			updateArgs = append(updateArgs, tenantID)
		}
	}
	result, err := tx.ExecContext(ctx, update, updateArgs...)
	if err != nil {
		return "", err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return "", fmt.Errorf("consume recovery token: %w", auth.ErrRecoveryTokenInvalid)
	}
	return userID, nil
}

func (s *UserStore) FindEmailVerificationOwnerTx(ctx context.Context, tx database.Executor, tokenHash string, now time.Time) (string, error) {
	if err := requireUserStore(s); err != nil {
		return "", err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return "", err
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return "", err
	}
	query := `SELECT user_id FROM auth_recovery_tokens WHERE purpose=` + s.bind(1) + ` AND token_hash=` + s.bind(2) + ` AND consumed_at IS NULL AND expires_at>` + s.bind(3)
	args := []any{auth.EmailVerificationPurpose, tokenHash, s.dbTime(now)}
	if s.RequireTenantScope {
		query += ` AND tenant_id=` + s.bind(4)
		args = append(args, tenantID)
	}
	var userID string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&userID); err != nil {
		return "", auth.ErrEmailVerificationInvalid
	}
	return userID, nil
}

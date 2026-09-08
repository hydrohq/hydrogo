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
	query := `UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`
	args := []any{passwordHash, time.Now().UTC(), userID}
	if s.dialect == database.SQLite {
		query = `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
		args = []any{passwordHash, time.Now().UTC().Format(time.RFC3339Nano), userID}
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

func (s *UserStore) CreateRecoveryToken(ctx context.Context, tx database.Executor, id, userID, purpose, tokenHash string, expiresAt, createdAt time.Time) error {
	query := `INSERT INTO auth_recovery_tokens (id, user_id, purpose, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
	args := []any{id, userID, purpose, tokenHash, expiresAt, createdAt}
	if s.dialect == database.SQLite {
		query = `INSERT INTO auth_recovery_tokens (id, user_id, purpose, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?)`
		args = []any{id, userID, purpose, tokenHash, expiresAt.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano)}
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) ConsumeRecoveryToken(ctx context.Context, tx database.Executor, purpose, tokenHash string, now time.Time) (string, error) {
	query := `SELECT user_id FROM auth_recovery_tokens WHERE purpose = $1 AND token_hash = $2 AND consumed_at IS NULL AND expires_at > $3`
	argNow := any(now)
	if s.dialect == database.SQLite {
		query = `SELECT user_id FROM auth_recovery_tokens WHERE purpose = ? AND token_hash = ? AND consumed_at IS NULL AND expires_at > ?`
		argNow = now.Format(time.RFC3339Nano)
	}
	var userID string
	if err := tx.QueryRowContext(ctx, query, purpose, tokenHash, argNow).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", auth.ErrRecoveryTokenInvalid
		}
		return "", err
	}
	update := `UPDATE auth_recovery_tokens SET consumed_at = $1 WHERE purpose = $2 AND token_hash = $3 AND consumed_at IS NULL`
	args := []any{now, purpose, tokenHash}
	if s.dialect == database.SQLite {
		update = `UPDATE auth_recovery_tokens SET consumed_at = ? WHERE purpose = ? AND token_hash = ? AND consumed_at IS NULL`
		args = []any{now.Format(time.RFC3339Nano), purpose, tokenHash}
	}
	result, err := tx.ExecContext(ctx, update, args...)
	if err != nil {
		return "", err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return "", fmt.Errorf("consume recovery token: %w", auth.ErrRecoveryTokenInvalid)
	}
	return userID, nil
}

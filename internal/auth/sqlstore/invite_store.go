package sqlstore

import (
	"context"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func (s *UserStore) CreateInvite(ctx context.Context, tx database.Executor, id, email string, role auth.Role, tokenHash string, expiresAt, createdAt time.Time) error {
	if err := requireUserStore(s); err != nil {
		return err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return err
	}
	query := `INSERT INTO auth_invites (id,email,role,token_hash,expires_at,created_at) VALUES ($1,$2,$3,$4,$5,$6)`
	args := []any{id, email, string(role), tokenHash, expiresAt, createdAt}
	if s.dialect == database.SQLite {
		query = `INSERT INTO auth_invites (id,email,role,token_hash,expires_at,created_at) VALUES (?,?,?,?,?,?)`
		args[4] = expiresAt.Format(time.RFC3339Nano)
		args[5] = createdAt.Format(time.RFC3339Nano)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (s *UserStore) ConsumeInvite(ctx context.Context, tx database.Executor, tokenHash string, now time.Time) (string, auth.Role, error) {
	if err := requireUserStore(s); err != nil {
		return "", "", err
	}
	if err := requireAuthExecutor(tx); err != nil {
		return "", "", err
	}
	query := `SELECT email,role FROM auth_invites WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > $2`
	args := []any{tokenHash, now}
	if s.dialect == database.SQLite {
		query = `SELECT email,role FROM auth_invites WHERE token_hash = ? AND consumed_at IS NULL AND expires_at > ?`
		args[1] = now.Format(time.RFC3339Nano)
	}
	var email, role string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&email, &role); err != nil {
		return "", "", auth.ErrInviteInvalid
	}
	update := `UPDATE auth_invites SET consumed_at = $1 WHERE token_hash = $2 AND consumed_at IS NULL`
	updateArgs := []any{now, tokenHash}
	if s.dialect == database.SQLite {
		update = `UPDATE auth_invites SET consumed_at = ? WHERE token_hash = ? AND consumed_at IS NULL`
		updateArgs[0] = now.Format(time.RFC3339Nano)
	}
	result, err := tx.ExecContext(ctx, update, updateArgs...)
	if err != nil {
		return "", "", err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return "", "", fmt.Errorf("consume invite: %w", auth.ErrInviteInvalid)
	}
	return email, auth.Role(role), nil
}

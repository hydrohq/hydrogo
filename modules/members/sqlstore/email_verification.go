package sqlstore

import (
	"context"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

func (s Store) CreateEmailVerificationToken(ctx context.Context, tx database.Executor, token members.VerificationToken) error {
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return err
		}
		tenantID = scope.TenantID
	}
	expiresAt, createdAt := any(token.ExpiresAt), any(token.CreatedAt)
	if s.Dialect == database.SQLite {
		expiresAt = token.ExpiresAt.UTC().Format(time.RFC3339Nano)
		createdAt = token.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	consumedAt := any(token.CreatedAt)
	if s.Dialect == database.SQLite {
		consumedAt = token.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	cutoff := token.CreatedAt.UTC().Add(-30 * 24 * time.Hour)
	cutoffValue := any(cutoff)
	if s.Dialect == database.SQLite {
		cutoffValue = cutoff.Format(time.RFC3339Nano)
	}
	cleanup := "DELETE FROM member_email_verification_tokens WHERE tenant_id = " + s.ph(1) + " AND ((consumed_at IS NOT NULL AND consumed_at < " + s.ph(2) + ") OR expires_at < " + s.ph(3) + ")"
	if _, err := tx.ExecContext(ctx, cleanup, tenantID, cutoffValue, cutoffValue); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE member_email_verification_tokens SET consumed_at = "+s.ph(1)+" WHERE member_id = "+s.ph(2)+" AND tenant_id = "+s.ph(3)+" AND consumed_at IS NULL", consumedAt, token.MemberID, tenantID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO member_email_verification_tokens (id,member_id,token_hash,expires_at,created_at,tenant_id) VALUES ("+s.ph(1)+","+s.ph(2)+","+s.ph(3)+","+s.ph(4)+","+s.ph(5)+","+s.ph(6)+")", token.ID, token.MemberID, token.Hash, expiresAt, createdAt, tenantID)
	return err
}

func (s Store) ConsumeEmailVerificationToken(ctx context.Context, tx database.Executor, hash string, now time.Time) error {
	clause, scopeID, err := s.tenantClause(ctx, 2)
	if err != nil {
		return err
	}
	nowValue := any(now)
	if s.Dialect == database.SQLite {
		nowValue = now.UTC().Format(time.RFC3339Nano)
	}
	args := []any{hash}
	if clause != "" {
		args = append(args, scopeID)
	}
	var memberID string
	var query string
	// The expiry placeholder follows the tenant discriminator when scoped.
	if clause == "" {
		query = "SELECT member_id FROM member_email_verification_tokens WHERE token_hash = " + s.ph(1) + " AND consumed_at IS NULL AND expires_at > " + s.ph(2)
		args = append(args, nowValue)
	} else {
		query = "SELECT member_id FROM member_email_verification_tokens WHERE token_hash = " + s.ph(1) + " AND consumed_at IS NULL AND tenant_id = " + s.ph(2) + " AND expires_at > " + s.ph(3)
		args = append(args, nowValue)
	}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&memberID); err != nil {
		return err
	}
	updateArgs := []any{nowValue, hash}
	update := "UPDATE member_email_verification_tokens SET consumed_at = " + s.ph(1) + " WHERE token_hash = " + s.ph(2) + " AND consumed_at IS NULL"
	if clause != "" {
		update += " AND tenant_id = " + s.ph(3)
		updateArgs = append(updateArgs, scopeID)
	}
	result, err := tx.ExecContext(ctx, update, updateArgs...)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return members.ErrMemberNotFound
	}
	verifiedAt, updatedAt := any(now), any(now)
	if s.Dialect == database.SQLite {
		verifiedAt, updatedAt = now.UTC().Format(time.RFC3339Nano), now.UTC().Format(time.RFC3339Nano)
	}
	memberClause, memberTenant, err := s.tenantClause(ctx, 3)
	if err != nil {
		return err
	}
	memberArgs := []any{verifiedAt, updatedAt, memberID}
	if memberClause != "" {
		memberArgs = append(memberArgs, memberTenant)
	}
	memberUpdate := "UPDATE members SET email_verified_at = " + s.ph(1) + ", updated_at = " + s.ph(2) + " WHERE id = " + s.ph(3) + " AND email_verified_at IS NULL" + memberClause
	result, err = tx.ExecContext(ctx, memberUpdate, memberArgs...)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return errors.New("member verification has already been completed")
	}
	return nil
}

var _ members.EmailVerificationTokens = Store{}

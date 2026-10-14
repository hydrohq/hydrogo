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
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

// OAuthStore links verified social identities to member accounts. It is the
// members-side counterpart of internal/auth/sqlstore.OAuthIdentityStore, but
// unlike that admin-only resolver it auto-provisions a new active member the
// first time a provider identity has no existing match: find by identity,
// else by email (linking it), else create one. Members are self-service
// accounts, so there is no role to check and no separate invitation step.
type OAuthStore struct {
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s OAuthStore) bind(position int) string {
	switch s.Dialect {
	case database.Postgres:
		return fmt.Sprintf("$%d", position)
	case database.SQLite:
		return fmt.Sprintf("?%d", position)
	default:
		return "?"
	}
}

func (s OAuthStore) ResolveOAuth(ctx context.Context, identity auth.OAuthIdentity) (*members.Member, error) {
	if err := database.RequireExecutor(s.DB); err != nil {
		return nil, err
	}
	if !identity.EmailVerified || identity.Provider == "" || identity.Subject == "" || identity.Email == "" {
		return nil, members.ErrInvalidCredentials
	}
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return nil, err
		}
		tenantID = scope.TenantID
	}
	var member *members.Member
	err := database.NewTenantTransactor(s.DB, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
		query := `SELECT m.id,m.email,m.password_hash,m.active,m.email_verified_at,m.last_login_at,m.created_at,m.updated_at FROM member_oauth_identities i JOIN members m ON m.id=i.member_id WHERE i.provider=` + s.bind(1) + ` AND i.subject=` + s.bind(2) + ` AND i.tenant_id=` + s.bind(3) + ` AND m.tenant_id=` + s.bind(3)
		identityArgs := []any{identity.Provider, identity.Subject, tenantID}
		if s.Dialect == database.MySQL {
			identityArgs = append(identityArgs, tenantID)
		}
		var err error
		member, err = scanMemberOAuthRow(tx.QueryRowContext(ctx, query, identityArgs...))
		if err == nil {
			if !member.Active {
				return members.ErrMemberInactive
			}
			return s.markEmailVerified(ctx, tx, member, identity, tenantID)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		query = `SELECT id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at FROM members WHERE lower(email)=lower(` + s.bind(1) + `) AND tenant_id=` + s.bind(2)
		member, err = scanMemberOAuthRow(tx.QueryRowContext(ctx, query, identity.Email, tenantID))
		switch {
		case errors.Is(err, sql.ErrNoRows):
			member, err = s.provisionMember(ctx, tx, identity, tenantID)
			if err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			if !member.Active {
				return members.ErrMemberInactive
			}
			if err := s.markEmailVerified(ctx, tx, member, identity, tenantID); err != nil {
				return err
			}
		}
		linkQuery := `INSERT INTO member_oauth_identities (provider,subject,member_id,tenant_id,created_at) VALUES (` + s.bind(1) + `,` + s.bind(2) + `,` + s.bind(3) + `,` + s.bind(4) + `,CURRENT_TIMESTAMP)`
		if _, err := tx.ExecContext(ctx, linkQuery, identity.Provider, identity.Subject, member.ID, tenantID); err != nil {
			return fmt.Errorf("link member social account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return member, nil
}

func (s OAuthStore) provisionMember(ctx context.Context, tx database.Executor, identity auth.OAuthIdentity, tenantID string) (*members.Member, error) {
	now := time.Now().UTC()
	member := &members.Member{ID: newMemberOAuthID(), Email: auth.NormalizeEmail(identity.Email), PasswordHash: "", Active: true, EmailVerifiedAt: &now, CreatedAt: now, UpdatedAt: now}
	var createdAtVal, updatedAtVal, verifiedAtVal any = now, now, now
	if s.Dialect == database.SQLite {
		createdAtVal, updatedAtVal, verifiedAtVal = now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)
	}
	query := `INSERT INTO members (id,email,password_hash,active,email_verified_at,created_at,updated_at) VALUES (` + s.bind(1) + `,` + s.bind(2) + `,` + s.bind(3) + `,` + s.bind(4) + `,` + s.bind(5) + `,` + s.bind(6) + `,` + s.bind(7) + `)`
	args := []any{member.ID, member.Email, member.PasswordHash, member.Active, verifiedAtVal, createdAtVal, updatedAtVal}
	if s.RequireTenantScope {
		query = `INSERT INTO members (id,email,password_hash,active,email_verified_at,created_at,updated_at,tenant_id) VALUES (` + s.bind(1) + `,` + s.bind(2) + `,` + s.bind(3) + `,` + s.bind(4) + `,` + s.bind(5) + `,` + s.bind(6) + `,` + s.bind(7) + `,` + s.bind(8) + `)`
		args = append(args, tenantID)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		if database.IsUniqueViolation(err) {
			return nil, members.ErrMemberExists
		}
		return nil, fmt.Errorf("provision member from social login: %w", err)
	}
	return member, nil
}

func (s OAuthStore) markEmailVerified(ctx context.Context, tx database.Executor, member *members.Member, identity auth.OAuthIdentity, tenantID string) error {
	if member.EmailVerifiedAt != nil || auth.NormalizeEmail(member.Email) != auth.NormalizeEmail(identity.Email) {
		return nil
	}
	now := time.Now().UTC()
	query := `UPDATE members SET email_verified_at=COALESCE(email_verified_at,` + s.bind(1) + `) WHERE id=` + s.bind(2) + ` AND tenant_id=` + s.bind(3)
	// MySQL and PostgreSQL drivers marshal time.Time natively; only SQLite
	// (TEXT storage) needs the string form. See the same fix in
	// internal/auth/sqlstore/oauth.go's markOAuthEmailVerified: sending
	// RFC3339Nano unconditionally fails MySQL's DATETIME column (error 1292).
	var verifiedAt any = now
	if s.Dialect == database.SQLite {
		verifiedAt = now.Format(time.RFC3339Nano)
	}
	if _, err := tx.ExecContext(ctx, query, verifiedAt, member.ID, tenantID); err != nil {
		return err
	}
	member.EmailVerifiedAt = &now
	return nil
}

func scanMemberOAuthRow(row interface{ Scan(...any) error }) (*members.Member, error) {
	var m members.Member
	var active, verified, lastLogin, created, updated any
	if err := row.Scan(&m.ID, &m.Email, &m.PasswordHash, &active, &verified, &lastLogin, &created, &updated); err != nil {
		return nil, err
	}
	m.Active = parseBool(active)
	m.EmailVerifiedAt = parseTimePtr(verified)
	m.LastLoginAt = parseTimePtr(lastLogin)
	m.CreatedAt = parseTime(created)
	m.UpdatedAt = parseTime(updated)
	return &m, nil
}

func newMemberOAuthID() string {
	raw, _, err := auth.GenerateSessionToken()
	if err != nil {
		return "member-oauth-id-unavailable"
	}
	return raw[:32]
}

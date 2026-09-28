package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// OAuthIdentityStore links verified social identities to existing active
// Hydro administrators. It never grants a role or creates an account.
type OAuthIdentityStore struct {
	DB                 *sql.DB
	Dialect            database.Dialect
	RequireTenantScope bool
}

func (s OAuthIdentityStore) ResolveOAuth(ctx context.Context, identity auth.OAuthIdentity) (*auth.User, error) {
	if err := database.RequireExecutor(s.DB); err != nil {
		return nil, err
	}
	if !identity.EmailVerified || identity.Provider == "" || identity.Subject == "" || identity.Email == "" {
		return nil, auth.ErrInvalidCredentials
	}
	tenantID := ""
	if s.RequireTenantScope {
		scope, err := tenant.Require(ctx)
		if err != nil {
			return nil, err
		}
		tenantID = scope.TenantID
	}
	var user *auth.User
	err := database.NewTenantTransactor(s.DB, s.Dialect, s.RequireTenantScope).Within(ctx, func(tx database.Executor) error {
		query := `SELECT u.id,u.email,u.password_hash,u.role,u.active,u.created_at,u.updated_at FROM auth_oauth_identities i JOIN users u ON u.id=i.user_id WHERE i.provider=$1 AND i.subject=$2 AND i.tenant_id=$3 AND u.tenant_id=$3`
		if s.Dialect == database.SQLite {
			query = `SELECT u.id,u.email,u.password_hash,u.role,u.active,u.created_at,u.updated_at FROM auth_oauth_identities i JOIN users u ON u.id=i.user_id WHERE i.provider=?1 AND i.subject=?2 AND i.tenant_id=?3 AND u.tenant_id=?3`
		}
		var err error
		user, err = scanOAuthUser(tx.QueryRowContext(ctx, query, identity.Provider, identity.Subject, tenantID))
		if err == nil {
			if !user.Active {
				return auth.ErrUserInactive
			}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		query = `SELECT id,email,password_hash,role,active,created_at,updated_at FROM users WHERE lower(email)=lower($1) AND tenant_id=$2`
		if s.Dialect == database.SQLite {
			query = `SELECT id,email,password_hash,role,active,created_at,updated_at FROM users WHERE lower(email)=lower(?1) AND tenant_id=?2`
		}
		user, err = scanOAuthUser(tx.QueryRowContext(ctx, query, identity.Email, tenantID))
		if errors.Is(err, sql.ErrNoRows) {
			return auth.ErrUserNotFound
		}
		if err != nil {
			return err
		}
		if !user.Active {
			return auth.ErrUserInactive
		}
		query = `INSERT INTO auth_oauth_identities (provider,subject,user_id,tenant_id,created_at) VALUES ($1,$2,$3,$4,CURRENT_TIMESTAMP)`
		if s.Dialect == database.SQLite {
			query = `INSERT INTO auth_oauth_identities (provider,subject,user_id,tenant_id,created_at) VALUES (?1,?2,?3,?4,CURRENT_TIMESTAMP)`
		}
		if _, err := tx.ExecContext(ctx, query, identity.Provider, identity.Subject, user.ID, tenantID); err != nil {
			return fmt.Errorf("link social account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

type rowScanner interface{ Scan(...any) error }

func scanOAuthUser(row rowScanner) (*auth.User, error) {
	var user auth.User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Active, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

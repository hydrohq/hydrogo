package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// APITokenStore persists staff personal access tokens and API keys.
type APITokenStore struct {
	db                 *sql.DB
	dialect            database.Dialect
	RequireTenantScope bool
}

func NewAPITokenStore(db *sql.DB, dialect database.Dialect) *APITokenStore {
	return &APITokenStore{db: db, dialect: dialect}
}

const apiTokenColumns = `id, user_id, name, kind, token_hash, token_prefix, scopes, expires_at, last_used_at, created_at, revoked_at`

func (s *APITokenStore) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func (s *APITokenStore) within(ctx context.Context, fn func(database.Executor) error) error {
	if s == nil || s.db == nil {
		return database.ErrDatabaseRequired
	}
	return database.NewTenantTransactor(s.db, s.dialect, s.RequireTenantScope).Within(ctx, fn)
}

func (s *APITokenStore) bind(position int) string {
	if s.dialect == database.Postgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

func (s *APITokenStore) dbTime(value time.Time) any {
	value = value.UTC()
	if s.dialect == database.SQLite {
		return value.Format(time.RFC3339Nano)
	}
	return value
}

func (s *APITokenStore) dbOptionalTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return s.dbTime(*value)
}

func (s *APITokenStore) Create(ctx context.Context, token *auth.APIToken) error {
	if token == nil {
		return errors.New("api token is required")
	}
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	args := []any{token.ID, token.UserID, tenantID, token.Name, string(token.Kind), token.TokenHash, token.Prefix, strings.Join(token.Scopes, ","), s.dbOptionalTime(token.ExpiresAt), s.dbOptionalTime(token.LastUsedAt), s.dbTime(token.CreatedAt), s.dbOptionalTime(token.RevokedAt)}
	placeholders := make([]string, len(args))
	for i := range args {
		placeholders[i] = s.bind(i + 1)
	}
	query := `INSERT INTO api_tokens (id, user_id, tenant_id, name, kind, token_hash, token_prefix, scopes, expires_at, last_used_at, created_at, revoked_at) VALUES (` + strings.Join(placeholders, ", ") + `)`
	return s.within(ctx, func(exec database.Executor) error {
		if _, err := exec.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to insert api token: %w", err)
		}
		return nil
	})
}

func (s *APITokenStore) ListForUser(ctx context.Context, userID string) ([]auth.APIToken, error) {
	return s.list(ctx, `user_id = `+s.bind(1)+` AND kind = `+s.bind(2), []any{userID, string(auth.APITokenPersonal)})
}

func (s *APITokenStore) ListAPIKeys(ctx context.Context) ([]auth.APIToken, error) {
	return s.list(ctx, `kind = `+s.bind(1), []any{string(auth.APITokenKey)})
}

func (s *APITokenStore) list(ctx context.Context, where string, args []any) ([]auth.APIToken, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + apiTokenColumns + ` FROM api_tokens WHERE ` + where + ` AND revoked_at IS NULL`
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.bind(len(args)+1)
		args = append(args, tenantID)
	}
	query += ` ORDER BY created_at DESC, id`
	var items []auth.APIToken
	err = s.within(ctx, func(exec database.Executor) error {
		queryer, ok := exec.(database.QueryExecutor)
		if !ok {
			return database.ErrDatabaseRequired
		}
		rows, err := queryer.QueryContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to list api tokens: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			token, err := scanAPIToken(rows)
			if err != nil {
				return err
			}
			items = append(items, *token)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []auth.APIToken{}
	}
	return items, nil
}

func (s *APITokenStore) FindByHash(ctx context.Context, tokenHash string) (*auth.APIToken, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + apiTokenColumns + ` FROM api_tokens WHERE token_hash = ` + s.bind(1)
	args := []any{tokenHash}
	if s.RequireTenantScope {
		query += ` AND tenant_id = ` + s.bind(2)
		args = append(args, tenantID)
	}
	var token *auth.APIToken
	err = s.within(ctx, func(exec database.Executor) error {
		row := exec.QueryRowContext(ctx, query, args...)
		scanned, err := scanAPIToken(row)
		if err != nil {
			return err
		}
		token = scanned
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, auth.ErrAPITokenNotFound
		}
		return nil, err
	}
	return token, nil
}

func (s *APITokenStore) Revoke(ctx context.Context, id, userID string) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	args := []any{s.dbTime(time.Now().UTC()), id}
	query := `UPDATE api_tokens SET revoked_at = ` + s.bind(1) + ` WHERE id = ` + s.bind(2) + ` AND revoked_at IS NULL`
	if userID != "" {
		args = append(args, userID)
		query += ` AND user_id = ` + s.bind(len(args))
	}
	if s.RequireTenantScope {
		args = append(args, tenantID)
		query += ` AND tenant_id = ` + s.bind(len(args))
	}
	return s.within(ctx, func(exec database.Executor) error {
		result, err := exec.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to revoke api token: %w", err)
		}
		if affected, err := result.RowsAffected(); err == nil && affected == 0 {
			return auth.ErrAPITokenNotFound
		}
		return nil
	})
}

func (s *APITokenStore) TouchLastUsed(ctx context.Context, id string, at time.Time) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return err
	}
	args := []any{s.dbTime(at), id}
	query := `UPDATE api_tokens SET last_used_at = ` + s.bind(1) + ` WHERE id = ` + s.bind(2)
	if s.RequireTenantScope {
		args = append(args, tenantID)
		query += ` AND tenant_id = ` + s.bind(3)
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, err := exec.ExecContext(ctx, query, args...)
		return err
	})
}

func scanAPIToken(row rowScanner) (*auth.APIToken, error) {
	var token auth.APIToken
	var kind, scopes string
	var expiresAt, lastUsedAt, createdAt, revokedAt any
	if err := row.Scan(&token.ID, &token.UserID, &token.Name, &kind, &token.TokenHash, &token.Prefix, &scopes, &expiresAt, &lastUsedAt, &createdAt, &revokedAt); err != nil {
		return nil, err
	}
	token.Kind = auth.APITokenKind(kind)
	if scopes != "" {
		token.Scopes = strings.Split(scopes, ",")
	}
	token.ExpiresAt = parseOptionalTime(expiresAt)
	token.LastUsedAt = parseOptionalTime(lastUsedAt)
	token.CreatedAt = parseTime(createdAt)
	token.RevokedAt = parseOptionalTime(revokedAt)
	return &token, nil
}

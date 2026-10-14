package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func newAuditRecorder(db *sql.DB, dialect database.Dialect, requireTenantScope bool) auth.AuditRecorder {
	store := auditstore.Store{Dialect: dialect, RequireTenantScope: requireTenantScope}
	return auth.AuditRecorderFunc(func(ctx context.Context, action, targetID string) error {
		return database.NewTenantTransactor(db, dialect, requireTenantScope).Within(ctx, func(tx database.Executor) error {
			return store.Record(ctx, tx, newAuditID(), action, targetID)
		})
	})
}

func configuredMFARoles(values []string) []auth.Role {
	roles := make([]auth.Role, 0, len(values))
	for _, value := range values {
		roles = append(roles, auth.Role(value))
	}
	return roles
}

func newAuditID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("audit-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func configuredSocialOAuth(db *sql.DB, dialect database.Dialect, service *auth.AuthService, audit auth.AuditRecorder, secure, requireTenantScope bool) (*auth.SocialOAuthHandler, error) {
	providers := make(map[string]auth.OAuthProviderConfig)
	for _, name := range []string{"google", "github", "apple"} {
		prefix := "HYDRO_OAUTH_" + strings.ToUpper(name) + "_"
		clientID, secret, redirect := os.Getenv(prefix+"CLIENT_ID"), os.Getenv(prefix+"CLIENT_SECRET"), os.Getenv(prefix+"REDIRECT_URL")
		if clientID == "" && secret == "" && redirect == "" {
			continue
		}
		if clientID == "" || secret == "" || redirect == "" {
			return nil, fmt.Errorf("%s requires CLIENT_ID, CLIENT_SECRET and REDIRECT_URL", prefix)
		}
		provider, err := auth.DefaultOAuthProvider(name, clientID, secret, redirect)
		if err != nil {
			return nil, err
		}
		providers[name] = provider
	}
	if len(providers) == 0 {
		return nil, nil
	}
	key := []byte(os.Getenv("HYDRO_OAUTH_STATE_KEY"))
	if len(key) < 32 {
		return nil, errors.New("HYDRO_OAUTH_STATE_KEY must contain at least 32 bytes when a provider is enabled")
	}
	resolver := authstore.OAuthIdentityStore{DB: db, Dialect: dialect, RequireTenantScope: requireTenantScope}
	return &auth.SocialOAuthHandler{Providers: providers, Resolver: resolver, Service: service, Audit: audit, StateKey: key, Secure: secure}, nil
}

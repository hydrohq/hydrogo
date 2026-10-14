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
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	membertransport "github.com/ardaozyildirim/hydrogo/modules/members/transport"
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

// configuredMemberSocialOAuth wires the member-facing social sign-in surface.
// It reuses each provider's admin app credentials (HYDRO_OAUTH_<NAME>_CLIENT_ID/
// _CLIENT_SECRET) and the shared HYDRO_OAUTH_STATE_KEY, since both surfaces run
// in the same process and signing the same way is enough to keep their state
// cookies independent (OAuthFlow gives each its own cookie name and path) —
// but requires its own HYDRO_MEMBER_OAUTH_<NAME>_REDIRECT_URL so the provider
// routes admin and member callbacks to different URLs.
func configuredMemberSocialOAuth(db *sql.DB, dialect database.Dialect, service members.Service, audit auth.AuditRecorder, secure, requireTenantScope bool) (*membertransport.OAuthHandler, error) {
	providers := make(map[string]auth.OAuthProviderConfig)
	for _, name := range []string{"google", "github", "apple"} {
		upper := strings.ToUpper(name)
		clientID, secret := os.Getenv("HYDRO_OAUTH_"+upper+"_CLIENT_ID"), os.Getenv("HYDRO_OAUTH_"+upper+"_CLIENT_SECRET")
		redirect := os.Getenv("HYDRO_MEMBER_OAUTH_" + upper + "_REDIRECT_URL")
		if clientID == "" && secret == "" && redirect == "" {
			continue
		}
		if clientID == "" || secret == "" || redirect == "" {
			return nil, fmt.Errorf("member social sign-in for %s requires HYDRO_OAUTH_%s_CLIENT_ID, HYDRO_OAUTH_%s_CLIENT_SECRET and HYDRO_MEMBER_OAUTH_%s_REDIRECT_URL", name, upper, upper, upper)
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
		return nil, errors.New("HYDRO_OAUTH_STATE_KEY must contain at least 32 bytes when a member social provider is enabled")
	}
	resolver := memberstore.OAuthStore{DB: db, Dialect: dialect, RequireTenantScope: requireTenantScope}
	flow := auth.OAuthFlow{Providers: providers, StateKey: key, Secure: secure, CookieName: "hydro_member_oauth_state", CookiePath: "/member/oauth/", DefaultReturnTo: "/member/account", CancelRedirect: "/member/login?social=cancelled"}
	return &membertransport.OAuthHandler{OAuthFlow: flow, Resolver: resolver, Service: service, Audit: audit, Secure: secure}, nil
}

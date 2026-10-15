package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/erasure"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// AuditRecorder records security-relevant authentication events.
type AuditRecorder interface {
	Record(ctx context.Context, action, targetID string) error
}

type AuditRecorderFunc func(context.Context, string, string) error

func (f AuditRecorderFunc) Record(ctx context.Context, action, targetID string) error {
	return f(ctx, action, targetID)
}

type RecoveryNotifier interface {
	SendPasswordReset(context.Context, string, string) error
}

type RecoveryNotifierFunc func(context.Context, string, string) error

func (f RecoveryNotifierFunc) SendPasswordReset(ctx context.Context, email, token string) error {
	return f(ctx, email, token)
}

type InviteNotifier interface {
	SendInvite(context.Context, string, string) error
}

type InviteNotifierFunc func(context.Context, string, string) error

func (f InviteNotifierFunc) SendInvite(ctx context.Context, email, token string) error {
	return f(ctx, email, token)
}

type HTTPRoute struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

const maxAuthFormBytes = 16 << 10
const maxPermissionFormBytes = 256 << 10

// HTTPHandler exposes the small, session-backed authentication surface used by the server.
type HTTPHandler struct {
	Service                *AuthService
	Audit                  AuditRecorder
	Secure                 bool
	MFA                    *MFAService
	MFARequiredRoles       []Role
	MFADB                  database.Executor
	Transactions           database.Transactor
	LoginLimiter           security.RateLimiter
	MFALimiter             security.RateLimiter
	RecoveryLimiter        security.RateLimiter
	RecoveryConfirmLimiter security.RateLimiter
	InviteAcceptLimiter    security.RateLimiter
	LoginTarpit            *security.Tarpit
	Recovery               *PasswordRecovery
	RecoveryDB             database.Executor
	Notifier               RecoveryNotifier
	Invitation             *InvitationService
	InviteDB               database.Executor
	PublicOrigin           string
	InviteNotice           InviteNotifier
	SocialOAuth            *SocialOAuthHandler
	LoginSecurity          LoginSecurityTracker
	PasswordResetTTL       time.Duration
	InvitationTTL          time.Duration
	TokenLifetimeSettings  TokenLifetimeSettings
	ErasureLog             *erasure.Log
	EditionID              string
}

func (h HTTPHandler) Routes() []HTTPRoute {
	routes := []HTTPRoute{
		{Pattern: "POST /auth/login", Public: true, Handler: http.HandlerFunc(h.login)},
		{Pattern: "POST /auth/logout", Handler: http.HandlerFunc(h.logout)},
		{Pattern: "GET /auth/me", Handler: http.HandlerFunc(h.me)},
		{Pattern: "GET /auth/me/export", Handler: http.HandlerFunc(h.exportUser)},
		{Pattern: "POST /auth/me/erasure", Handler: http.HandlerFunc(h.requestErasure)},
		{Pattern: "GET /auth/me/sessions", Handler: http.HandlerFunc(h.listSessions)},
		{Pattern: "POST /auth/me/sessions/revoke-others", Handler: http.HandlerFunc(h.revokeOtherSessions)},
		{Pattern: "POST /auth/me/session/extend", Handler: http.HandlerFunc(h.extendSession)},
		{Pattern: "POST /auth/me/credentials", Handler: http.HandlerFunc(h.changeCredentials)},
		{Pattern: "POST /auth/users/{id}/disable", Handler: http.HandlerFunc(h.disableUser)},
		{Pattern: "POST /auth/users/{id}/role", Handler: http.HandlerFunc(h.updateUserRole)},
		{Pattern: "POST /auth/users/{id}/permissions", Handler: http.HandlerFunc(h.updateUserPermissions)},
		{Pattern: "POST /auth/me/mfa/enroll", Handler: http.HandlerFunc(h.enrollMFA)},
		{Pattern: "POST /auth/me/mfa/enable", Handler: http.HandlerFunc(h.enableMFA)},
		{Pattern: "POST /auth/me/mfa/disable", Handler: http.HandlerFunc(h.disableMFA)},
		{Pattern: "POST /auth/password-reset/request", Public: true, Handler: http.HandlerFunc(h.requestPasswordReset)},
		{Pattern: "POST /auth/password-reset/confirm", Public: true, Handler: http.HandlerFunc(h.confirmPasswordReset)},
		{Pattern: "POST /auth/users/{id}/password-reset-link", Handler: http.HandlerFunc(h.adminPasswordResetLink)},
		{Pattern: "POST /admin/auth/invites", Handler: http.HandlerFunc(h.issueInvite)},
		{Pattern: "GET /api/v1/admin/me", Handler: http.HandlerFunc(h.adminMe)},
		{Pattern: "GET /api/v1/admin/account/tokens", Handler: http.HandlerFunc(h.listPersonalTokens)},
		{Pattern: "POST /api/v1/admin/account/tokens", Handler: http.HandlerFunc(h.issuePersonalToken)},
		{Pattern: "DELETE /api/v1/admin/account/tokens/{id}", Handler: http.HandlerFunc(h.revokePersonalToken)},
		{Pattern: "GET /api/v1/admin/api-keys", Handler: http.HandlerFunc(h.listAPIKeys)},
		{Pattern: "POST /api/v1/admin/api-keys", Handler: http.HandlerFunc(h.issueAPIKey)},
		{Pattern: "DELETE /api/v1/admin/api-keys/{id}", Handler: http.HandlerFunc(h.revokeAPIKey)},
		{Pattern: "POST /auth/invites/accept", Public: true, Handler: http.HandlerFunc(h.acceptInvite)},
	}
	if h.SocialOAuth != nil {
		social := *h.SocialOAuth
		social.Secure = h.Secure
		routes = append(routes, social.Routes()...)
	}
	return routes
}

func (h HTTPHandler) readTokenLifetimeSettings(ctx context.Context) (TokenLifetimes, bool, error) {
	if h.TokenLifetimeSettings == nil {
		return TokenLifetimes{}, false, nil
	}
	return h.TokenLifetimeSettings.Read(ctx)
}

func (h HTTPHandler) effectiveTokenLifetimes(ctx context.Context) (TokenLifetimes, error) {
	value := TokenLifetimes{PasswordReset: h.PasswordResetTTL, Invitation: h.InvitationTTL}
	if value.PasswordReset <= 0 {
		value.PasswordReset = time.Hour
	}
	if value.Invitation <= 0 {
		value.Invitation = 24 * time.Hour
	}
	configured, ok, err := h.readTokenLifetimeSettings(ctx)
	if err != nil {
		return TokenLifetimes{}, err
	}
	if ok {
		value = configured
	}
	return value, value.Validate()
}

func (h HTTPHandler) RouterRoutes() []router.Route {
	routes := h.Routes()
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if route.Public {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func noStoreAuthResponse(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func (h HTTPHandler) recordAuthFailure(r *http.Request, email string) {
	digest := sha256.Sum256([]byte(NormalizeEmail(email)))
	h.recordAuthEvent(r, "auth.login_failed", hex.EncodeToString(digest[:]))
}

func (h HTTPHandler) recordAuthEvent(r *http.Request, action, target string) {
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), action, target)
	}
}

func sessionToken(r *http.Request) string {
	cookie, _ := r.Cookie(SessionCookieName)
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

func clientIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (h HTTPHandler) adminUser(r *http.Request) bool {
	user := UserFromContext(r.Context())
	return user != nil && user.Active && user.Role == RoleAdmin
}

func (h HTTPHandler) adminUserManage(r *http.Request) bool {
	user := UserFromContext(r.Context())
	return h.adminUser(r) && h.Service.Can(user, PermissionUsersManage)
}

func writeAuthDependencyError(w http.ResponseWriter, message string, err error, fallbackStatus int) {
	status := fallbackStatus
	if errors.Is(err, database.ErrDatabaseRequired) ||
		errors.Is(err, ErrUserRepositoryRequired) ||
		errors.Is(err, ErrSessionRepositoryRequired) ||
		errors.Is(err, ErrMFAStoreRequired) ||
		errors.Is(err, ErrRecoveryDependencyRequired) ||
		errors.Is(err, ErrInviteDependencyRequired) {
		status = http.StatusServiceUnavailable
	}
	http.Error(w, message, status)
}

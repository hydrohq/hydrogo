package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/erasure"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
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

func (h HTTPHandler) adminPasswordResetLink(w http.ResponseWriter, r *http.Request) {
	admin := UserFromContext(r.Context())
	if !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Recovery == nil || guards.IsNil(h.Recovery.Users) || guards.IsNil(h.Recovery.Tokens) || guards.IsNil(h.Transactions) {
		http.Error(w, "password reset is unavailable", http.StatusServiceUnavailable)
		return
	}
	origin, err := url.Parse(strings.TrimRight(strings.TrimSpace(h.PublicOrigin), "/"))
	if err != nil || !origin.IsAbs() || origin.Host == "" || origin.User != nil || origin.EscapedPath() != "" || origin.Scheme != "https" && origin.Scheme != "http" {
		http.Error(w, "password reset is unavailable", http.StatusServiceUnavailable)
		return
	}
	targetID := strings.TrimSpace(r.PathValue("id"))
	if targetID == "" || len(targetID) > 128 {
		http.Error(w, "user unavailable", http.StatusNotFound)
		return
	}
	if h.RecoveryConfirmLimiter != nil && !security.AllowForRequest(h.RecoveryConfirmLimiter, r.Context(), "admin-reset-link:"+admin.ID+":"+targetID) {
		security.WriteRateLimitedResponse(w, "password reset link is temporarily unavailable", time.Hour)
		return
	}
	user, err := h.Recovery.Users.GetByID(r.Context(), targetID)
	if err != nil || user == nil || !user.Active {
		http.Error(w, "user unavailable", http.StatusNotFound)
		return
	}
	policy, err := h.effectiveTokenLifetimes(r.Context())
	if err != nil {
		http.Error(w, "password reset link could not be created", http.StatusServiceUnavailable)
		return
	}
	var token string
	err = h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var issueErr error
		token, issueErr = h.Recovery.IssueForUser(r.Context(), tx, user, policy.PasswordReset)
		return issueErr
	})
	if err != nil || token == "" {
		http.Error(w, "password reset link could not be created", http.StatusServiceUnavailable)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.password_reset_link.generated", user.ID)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"url": strings.TrimRight(h.PublicOrigin, "/") + "/password-reset?token=" + url.QueryEscape(token)})
}

func (h HTTPHandler) issueInvite(w http.ResponseWriter, r *http.Request) {
	if !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Invitation == nil || database.RequireExecutor(h.InviteDB) != nil || guards.IsNil(h.InviteNotice) || strings.TrimSpace(h.PublicOrigin) == "" {
		http.Error(w, "invitation unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("email")) == "" {
		http.Error(w, "invalid invitation request", http.StatusBadRequest)
		return
	}
	role := Role(r.FormValue("role"))
	ttl := h.InvitationTTL
	if policy, ok, policyErr := h.readTokenLifetimeSettings(r.Context()); policyErr != nil {
		http.Error(w, "invitation is temporarily unavailable", http.StatusServiceUnavailable)
		return
	} else if ok {
		ttl = policy.Invitation
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	token, err := h.Invitation.Issue(r.Context(), h.InviteDB, r.FormValue("email"), role, ttl)
	if errors.Is(err, ErrUserAlreadyExists) {
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}
	if err != nil {
		writeAuthDependencyError(w, "invitation failed", err, http.StatusBadRequest)
		return
	}
	if err := h.InviteNotice.SendInvite(r.Context(), NormalizeEmail(r.FormValue("email")), token); err != nil {
		http.Error(w, "invitation delivery failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"email": NormalizeEmail(r.FormValue("email")),
		"url":   strings.TrimRight(h.PublicOrigin, "/") + "/auth/invites/accept?token=" + url.QueryEscape(token),
	})
}

func (h HTTPHandler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	html := strings.Contains(r.Header.Get("Accept"), "text/html")
	if h.Invitation == nil || h.Transactions == (database.Transactor{}) {
		if html {
			redirectInviteForm(w, r, "unavailable")
			return
		}
		http.Error(w, "invitation unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("token")) == "" || r.FormValue("password") == "" {
		if html {
			redirectInviteForm(w, r, "invalid")
			return
		}
		http.Error(w, "invalid invitation request", http.StatusBadRequest)
		return
	}
	if html && r.FormValue("password") != r.FormValue("password_confirm") {
		redirectInviteForm(w, r, "mismatch")
		return
	}
	if h.InviteAcceptLimiter != nil && !security.AllowForRequest(h.InviteAcceptLimiter, r.Context(), clientIP(r)) {
		if html {
			redirectInviteForm(w, r, "rate")
			return
		}
		security.WriteRateLimitedResponse(w, "too many invitation attempts", time.Minute)
		return
	}
	var user *User
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var err error
		user, err = h.Invitation.Accept(r.Context(), tx, r.FormValue("token"), r.FormValue("password"))
		return err
	})
	if errors.Is(err, ErrInviteInvalid) {
		if html {
			redirectInviteForm(w, r, "invalid")
			return
		}
		http.Error(w, "This invitation is invalid or has expired. Ask your administrator to send a new invitation.", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrWeakPassword) {
		if html {
			redirectInviteForm(w, r, "weak")
			return
		}
		http.Error(w, "Choose a longer, less common password.", http.StatusBadRequest)
		return
	}
	if errors.Is(err, ErrUserAlreadyExists) {
		if html {
			redirectInviteForm(w, r, "invalid")
			return
		}
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}
	if err != nil {
		if html {
			redirectInviteForm(w, r, "unavailable")
			return
		}
		writeAuthDependencyError(w, "invitation acceptance failed", err, http.StatusInternalServerError)
		return
	}
	if html {
		values := url.Values{}
		values.Set("invite_accepted", "1")
		if r.FormValue("lang") == "tr" {
			values.Set("lang", "tr")
		}
		http.Redirect(w, r, "/login?"+values.Encode(), http.StatusSeeOther)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"user_id": user.ID, "email": user.Email, "role": user.Role})
}

func redirectInviteForm(w http.ResponseWriter, r *http.Request, code string) {
	values := url.Values{}
	values.Set("token", r.FormValue("token"))
	values.Set("error", code)
	if r.FormValue("lang") == "tr" {
		values.Set("lang", "tr")
	}
	http.Redirect(w, r, "/auth/invites/accept?"+values.Encode(), http.StatusSeeOther)
}

func (h HTTPHandler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if h.Recovery == nil || database.RequireExecutor(h.RecoveryDB) != nil || guards.IsNil(h.Notifier) {
		http.Error(w, "password recovery unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("email")) == "" {
		http.Error(w, "invalid password recovery request", http.StatusBadRequest)
		return
	}
	if h.RecoveryLimiter != nil && !security.AllowForRequest(h.RecoveryLimiter, r.Context(), clientIP(r)) {
		security.WriteRateLimitedResponse(w, "too many password recovery requests", time.Minute)
		return
	}
	ttl := h.PasswordResetTTL
	if policy, ok, policyErr := h.readTokenLifetimeSettings(r.Context()); policyErr != nil {
		// Preserve the generic accepted response and avoid revealing account state.
		w.WriteHeader(http.StatusAccepted)
		return
	} else if ok {
		ttl = policy.PasswordReset
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	token, err := h.Recovery.Issue(r.Context(), h.RecoveryDB, r.FormValue("email"), ttl)
	if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrUserInactive) {
		// Do not disclose whether an account exists.
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err != nil {
		writeAuthDependencyError(w, "password recovery unavailable", err, http.StatusInternalServerError)
		return
	}
	user, err := h.Recovery.Users.GetByEmail(r.Context(), NormalizeEmail(r.FormValue("email")))
	if err != nil {
		writeAuthDependencyError(w, "password recovery unavailable", err, http.StatusInternalServerError)
		return
	}
	if err := h.Notifier.SendPasswordReset(r.Context(), user.Email, token); err != nil {
		http.Error(w, "password recovery unavailable", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h HTTPHandler) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	html := strings.Contains(r.Header.Get("Accept"), "text/html")
	if h.Recovery == nil || h.Transactions == (database.Transactor{}) {
		if html {
			redirectPasswordResetForm(w, r, "unavailable")
			return
		}
		http.Error(w, "password recovery unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("token")) == "" || r.FormValue("password") == "" {
		if html {
			redirectPasswordResetForm(w, r, "invalid")
			return
		}
		http.Error(w, "invalid password recovery request", http.StatusBadRequest)
		return
	}
	if html && r.FormValue("password") != r.FormValue("password_confirm") {
		redirectPasswordResetForm(w, r, "mismatch")
		return
	}
	if h.RecoveryConfirmLimiter != nil && !security.AllowForRequest(h.RecoveryConfirmLimiter, r.Context(), clientIP(r)) {
		if html {
			redirectPasswordResetForm(w, r, "rate")
			return
		}
		security.WriteRateLimitedResponse(w, "too many password recovery attempts", time.Minute)
		return
	}
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.Recovery.Reset(r.Context(), tx, r.FormValue("token"), r.FormValue("password"))
	})
	if errors.Is(err, ErrRecoveryTokenInvalid) {
		if html {
			redirectPasswordResetForm(w, r, "invalid")
			return
		}
		http.Error(w, "This password reset link is invalid or has expired. Request a new link to continue.", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrWeakPassword) {
		if html {
			redirectPasswordResetForm(w, r, "weak")
			return
		}
		http.Error(w, "Choose a longer, less common password.", http.StatusBadRequest)
		return
	}
	if err != nil {
		if html {
			redirectPasswordResetForm(w, r, "unavailable")
			return
		}
		writeAuthDependencyError(w, "password recovery failed", err, http.StatusInternalServerError)
		return
	}
	if html {
		values := url.Values{"password_reset": {"1"}}
		if r.FormValue("lang") == "tr" {
			values.Set("lang", "tr")
		}
		http.Redirect(w, r, "/login?"+values.Encode(), http.StatusSeeOther)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func redirectPasswordResetForm(w http.ResponseWriter, r *http.Request, code string) {
	values := url.Values{}
	values.Set("token", r.FormValue("token"))
	values.Set("error", code)
	if r.FormValue("lang") == "tr" {
		values.Set("lang", "tr")
	}
	http.Redirect(w, r, "/password-reset?"+values.Encode(), http.StatusSeeOther)
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

func (h HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if h.Service == nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	if h.LoginLimiter != nil && !security.AllowForRequest(h.LoginLimiter, r.Context(), clientIP(r)) {
		security.WriteRateLimitedResponse(w, "too many authentication attempts", time.Minute)
		return
	}
	user, err := h.Service.Authenticate(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
			if h.LoginSecurity != nil {
				_ = h.LoginSecurity.RecordFailedLogin(r.Context(), r.FormValue("email"))
			}
			h.recordAuthFailure(r, r.FormValue("email"))
			if h.LoginTarpit != nil {
				if err := h.LoginTarpit.Wait(r.Context(), loginAttemptKey(r)); err != nil {
					return
				}
			}
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		writeAuthDependencyError(w, "authentication failed", err, http.StatusInternalServerError)
		return
	}
	if h.MFA != nil {
		if err := database.RequireExecutor(h.MFADB); err != nil {
			http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		enabled, err := h.MFA.Enabled(r.Context(), h.MFADB, user.ID)
		if err != nil {
			writeAuthDependencyError(w, "authentication failed", err, http.StatusInternalServerError)
			return
		}
		if enabled {
			code := strings.TrimSpace(r.FormValue("mfa_code"))
			if code == "" {
				api.WriteJSON(w, http.StatusUnauthorized, map[string]any{
					"error":               "mfa_code_required",
					"totp_period_seconds": 30,
					"server_time_unix":    time.Now().UTC().Unix(),
				})
				return
			}
			if h.MFALimiter != nil && !security.AllowForRequest(h.MFALimiter, r.Context(), clientIP(r)+":"+user.ID) {
				security.WriteRateLimitedResponse(w, "too many MFA attempts", time.Minute)
				return
			}
			if err := h.MFA.Verify(r.Context(), h.MFADB, user.ID, code); err != nil {
				if !errors.Is(err, ErrMFAInvalidCode) {
					writeAuthDependencyError(w, "authentication failed", err, http.StatusInternalServerError)
					return
				}
				h.recordAuthEvent(r, "auth.mfa_failed", user.ID)
				if recoveryErr := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
					return h.MFA.ConsumeRecoveryCode(r.Context(), tx, user.ID, code)
				}); recoveryErr != nil {
					if errors.Is(recoveryErr, database.ErrDatabaseRequired) || errors.Is(recoveryErr, ErrMFAStoreRequired) {
						writeAuthDependencyError(w, "authentication failed", recoveryErr, http.StatusInternalServerError)
						return
					}
					http.Error(w, "invalid MFA code", http.StatusUnauthorized)
					return
				}
			}
		}
		if h.requiresMFA(user.Role) && !enabled {
			http.Error(w, "MFA setup required for this account", http.StatusForbidden)
			return
		}
	} else if h.requiresMFA(user.Role) {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	if h.LoginTarpit != nil {
		h.LoginTarpit.Reset(loginAttemptKey(r))
	}
	token, session, err := h.Service.CreateSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), DefaultSessionTTL)
	if err != nil {
		writeAuthDependencyError(w, "authentication failed", err, http.StatusInternalServerError)
		return
	}
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "auth.login", user.ID); err != nil {
			_ = h.Service.RevokeSession(r.Context(), token)
			http.Error(w, "authentication failed", http.StatusInternalServerError)
			return
		}
	}
	notice := LoginSecurityNotice{}
	if h.LoginSecurity != nil {
		var trackingErr error
		notice, trackingErr = h.LoginSecurity.RecordSuccessfulLogin(r.Context(), user.ID, clientIP(r), time.Now().UTC())
		if trackingErr != nil {
			_ = h.Service.RevokeSession(r.Context(), token)
			writeAuthDependencyError(w, "authentication failed", trackingErr, http.StatusInternalServerError)
			return
		}
	}
	SetSessionCookie(w, token, session.ExpiresAt, h.Secure)
	api.WriteJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user_id": user.ID, "role": user.Role, "security_notice": notice})
}

func (h HTTPHandler) requiresMFA(role Role) bool {
	for _, required := range h.MFARequiredRoles {
		if required == role {
			return true
		}
	}
	return false
}

func loginAttemptKey(r *http.Request) string {
	return clientIP(r) + ":" + NormalizeEmail(r.FormValue("email"))
}

func (h HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if h.Service == nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	session := SessionFromContext(r.Context())
	if session != nil {
		if err := h.Service.RevokeSession(r.Context(), sessionToken(r)); err != nil && !errors.Is(err, ErrSessionNotFound) {
			writeAuthDependencyError(w, "logout failed", err, http.StatusInternalServerError)
			return
		}
		if h.Audit != nil {
			// Logout remains successful even if the audit sink is temporarily
			// unavailable: the session was already revoked and the browser must
			// not be left believing it is still authenticated.
			_ = h.Audit.Record(r.Context(), "auth.logout", session.UserID)
		}
	}
	ClearSessionCookie(w, h.Secure)
	w.WriteHeader(http.StatusNoContent)
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

func (h HTTPHandler) me(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	session := SessionFromContext(r.Context())
	expiresAt := time.Time{}
	if session != nil {
		expiresAt = session.ExpiresAt
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"id": user.ID, "email": user.Email, "role": user.Role, "active": user.Active, "session_expires_at": expiresAt})
}

func (h HTTPHandler) listSessions(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	session := SessionFromContext(r.Context())
	if user == nil || session == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	items, err := h.Service.ListUserSessions(r.Context(), user.ID)
	if err != nil {
		writeAuthDependencyError(w, "sessions unavailable", err, http.StatusServiceUnavailable)
		return
	}
	type item struct {
		ID, UserAgent, IPAddress         string
		CreatedAt, LastSeenAt, ExpiresAt time.Time
		Current                          bool
	}
	out := make([]item, 0, len(items))
	for _, x := range items {
		out = append(out, item{ID: x.ID, UserAgent: x.UserAgent, IPAddress: x.IPAddress, CreatedAt: x.CreatedAt, LastSeenAt: x.LastSeenAt, ExpiresAt: x.ExpiresAt, Current: x.ID == session.ID})
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

func (h HTTPHandler) revokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	session := SessionFromContext(r.Context())
	if user == nil || session == nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := h.Service.RevokeOtherUserSessions(r.Context(), user.ID, session.TokenHash); err != nil {
		writeAuthDependencyError(w, "sessions could not be revoked", err, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.FormValue("_csrf") != "" {
		http.Redirect(w, r, "/auth/me/sessions/manage", http.StatusSeeOther)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h HTTPHandler) extendSession(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	user, current := UserFromContext(r.Context()), SessionFromContext(r.Context())
	oldToken := sessionToken(r)
	if user == nil || !user.Active || current == nil || oldToken == "" || h.Service == nil {
		http.Error(w, "session extension unavailable", http.StatusUnauthorized)
		return
	}
	token, next, err := h.Service.CreateSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), DefaultSessionTTL)
	if err != nil {
		writeAuthDependencyError(w, "session extension failed", err, http.StatusInternalServerError)
		return
	}
	if err := h.Service.RevokeSession(r.Context(), oldToken); err != nil {
		_ = h.Service.RevokeSession(r.Context(), token)
		writeAuthDependencyError(w, "session extension failed", err, http.StatusInternalServerError)
		return
	}
	SetSessionCookie(w, token, next.ExpiresAt, h.Secure)
	api.WriteJSON(w, http.StatusOK, map[string]any{"session_expires_at": next.ExpiresAt})
}

func (h HTTPHandler) changeCredentials(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.Service == nil || h.Transactions == (database.Transactor{}) {
		http.Error(w, "authentication unavailable", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid account update", http.StatusBadRequest)
		return
	}
	newEmail := strings.TrimSpace(r.FormValue("email"))
	newPassword := r.FormValue("new_password")
	if newEmail == user.Email {
		newEmail = ""
	}
	if newEmail == "" && newPassword == "" {
		http.Error(w, "no account changes requested", http.StatusBadRequest)
		return
	}
	if newPassword != "" && newPassword != r.FormValue("confirm_password") {
		http.Error(w, "new passwords do not match", http.StatusBadRequest)
		return
	}
	if newPassword != "" && len([]rune(newPassword)) < 8 {
		http.Error(w, "new password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if newPassword != "" {
		if err := ValidatePasswordStrength(newPassword); err != nil {
			http.Error(w, "Choose a longer, less common password.", http.StatusBadRequest)
			return
		}
	}
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.Service.ChangeCredentials(r.Context(), user.ID, r.FormValue("current_password"), newEmail, newPassword, tx)
	})
	if errors.Is(err, ErrInvalidCredentials) {
		http.Error(w, "current password is incorrect", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrUserAlreadyExists) {
		http.Error(w, "email is already in use", http.StatusConflict)
		return
	}
	if err != nil {
		writeAuthDependencyError(w, "account update failed", err, http.StatusBadRequest)
		return
	}
	if r.FormValue("invalidate_other_sessions") == "true" {
		if err := h.Service.RevokeOtherUserSessions(r.Context(), user.ID, HashSessionToken(sessionToken(r))); err != nil {
			writeAuthDependencyError(w, "account update failed", err, http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.Service.RevokeAllUserSessions(r.Context(), user.ID); err != nil {
			writeAuthDependencyError(w, "account update failed", err, http.StatusInternalServerError)
			return
		}
	}
	if h.Audit != nil {
		h.recordAuthEvent(r, "auth.credentials_changed", user.ID)
	}
	if r.FormValue("invalidate_other_sessions") != "true" {
		ClearSessionCookie(w, h.Secure)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) disableUser(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	actor := UserFromContext(r.Context())
	if !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	targetID := r.PathValue("id")
	if targetID == "" || targetID == actor.ID {
		http.Error(w, "administrator cannot disable the current account", http.StatusConflict)
		return
	}
	if err := h.Service.SetUserActive(r.Context(), targetID, false); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		writeAuthDependencyError(w, "user state update failed", err, http.StatusInternalServerError)
		return
	}
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "auth.user_disabled", targetID); err != nil {
			_ = h.Service.SetUserActive(r.Context(), targetID, true)
			http.Error(w, "audit failed", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) updateUserRole(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := UserFromContext(r.Context())
	targetID := r.PathValue("id")
	if targetID == "" || targetID == actor.ID {
		http.Error(w, "administrator role cannot be changed here", http.StatusConflict)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid role update", http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateUserRole(r.Context(), targetID, Role(strings.TrimSpace(r.FormValue("role")))); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		writeAuthDependencyError(w, "user role update failed", err, http.StatusBadRequest)
		return
	}
	h.recordAuthEvent(r, "auth.user_role_changed", targetID)
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) updateUserPermissions(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := UserFromContext(r.Context())
	targetID := r.PathValue("id")
	if targetID == "" || targetID == actor.ID {
		http.Error(w, "administrator permissions cannot be changed for the current account", http.StatusConflict)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPermissionFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid permission update", http.StatusBadRequest)
		return
	}
	overrides := make(map[string]bool)
	for _, permission := range h.Service.AvailablePermissions() {
		switch r.FormValue("perm_" + permission) {
		case "allow":
			overrides[permission] = true
		case "deny":
			overrides[permission] = false
		}
	}
	if err := h.Service.ReplaceUserPermissions(r.Context(), targetID, overrides); err != nil {
		writeAuthDependencyError(w, "user permissions update failed", err, http.StatusBadRequest)
		return
	}
	h.recordAuthEvent(r, "auth.user_permissions_changed", targetID)
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) enrollMFA(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.MFA == nil || database.RequireExecutor(h.MFADB) != nil {
		http.Error(w, "MFA unavailable", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid MFA request", http.StatusBadRequest)
		return
	}
	if h.MFALimiter != nil && !security.AllowForRequest(h.MFALimiter, r.Context(), clientIP(r)+":"+user.ID) {
		security.WriteRateLimitedResponse(w, "too many MFA attempts", time.Minute)
		return
	}
	enabled, err := h.MFA.Enabled(r.Context(), h.MFADB, user.ID)
	if err != nil {
		writeAuthDependencyError(w, "MFA enrollment unavailable", err, http.StatusServiceUnavailable)
		return
	}
	if enabled {
		http.Error(w, "MFA is already enabled", http.StatusConflict)
		return
	}
	var enrollment MFAEnrollment
	err = h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var err error
		enrollment, err = h.MFA.Enroll(r.Context(), tx, user.ID, user.Email)
		return err
	})
	if err != nil {
		if errors.Is(err, ErrMFAAlreadyEnabled) {
			http.Error(w, "MFA is already enabled", http.StatusConflict)
			return
		}
		writeAuthDependencyError(w, "MFA enrollment failed", err, http.StatusInternalServerError)
		return
	}
	h.recordAuthEvent(r, "auth.mfa_enrollment_started", user.ID)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	api.WriteJSON(w, http.StatusCreated, enrollment)
}

func (h HTTPHandler) enableMFA(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.MFA == nil {
		http.Error(w, "MFA unavailable", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid MFA request", http.StatusBadRequest)
		return
	}
	if h.MFALimiter != nil && !security.AllowForRequest(h.MFALimiter, r.Context(), clientIP(r)+":"+user.ID) {
		security.WriteRateLimitedResponse(w, "too many MFA attempts", time.Minute)
		return
	}
	if err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.MFA.EnableWithCode(r.Context(), tx, user.ID, r.FormValue("code"))
	}); err != nil {
		if errors.Is(err, ErrMFAInvalidCode) {
			http.Error(w, "invalid MFA code", http.StatusUnauthorized)
			return
		}
		writeAuthDependencyError(w, "MFA enable failed", err, http.StatusInternalServerError)
		return
	}
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "auth.mfa_enabled", user.ID); err != nil {
			// Keep the factor disabled when its security event cannot be recorded.
			_ = h.Transactions.Within(r.Context(), func(tx database.Executor) error {
				return h.MFA.Disable(r.Context(), tx, user.ID)
			})
			http.Error(w, "audit failed", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h HTTPHandler) exportUser(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var mfaEnabled bool
	if h.MFA != nil && h.MFADB != nil {
		mfaEnabled, _ = h.MFA.Enabled(r.Context(), h.MFADB, user.ID)
	}

	sessions := []map[string]any{}
	if h.Service != nil {
		items, err := h.Service.ListUserSessions(r.Context(), user.ID)
		if err == nil {
			for _, x := range items {
				sessions = append(sessions, map[string]any{
					"id":           x.ID,
					"user_agent":   x.UserAgent,
					"ip_address":   x.IPAddress,
					"created_at":   x.CreatedAt,
					"last_seen_at": x.LastSeenAt,
					"expires_at":   x.ExpiresAt,
				})
			}
		}
	}

	exportData := map[string]any{
		"user": map[string]any{
			"id":         user.ID,
			"email":      user.Email,
			"role":       user.Role,
			"active":     user.Active,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
		"security": map[string]any{
			"mfa_enabled": mfaEnabled,
		},
		"sessions": sessions,
	}

	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusOK, exportData)
}

func (h HTTPHandler) requestErasure(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || h.Service == nil || h.ErasureLog == nil {
		http.Error(w, "erasure unavailable", http.StatusServiceUnavailable)
		return
	}

	intentIDBytes := make([]byte, 16)
	rand.Read(intentIDBytes)
	intentID := hex.EncodeToString(intentIDBytes)

	editionID := h.EditionID
	if editionID == "" {
		editionID = "default"
	}

	intent := erasure.Intent{
		ID:        intentID,
		EditionID: editionID,
		TargetID:  user.ID,
		Kind:      "user_erasure",
		CreatedAt: time.Now().UTC(),
	}

	if _, err := h.ErasureLog.Append(intent); err != nil {
		writeAuthDependencyError(w, "erasure failed", err, http.StatusInternalServerError)
		return
	}

	if err := h.Service.SetUserActive(r.Context(), user.ID, false); err != nil {
		writeAuthDependencyError(w, "erasure failed", err, http.StatusInternalServerError)
		return
	}

	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.user_erasure_requested", user.ID)
	}

	ClearSessionCookie(w, h.Secure)
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "erasure_requested"})
}

func (h HTTPHandler) disableMFA(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.MFA == nil || h.Transactions == (database.Transactor{}) {
		http.Error(w, "MFA unavailable", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid MFA request", http.StatusBadRequest)
		return
	}
	if h.MFALimiter != nil && !security.AllowForRequest(h.MFALimiter, r.Context(), clientIP(r)+":"+user.ID) {
		security.WriteRateLimitedResponse(w, "too many MFA attempts", time.Minute)
		return
	}

	if err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		code := r.FormValue("code")
		if err := h.MFA.Verify(r.Context(), tx, user.ID, code); err != nil {
			if !errors.Is(err, ErrMFAInvalidCode) {
				return err
			}
			if err := h.MFA.ConsumeRecoveryCode(r.Context(), tx, user.ID, code); err != nil {
				return ErrMFAInvalidCode
			}
		}
		return h.MFA.Disable(r.Context(), tx, user.ID)
	}); err != nil {
		if errors.Is(err, ErrMFAInvalidCode) {
			http.Error(w, "invalid MFA code", http.StatusUnauthorized)
			return
		}
		writeAuthDependencyError(w, "MFA disable failed", err, http.StatusInternalServerError)
		return
	}

	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.mfa_disabled", user.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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

// HTTPHandler exposes the small, session-backed authentication surface used by the server.
type HTTPHandler struct {
	Service      *AuthService
	Audit        AuditRecorder
	Secure       bool
	MFA          *MFAService
	MFADB        database.Executor
	Transactions database.Transactor
	LoginLimiter *security.AttemptLimiter
	MFALimiter   *security.AttemptLimiter
	Recovery     *PasswordRecovery
	RecoveryDB   database.Executor
	Notifier     RecoveryNotifier
	Invitation   *InvitationService
	InviteDB     database.Executor
	InviteNotice InviteNotifier
}

func (h HTTPHandler) Routes() []HTTPRoute {
	return []HTTPRoute{
		{Pattern: "POST /auth/login", Public: true, Handler: http.HandlerFunc(h.login)},
		{Pattern: "POST /auth/logout", Handler: http.HandlerFunc(h.logout)},
		{Pattern: "GET /auth/me", Handler: http.HandlerFunc(h.me)},
		{Pattern: "POST /auth/users/{id}/disable", Handler: http.HandlerFunc(h.disableUser)},
		{Pattern: "POST /auth/me/mfa/enroll", Handler: http.HandlerFunc(h.enrollMFA)},
		{Pattern: "POST /auth/me/mfa/enable", Handler: http.HandlerFunc(h.enableMFA)},
		{Pattern: "POST /auth/password-reset/request", Public: true, Handler: http.HandlerFunc(h.requestPasswordReset)},
		{Pattern: "POST /auth/password-reset/confirm", Public: true, Handler: http.HandlerFunc(h.confirmPasswordReset)},
		{Pattern: "POST /admin/auth/invites", Handler: http.HandlerFunc(h.issueInvite)},
		{Pattern: "POST /auth/invites/accept", Public: true, Handler: http.HandlerFunc(h.acceptInvite)},
	}
}

func (h HTTPHandler) issueInvite(w http.ResponseWriter, r *http.Request) {
	if !h.adminUser(r) || h.Invitation == nil || h.InviteDB == nil || h.InviteNotice == nil {
		http.Error(w, "invitation unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("email")) == "" {
		http.Error(w, "invalid invitation request", http.StatusBadRequest)
		return
	}
	role := Role(r.FormValue("role"))
	token, err := h.Invitation.Issue(r.Context(), h.InviteDB, r.FormValue("email"), role, 24*time.Hour)
	if errors.Is(err, ErrUserAlreadyExists) {
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "invitation failed", http.StatusBadRequest)
		return
	}
	if err := h.InviteNotice.SendInvite(r.Context(), NormalizeEmail(r.FormValue("email")), token); err != nil {
		http.Error(w, "invitation delivery failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h HTTPHandler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	if h.Invitation == nil || h.Transactions == (database.Transactor{}) {
		http.Error(w, "invitation unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("token")) == "" || r.FormValue("password") == "" {
		http.Error(w, "invalid invitation request", http.StatusBadRequest)
		return
	}
	var user *User
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var err error
		user, err = h.Invitation.Accept(r.Context(), tx, r.FormValue("token"), r.FormValue("password"))
		return err
	})
	if errors.Is(err, ErrInviteInvalid) {
		http.Error(w, "invalid or expired invitation", http.StatusUnauthorized)
		return
	}
	if errors.Is(err, ErrUserAlreadyExists) {
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "invitation acceptance failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": user.ID, "email": user.Email, "role": user.Role})
}

func (h HTTPHandler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if h.Recovery == nil || h.RecoveryDB == nil || h.Notifier == nil {
		http.Error(w, "password recovery unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("email")) == "" {
		http.Error(w, "invalid password recovery request", http.StatusBadRequest)
		return
	}
	token, err := h.Recovery.Issue(r.Context(), h.RecoveryDB, r.FormValue("email"), 30*time.Minute)
	if errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrUserInactive) {
		// Do not disclose whether an account exists.
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err != nil {
		http.Error(w, "password recovery unavailable", http.StatusInternalServerError)
		return
	}
	user, err := h.Recovery.Users.GetByEmail(r.Context(), NormalizeEmail(r.FormValue("email")))
	if err != nil {
		http.Error(w, "password recovery unavailable", http.StatusInternalServerError)
		return
	}
	if err := h.Notifier.SendPasswordReset(r.Context(), user.Email, token); err != nil {
		http.Error(w, "password recovery unavailable", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h HTTPHandler) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	if h.Recovery == nil || h.Transactions == (database.Transactor{}) {
		http.Error(w, "password recovery unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil || strings.TrimSpace(r.FormValue("token")) == "" || r.FormValue("password") == "" {
		http.Error(w, "invalid password recovery request", http.StatusBadRequest)
		return
	}
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		return h.Recovery.Reset(r.Context(), tx, r.FormValue("token"), r.FormValue("password"))
	})
	if errors.Is(err, ErrRecoveryTokenInvalid) {
		http.Error(w, "invalid or expired recovery token", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "password recovery failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if h.Service == nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	if h.LoginLimiter != nil && !h.LoginLimiter.Allow(clientIP(r)+":"+NormalizeEmail(r.FormValue("email"))) {
		security.WriteRateLimitedResponse(w, "too many authentication attempts", time.Minute)
		return
	}
	user, err := h.Service.Authenticate(r.Context(), r.FormValue("email"), r.FormValue("password"))
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	if h.MFA != nil && h.MFADB != nil {
		enabled, err := h.MFA.Enabled(r.Context(), h.MFADB, user.ID)
		if err != nil {
			http.Error(w, "authentication failed", http.StatusInternalServerError)
			return
		}
		if enabled {
			if h.MFALimiter != nil && !h.MFALimiter.Allow(clientIP(r)+":"+user.ID) {
				security.WriteRateLimitedResponse(w, "too many MFA attempts", time.Minute)
				return
			}
			code := strings.TrimSpace(r.FormValue("mfa_code"))
			if code == "" {
				http.Error(w, "MFA code required", http.StatusUnauthorized)
				return
			}
			if err := h.MFA.Verify(r.Context(), h.MFADB, user.ID, code); err != nil {
				if recoveryErr := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
					return h.MFA.ConsumeRecoveryCode(r.Context(), tx, user.ID, code)
				}); recoveryErr != nil {
					http.Error(w, "invalid MFA code", http.StatusUnauthorized)
					return
				}
			}
		}
	}
	token, session, err := h.Service.CreateSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), DefaultSessionTTL)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	SetSessionCookie(w, token, session.ExpiresAt, h.Secure)
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "auth.login", user.ID); err != nil {
			_ = h.Service.RevokeSession(r.Context(), token)
			ClearSessionCookie(w, h.Secure)
			http.Error(w, "authentication failed", http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "user_id": user.ID, "role": user.Role})
}

func (h HTTPHandler) logout(w http.ResponseWriter, r *http.Request) {
	session := SessionFromContext(r.Context())
	if session != nil {
		if err := h.Service.RevokeSession(r.Context(), sessionToken(r)); err != nil && !errors.Is(err, ErrSessionNotFound) {
			http.Error(w, "logout failed", http.StatusInternalServerError)
			return
		}
		if h.Audit != nil {
			if err := h.Audit.Record(r.Context(), "auth.logout", session.UserID); err != nil {
				ClearSessionCookie(w, h.Secure)
				http.Error(w, "logout failed", http.StatusInternalServerError)
				return
			}
		}
	}
	ClearSessionCookie(w, h.Secure)
	w.WriteHeader(http.StatusNoContent)
}

func (h HTTPHandler) me(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "email": user.Email, "role": user.Role, "active": user.Active})
}

func (h HTTPHandler) disableUser(w http.ResponseWriter, r *http.Request) {
	actor := UserFromContext(r.Context())
	if actor == nil || !actor.Active || actor.Role != RoleAdmin {
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
		http.Error(w, "user state update failed", http.StatusInternalServerError)
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

func (h HTTPHandler) enrollMFA(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.MFA == nil {
		http.Error(w, "MFA unavailable", http.StatusForbidden)
		return
	}
	var enrollment MFAEnrollment
	err := h.Transactions.Within(r.Context(), func(tx database.Executor) error {
		var err error
		enrollment, err = h.MFA.Enroll(r.Context(), tx, user.ID, user.Email)
		return err
	})
	if err != nil {
		http.Error(w, "MFA enrollment failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, enrollment)
}

func (h HTTPHandler) enableMFA(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil || !user.Active || h.MFA == nil {
		http.Error(w, "MFA unavailable", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid MFA request", http.StatusBadRequest)
		return
	}
	if h.MFALimiter != nil && !h.MFALimiter.Allow(clientIP(r)+":"+user.ID) {
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
		http.Error(w, "MFA enable failed", http.StatusInternalServerError)
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

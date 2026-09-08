package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

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

type HTTPRoute struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

// HTTPHandler exposes the small, session-backed authentication surface used by the server.
type HTTPHandler struct {
	Service      *AuthService
	Audit        AuditRecorder
	Secure       bool
	MFA          *MFAService
	Transactions database.Transactor
	LoginLimiter *security.AttemptLimiter
}

func (h HTTPHandler) Routes() []HTTPRoute {
	return []HTTPRoute{
		{Pattern: "POST /auth/login", Public: true, Handler: http.HandlerFunc(h.login)},
		{Pattern: "POST /auth/logout", Handler: http.HandlerFunc(h.logout)},
		{Pattern: "GET /auth/me", Handler: http.HandlerFunc(h.me)},
		{Pattern: "POST /auth/users/{id}/disable", Handler: http.HandlerFunc(h.disableUser)},
		{Pattern: "POST /auth/me/mfa/enroll", Handler: http.HandlerFunc(h.enrollMFA)},
		{Pattern: "POST /auth/me/mfa/enable", Handler: http.HandlerFunc(h.enableMFA)},
	}
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid login request", http.StatusBadRequest)
		return
	}
	if h.LoginLimiter != nil && !h.LoginLimiter.Allow(clientIP(r)+":"+NormalizeEmail(r.FormValue("email"))) {
		http.Error(w, "too many authentication attempts", http.StatusTooManyRequests)
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
	token, session, err := h.Service.CreateSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), DefaultSessionTTL)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	SetSessionCookie(w, token, session.ExpiresAt, h.Secure)
	if h.Audit != nil {
		if err := h.Audit.Record(r.Context(), "auth.login", user.ID); err != nil {
			_ = h.Service.RevokeSession(r.Context(), token)
			ClearSessionCookie(w)
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
				ClearSessionCookie(w)
				http.Error(w, "logout failed", http.StatusInternalServerError)
				return
			}
		}
	}
	ClearSessionCookie(w)
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
		_ = h.Audit.Record(r.Context(), "auth.mfa_enabled", user.ID)
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

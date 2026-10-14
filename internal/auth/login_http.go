package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func (h HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	noStoreAuthResponse(w)
	if h.Service == nil {
		http.Error(w, "authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthFormBytes)
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	var email, password, mfaCode string
	if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			MFACode  string `json:"mfa_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid login request", http.StatusBadRequest)
			return
		}
		email, password, mfaCode = req.Email, req.Password, req.MFACode
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid login request", http.StatusBadRequest)
			return
		}
		email, password, mfaCode = r.FormValue("email"), r.FormValue("password"), r.FormValue("mfa_code")
	}
	if h.LoginLimiter != nil && !security.AllowForRequest(h.LoginLimiter, r.Context(), clientIP(r)) {
		security.WriteRateLimitedResponse(w, "too many authentication attempts", time.Minute)
		return
	}
	user, err := h.Service.Authenticate(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
			if h.LoginSecurity != nil {
				_ = h.LoginSecurity.RecordFailedLogin(r.Context(), email)
			}
			h.recordAuthFailure(r, email)
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
			code := strings.TrimSpace(mfaCode)
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

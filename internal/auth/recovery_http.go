package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

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
	_ = json.NewEncoder(w).Encode(map[string]string{"url": PasswordResetLink(h.PublicOrigin, token)})
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
		http.Redirect(w, r, LoginPagePath+"?"+values.Encode(), http.StatusSeeOther)
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
	http.Redirect(w, r, PasswordResetPagePath+"?"+values.Encode(), http.StatusSeeOther)
}

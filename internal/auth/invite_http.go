package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

func (h HTTPHandler) issueInvite(w http.ResponseWriter, r *http.Request) {
	if !h.adminUserManage(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Invitation == nil || database.RequireExecutor(h.InviteDB) != nil || strings.TrimSpace(h.PublicOrigin) == "" {
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
	// The invitation exists once it is issued. When no mail is configured, or
	// the mail cannot be delivered, the administrator still gets the link in the
	// answer (as with a password-reset link) and can pass it on; "delivered" says
	// which case this is.
	delivered := false
	if !guards.IsNil(h.InviteNotice) {
		if err := h.InviteNotice.SendInvite(r.Context(), NormalizeEmail(r.FormValue("email")), token); err != nil {
			if h.Logger != nil {
				h.Logger.Warn("invitation e-mail not delivered", "error", err.Error())
			}
		} else {
			delivered = true
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"email":     NormalizeEmail(r.FormValue("email")),
		"url":       InviteAcceptLink(h.PublicOrigin, token),
		"delivered": delivered,
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
		http.Redirect(w, r, LoginPagePath+"?"+values.Encode(), http.StatusSeeOther)
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
	http.Redirect(w, r, InviteAcceptPagePath+"?"+values.Encode(), http.StatusSeeOther)
}

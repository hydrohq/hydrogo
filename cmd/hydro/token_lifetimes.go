package main

import (
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type adminTokenLifetimeSettingsHandler struct {
	Settings auth.TokenLifetimeSettings
	Defaults auth.TokenLifetimes
	Audit    auth.AuditRecorder
}

func (h adminTokenLifetimeSettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		h.save(w, r, user)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	value, ok, err := h.Settings.Read(r.Context())
	if err != nil {
		http.Error(w, "Security settings are temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	if !ok {
		value = h.Defaults
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Security token lifetimes", "Security", security.NonceFromContext(r.Context()), "security"); err != nil {
		return
	}
	_ = designsystem.TokenLifetimeForm(designsystem.TokenLifetimeFormProps{
		CSRF: security.CSRFTokenFromContext(r.Context()), Reset: value.PasswordReset.String(), Invitation: value.Invitation.String(), Saved: r.URL.Query().Get("saved") == "1",
	}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h adminTokenLifetimeSettingsHandler) save(w http.ResponseWriter, r *http.Request, user *auth.User) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid security settings", http.StatusBadRequest)
		return
	}
	value := auth.TokenLifetimes{}
	var err error
	value.PasswordReset, err = time.ParseDuration(r.FormValue("password_reset_ttl"))
	if err == nil {
		value.Invitation, err = time.ParseDuration(r.FormValue("invitation_ttl"))
	}
	if err != nil || value.Validate() != nil {
		http.Error(w, "Choose one of the available expiration periods.", http.StatusBadRequest)
		return
	}
	if err := h.Settings.Write(r.Context(), value); err != nil {
		http.Error(w, "Security settings could not be saved.", http.StatusServiceUnavailable)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.token_lifetimes.updated", user.ID)
	}
	http.Redirect(w, r, "/admin/security/token-lifetimes?saved=1", http.StatusSeeOther)
}

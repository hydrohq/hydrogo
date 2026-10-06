package mailer

import (
	"context"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

type EmailBrandingSettings interface {
	EmailBrandingProvider
	WriteEmailBranding(context.Context, EmailBranding) error
}

type EmailBrandingHandler struct {
	Settings EmailBrandingSettings
	Audit    auth.AuditRecorder
}

func (h EmailBrandingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	value, err := h.Settings.ReadEmailBranding(r.Context())
	if err != nil {
		http.Error(w, "Email settings are temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	value, err = value.Normalize()
	if err != nil {
		http.Error(w, "Email settings are temporarily unavailable.", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStartForRequest(w, r, "Email identity", "Mailer", "settings"); err != nil {
		return
	}
	_ = designsystem.EmailBrandingForm(designsystem.EmailBrandingFormProps{
		CSRF: security.CSRFTokenFromContext(r.Context()), LegalName: value.LegalName, PhysicalAddress: value.PhysicalAddress,
		PrivacyPolicyURL: value.PrivacyPolicyURL, FooterNotice: value.FooterNotice, FooterText: value.FooterText,
		LogoAssetID: value.LogoAssetID, PrimaryColor: value.PrimaryColor, Saved: r.URL.Query().Get("saved") == "1",
	}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h EmailBrandingHandler) save(w http.ResponseWriter, r *http.Request, user *auth.User) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid email settings", http.StatusBadRequest)
		return
	}
	value := EmailBranding{
		LegalName: strings.TrimSpace(r.FormValue("legal_name")), PhysicalAddress: strings.TrimSpace(r.FormValue("physical_address")),
		PrivacyPolicyURL: strings.TrimSpace(r.FormValue("privacy_policy_url")), FooterNotice: strings.TrimSpace(r.FormValue("footer_notice")),
		FooterText: strings.TrimSpace(r.FormValue("footer_text")), LogoAssetID: strings.TrimSpace(r.FormValue("logo_asset_id")), PrimaryColor: strings.TrimSpace(r.FormValue("primary_color")),
	}
	value, err := value.Normalize()
	if err != nil {
		http.Error(w, "Check the email settings and try again.", http.StatusBadRequest)
		return
	}
	if err := h.Settings.WriteEmailBranding(r.Context(), value); err != nil {
		http.Error(w, "Email settings could not be saved.", http.StatusServiceUnavailable)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "mailer.branding.updated", user.ID)
	}
	http.Redirect(w, r, "/admin/mailer/branding?saved=1", http.StatusSeeOther)
}

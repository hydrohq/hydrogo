package transport

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

func (h Handler) customCodeAdmin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Role == auth.RoleAdmin && auth.Can(user, auth.PermissionSiteManage)
}

func (h Handler) customCodePage(w http.ResponseWriter, r *http.Request) {
	if !h.customCodeAdmin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	var code site.CustomCode
	if reader, ok := h.Store.(privateSettingReader); ok {
		if setting, found, err := reader.GetSetting(r.Context(), h.DB, site.CustomCodeSettingKey); err == nil && found {
			code = site.CustomCodeFromSetting(setting)
		}
	}
	if ui.AdminStartForRequest(w, r, "Custom code", "Site geneli özel HTML ve izleme kodları", "developer") != nil {
		return
	}
	_, _ = io.WriteString(w, ui.RenderDeveloperTabs("custom-code"))
	_ = designsystem.CustomCodeEditor(code.Head, code.Body, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveCustomCode(w http.ResponseWriter, r *http.Request) {
	if !h.customCodeAdmin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 40<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Kod kaydedilemedi. İçeriği ve boyutunu kontrol edin.", http.StatusBadRequest)
		return
	}
	setting, err := (site.CustomCode{Head: r.FormValue("custom_head_code"), Body: r.FormValue("custom_body_end_code")}).Setting()
	if err != nil {
		http.Error(w, "Kod, desteklenen sağlayıcı kaynaklarıyla sınırlı olmalı.", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveSetting(r.Context(), setting); err != nil {
		http.Error(w, "Kod kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/custom-code?saved=1", http.StatusSeeOther)
}

func (h Handler) maintenanceSettings(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	settings := site.MaintenanceSettings{}
	if reader, ok := h.Store.(privateSettingReader); ok {
		if setting, found, err := reader.GetSetting(r.Context(), h.DB, site.MaintenanceSettingKey); err == nil && found {
			settings = site.MaintenanceFromSetting(setting)
		}
	}
	if ui.AdminStartForRequest(w, r, "Bakım modu", "Site ziyaretçileri için geçici bilgilendirme", "site") != nil {
		return
	}
	_ = designsystem.MaintenanceEditor(settings.Enabled, strings.Join(settings.Allowlist, "\n"), maintenanceFormTime(settings.StartsAt), maintenanceFormTime(settings.EndsAt), settings.Headline, settings.Message, security.CSRFTokenFromContext(r.Context()), settings.PreviewKeyHash != "").Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveMaintenanceSettings(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ayarlar kaydedilemedi.", http.StatusBadRequest)
		return
	}
	entries := strings.FieldsFunc(r.FormValue("allowlist"), func(ch rune) bool { return ch == '\n' || ch == '\r' || ch == ',' || ch == ' ' || ch == '\t' })
	settings := site.MaintenanceSettings{Enabled: r.FormValue("enabled") == "true", Allowlist: entries, Headline: strings.TrimSpace(r.FormValue("headline")), Message: strings.TrimSpace(r.FormValue("message"))}
	for field, target := range map[string]*string{"starts_at": &settings.StartsAt, "ends_at": &settings.EndsAt} {
		value := strings.TrimSpace(r.FormValue(field))
		if value == "" {
			continue
		}
		parsed, parseErr := time.ParseInLocation("2006-01-02T15:04", value, time.UTC)
		if parseErr != nil {
			http.Error(w, "Bakım başlangıç ve bitiş saatlerini kontrol edin.", http.StatusBadRequest)
			return
		}
		*target = parsed.Format(time.RFC3339)
	}
	if reader, ok := h.Store.(privateSettingReader); ok {
		if current, found, err := reader.GetSetting(r.Context(), h.DB, site.MaintenanceSettingKey); err == nil && found {
			settings.PreviewKeyHash = site.MaintenanceFromSetting(current).PreviewKeyHash
		}
	}
	if r.FormValue("clear_preview_key") == "true" {
		settings.PreviewKeyHash = ""
	}
	if key := r.FormValue("preview_key"); key != "" {
		settings.PreviewKeyHash = site.MaintenancePreviewKeyHash(key)
		if settings.PreviewKeyHash == "" {
			http.Error(w, "Önizleme anahtarı 16 ile 128 karakter arasında olmalıdır.", http.StatusBadRequest)
			return
		}
	}
	setting, err := settings.Setting()
	if err != nil {
		http.Error(w, "IP adreslerini veya ağları kontrol edin.", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveSetting(r.Context(), setting); err != nil {
		http.Error(w, "Bakım ayarları kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/maintenance?saved=1", http.StatusSeeOther)
}

func maintenanceFormTime(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return ""
	}
	return parsed.UTC().Format("2006-01-02T15:04")
}

func (h Handler) staffBroadcastNotice(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	notice := site.StaffBroadcastNotice{Severity: "info", ExpiresAt: time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)}
	if reader, ok := h.Store.(privateSettingReader); ok {
		if setting, found, err := reader.GetSetting(r.Context(), h.DB, site.StaffNoticeSettingKey); err == nil && found {
			notice = site.StaffBroadcastNoticeFromSetting(setting)
		}
	}
	if err := ui.AdminStartForRequest(w, r, "Staff notice", "Internal announcement for the team", "developer"); err != nil {
		return
	}
	_, _ = io.WriteString(w, ui.RenderDeveloperTabs("broadcast"))
	_ = designsystem.StaffBroadcastNoticeEditor(notice.Active, notice.Severity, notice.Message, maintenanceFormTime(notice.ExpiresAt), security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveStaffBroadcastNotice(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "Bu sayfaya erişim yetkiniz yok.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Duyuru kaydedilemedi.", http.StatusBadRequest)
		return
	}
	expires, err := time.ParseInLocation("2006-01-02T15:04", r.FormValue("expires_at"), time.UTC)
	if err != nil {
		http.Error(w, "Duyuru bitiş zamanını kontrol edin.", http.StatusBadRequest)
		return
	}
	notice := site.StaffBroadcastNotice{Active: r.FormValue("active") == "true", Severity: strings.TrimSpace(r.FormValue("severity")), Message: strings.TrimSpace(r.FormValue("message")), ExpiresAt: expires.Format(time.RFC3339)}
	setting, err := notice.Setting()
	if err != nil {
		http.Error(w, "Duyuru mesajını ve önem derecesini kontrol edin.", http.StatusBadRequest)
		return
	}
	if err := h.Service.SaveSetting(r.Context(), setting); err != nil {
		http.Error(w, "Duyuru kaydedilemedi.", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/broadcast?saved=1", http.StatusSeeOther)
}

func (h Handler) socialLinks(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "social settings unavailable", http.StatusServiceUnavailable)
		return
	}
	urls := make(map[string]string)
	for _, profile := range site.SocialProfilesFromSettings(settings) {
		urls[profile.Platform] = profile.URL
	}
	if ui.AdminStartForRequest(w, r, "Sosyal bağlantılar", "Marka profillerini yönetin", "site") != nil {
		return
	}
	_ = designsystem.SocialAccounts(designsystem.SocialAccountsProps{CSRF: security.CSRFTokenFromContext(r.Context()), URLs: urls}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveSocialLinks(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid social links", http.StatusBadRequest)
		return
	}
	profiles := make([]site.SocialProfile, 0, len(designsystem.SocialAccountPlatforms()))
	for _, platform := range designsystem.SocialAccountPlatforms() {
		profiles = append(profiles, site.SocialProfile{Platform: platform, URL: r.FormValue("social_" + platform)})
	}
	if err := h.Service.SaveSocialProfiles(r.Context(), profiles); err != nil {
		http.Error(w, "invalid social links", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/site/social-links?saved=1", http.StatusSeeOther)
}

func (h Handler) contactWidget(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil || guards.IsNil(h.Store) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "contact settings unavailable", http.StatusServiceUnavailable)
		return
	}
	value := site.ContactWidgetFromSettings(settings)
	if err := ui.AdminStartForRequest(w, r, "İletişim bileşeni", "Arama ve WhatsApp seçeneklerini yönetin", "site"); err != nil {
		return
	}
	_ = designsystem.ContactWidgetSettings(designsystem.ContactWidgetSettingsProps{CSRF: security.CSRFTokenFromContext(r.Context()), Enabled: value.Enabled, Phone: value.Phone, WhatsAppNumber: value.WhatsAppNumber, Message: value.Message, Position: value.Position, MobileOnly: value.MobileOnly}).Render(r.Context(), w)
	_ = ui.AdminEnd(w)
}

func (h Handler) saveContactWidget(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid contact settings", http.StatusBadRequest)
		return
	}
	value := site.ContactWidget{Enabled: r.FormValue("enabled") == "true", Phone: r.FormValue("phone"), WhatsAppNumber: r.FormValue("whatsapp_number"), Message: r.FormValue("message"), Position: r.FormValue("position"), MobileOnly: r.FormValue("mobile_only") == "true"}
	if value.Position == "" {
		value.Position = "bottom-right"
	}
	if err := h.Service.SaveContactWidget(r.Context(), value); err != nil {
		http.Error(w, "invalid contact settings", siteStatus(err))
		return
	}
	http.Redirect(w, r, "/admin/appearance/contact-widget?saved=1", http.StatusSeeOther)
}

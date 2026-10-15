package main

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

const adminLocaleCookieName = "hydro_admin_locale"

func adminLocaleFromRequest(r *http.Request) string {
	if r != nil {
		if cookie, err := r.Cookie(adminLocaleCookieName); err == nil && i18n.IsSupportedCatalogLocale(cookie.Value) {
			return i18n.ResolveCatalogLocale(cookie.Value)
		}
	}
	return i18n.ResolveCatalogLocale("")
}

func setAdminLocaleCookie(w http.ResponseWriter, locale string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: adminLocaleCookieName, Value: i18n.ResolveCatalogLocale(locale), Path: "/", MaxAge: int((365 * 24 * time.Hour).Seconds()), Secure: secure, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

// adminAccountLocaleHandler stores the admin UI language preference.
func (a *serveApp) adminAccountLocaleHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	var input struct {
		Locale string `json:"locale"`
	}
	if !api.DecodeJSON(w, r, &input, 4<<10) {
		return
	}
	locale := strings.TrimSpace(input.Locale)
	if !i18n.IsSupportedCatalogLocale(locale) {
		api.WriteValidation(w, r, map[string]string{"locale": "invalid"})
		return
	}
	setAdminLocaleCookie(w, locale, a.cfg.Security.SecureCookies)
	api.WriteData(w, http.StatusOK, map[string]string{"locale": i18n.ResolveCatalogLocale(locale)}, nil)
}

// adminI18nHandler serves the admin catalog for one locale as a flat map.
func (a *serveApp) adminI18nHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	requested := r.PathValue("locale")
	if !i18n.IsSupportedCatalogLocale(requested) {
		api.WriteError(w, r, http.StatusNotFound, "unsupported locale", nil)
		return
	}
	tenantID := ""
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	}
	rows := i18n.CatalogRows(tenantID, requested)
	messages := make(map[string]string, len(rows))
	for _, row := range rows {
		messages[row.Key] = row.Translation
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, messages, map[string]any{"locale": i18n.ResolveCatalogLocale(requested), "supported": i18n.SupportedCatalogLocales()})
}

type adminSystemInfo struct {
	Version         string          `json:"version"`
	GoVersion       string          `json:"go_version"`
	Dialect         string          `json:"dialect"`
	CapacityProfile string          `json:"capacity_profile"`
	DevMode         bool            `json:"dev_mode"`
	SafeMode        bool            `json:"safe_mode"`
	Modules         map[string]bool `json:"modules"`
	PublicOrigin    string          `json:"public_origin,omitempty"`
	TenancyEnabled  bool            `json:"tenancy_enabled"`
	Support         adminSupport    `json:"support"`
	StaffNotice     *adminNotice    `json:"staff_notice,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
}

type adminSupport struct {
	Enabled     bool   `json:"enabled"`
	AgencyName  string `json:"agency_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	WhatsAppURL string `json:"whatsapp_url,omitempty"`
	SLABadge    string `json:"sla_badge,omitempty"`
	GuideURL    string `json:"guide_url,omitempty"`
}

type adminNotice struct {
	Message    string `json:"message"`
	Severity   string `json:"severity"`
	DismissKey string `json:"dismiss_key"`
}

// adminSystemInfoHandler describes the running instance: version, database,
// enabled modules, support contact and any active staff broadcast notice.
func (a *serveApp) adminSystemInfoHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	cfg := a.cfg
	info := adminSystemInfo{Version: version, GoVersion: runtime.Version(), Dialect: string(cfg.Database.Dialect), CapacityProfile: string(cfg.Profile), DevMode: a.devMode, SafeMode: cfg.SafeMode, Modules: enabledOnly(a.modules), PublicOrigin: cfg.HTTP.PublicOrigin, TenancyEnabled: a.requireTenantScope, Support: adminSupport{Enabled: cfg.AdminSupport.Enabled, AgencyName: cfg.AdminSupport.AgencyName, Email: cfg.AdminSupport.Email, Phone: cfg.AdminSupport.Phone, WhatsAppURL: cfg.AdminSupport.WhatsAppURL, SLABadge: cfg.AdminSupport.SLABadge, GuideURL: cfg.AdminSupport.GuideURL}}
	if a.metrics != nil {
		info.StartedAt = a.metrics.StartedAt()
	}
	if support := adminstudio.SupportConfigFromContext(r.Context()); support.StaffNoticeActive {
		info.StaffNotice = &adminNotice{Message: support.StaffNoticeMessage, Severity: support.StaffNoticeSeverity, DismissKey: support.StaffNoticeDismissKey}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, info, nil)
}

// adminSystemMetricsHandler exposes the in-process runtime snapshot.
func (a *serveApp) adminSystemMetricsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if !activePlatformAdmin(user) || !auth.Can(user, auth.PermissionOperations) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, a.metrics.Snapshot(), nil)
}

// adminSystemHealthHandler runs the readiness check the load balancer uses
// and reports the outcome as JSON.
func (a *serveApp) adminSystemHealthHandler(ready func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		status := map[string]any{"ok": true, "checked_at": time.Now().UTC()}
		if ready != nil {
			checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ready(checkCtx); err != nil {
				status["ok"] = false
				status["error"] = err.Error()
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		api.WriteData(w, http.StatusOK, status, nil)
	}
}

type adminClientError struct {
	Message   string `json:"message"`
	Stack     string `json:"stack"`
	URL       string `json:"url"`
	UserAgent string `json:"user_agent"`
}

// adminClientErrorHandler records browser-side failures next to the server
// logs so one request id links both sides.
func (a *serveApp) adminClientErrorHandler(limiter security.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active {
			api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		if !security.AllowForRequest(limiter, r.Context(), "client-errors:"+user.ID) {
			api.WriteError(w, r, http.StatusTooManyRequests, "too many error reports", nil)
			return
		}
		var input adminClientError
		if !api.DecodeJSON(w, r, &input, 64<<10) {
			return
		}
		input.Message = strings.TrimSpace(input.Message)
		if input.Message == "" || len(input.Message) > 2000 {
			api.WriteValidation(w, r, map[string]string{"message": "invalid"})
			return
		}
		if len(input.Stack) > 16000 {
			input.Stack = input.Stack[:16000]
		}
		logger := a.logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.Warn("admin client error", "request_id", security.RequestIDFromContext(r.Context()), "user_id", user.ID, "message", input.Message, "url", input.URL, "user_agent", input.UserAgent, "stack", input.Stack)
		w.WriteHeader(http.StatusAccepted)
	}
}

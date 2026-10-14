package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/geolocation"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Route struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

type Handler struct {
	DB                database.Executor
	Store             site.Store
	Service           *site.Service
	Media             *media.Service
	SubmissionLimiter security.RateLimiter
	FormViewLimiter   security.RateLimiter
	FormMetrics       analytics.FormMetrics
	VisitorHasher     *analytics.VisitorHasher
	AllowedForms      map[string]struct{}
	FormsEnabled      bool
	Turnstile         interface {
		Verify(context.Context, string, string) (bool, error)
	}
	GeoResolver interface {
		ResolveRequest(*http.Request) geolocation.Location
	}
	MailSpool interface {
		List(context.Context, string, int) ([]mailer.SpoolEntry, error)
		RetryNow(context.Context, string) error
	}
	Redirects             *site.RedirectRegistry
	SitemapOrigin         string
	SitemapProvider       func(context.Context) ([]site.SitemapEntry, error)
	SitemapStream         site.SitemapStream
	SitemapCategories     []string
	InvalidateDiscovery   func()
	FontsDir              string
	FontsEnabled          bool
	fontHTTPClient        *http.Client
	NavigationLocales     []string
	NavigationSuggestions func(context.Context, string) ([]site.NavigationSuggestion, error)
	Realtime              interface {
		Publish(context.Context, realtime.Event) error
	}
	LinkScanManager    *linkcheck.Manager
	LinkScanLimiter    security.RateLimiter
	RequireTenantScope bool
	Assistant          ai.Assistant
	AIAudit            audit.Writer
}

func (h Handler) assistantEnabled() bool {
	return h.Assistant.Enabled && h.Assistant.Provider != nil && h.AIAudit != nil
}

func submissionLocation(item site.Submission) string {
	if item.LocationLocal {
		return "Yerel ağ"
	}
	if item.LocationCountryCode == "" {
		return "—"
	}
	label := geolocation.CountryFlag(item.LocationCountryCode) + " " + item.LocationCountryCode
	if item.LocationCity != "" {
		label = geolocation.CountryFlag(item.LocationCountryCode) + " " + item.LocationCity
	}
	return label
}

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "GET /site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
		{Pattern: "GET /sitemap-index.xml", Public: true, Handler: http.HandlerFunc(h.sitemapIndex)},
		{Pattern: "GET /sitemap-pages.xml", Public: true, Handler: http.HandlerFunc(h.sitemapPart)},
		{Pattern: "GET /sitemap-articles.xml", Public: true, Handler: http.HandlerFunc(h.sitemapArticlePart)},
		{Pattern: "POST /admin/site/settings", Handler: http.HandlerFunc(h.saveSetting)},
		{Pattern: "GET /admin/translations", Handler: http.HandlerFunc(h.translations)},
		{Pattern: "POST /admin/translations", Handler: http.HandlerFunc(h.saveTranslation)},
		{Pattern: "GET /admin/site/appearance", Handler: http.HandlerFunc(h.appearance)},
		{Pattern: "POST /admin/site/appearance", Handler: http.HandlerFunc(h.saveAppearance)},
		{Pattern: "GET /admin/site/error-pages", Handler: http.HandlerFunc(h.errorPages)},
		{Pattern: "POST /admin/site/error-pages/{status}", Handler: http.HandlerFunc(h.saveErrorPage)},
		{Pattern: "GET /admin/site/maintenance", Handler: http.HandlerFunc(h.maintenanceSettings)},
		{Pattern: "POST /admin/site/maintenance", Handler: http.HandlerFunc(h.saveMaintenanceSettings)},
		{Pattern: "GET /admin/site/broadcast", Handler: http.HandlerFunc(h.staffBroadcastNotice)},
		{Pattern: "POST /admin/site/broadcast", Handler: http.HandlerFunc(h.saveStaffBroadcastNotice)},
		{Pattern: "GET /admin/site/custom-code", Handler: http.HandlerFunc(h.customCodePage)},
		{Pattern: "POST /admin/site/custom-code", Handler: http.HandlerFunc(h.saveCustomCode)},
		{Pattern: "GET /admin/site/link-check", Handler: http.HandlerFunc(h.linkCheckPage)},
		{Pattern: "POST /admin/site/link-check", Handler: http.HandlerFunc(h.runLinkCheck)},
		{Pattern: "GET /admin/appearance/contact-widget", Handler: http.HandlerFunc(h.contactWidget)},
		{Pattern: "POST /admin/appearance/contact-widget", Handler: http.HandlerFunc(h.saveContactWidget)},
		{Pattern: "GET /admin/site/social-links", Handler: http.HandlerFunc(h.socialLinks)},
		{Pattern: "POST /admin/site/social-links", Handler: http.HandlerFunc(h.saveSocialLinks)},
		{Pattern: "GET /admin/site/seo", Handler: http.HandlerFunc(h.seoSettings)},
		{Pattern: "POST /admin/site/seo", Handler: http.HandlerFunc(h.saveSEOSettings)},
		{Pattern: "POST /admin/site/redirects", Handler: http.HandlerFunc(h.saveRedirect)},
		{Pattern: "GET /admin/site/redirect-rules", Handler: http.HandlerFunc(h.redirectRules)},
		{Pattern: "GET /admin/redirects", Handler: http.HandlerFunc(h.notFoundPaths)},
		{Pattern: "POST /admin/redirects/convert", Handler: http.HandlerFunc(h.convertNotFoundPath)},
		{Pattern: "POST /admin/site/redirect-rules", Handler: http.HandlerFunc(h.saveRedirectRule)},
		{Pattern: "POST /admin/site/redirect-rules/import", Handler: http.HandlerFunc(h.importRedirectRules)},
		{Pattern: "POST /admin/site/redirect-rules/{id}/delete", Handler: http.HandlerFunc(h.deleteRedirectRule)},
		{Pattern: "POST /admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.saveNavigation)},
		{Pattern: "GET /admin/site/navigation/editor", Handler: http.HandlerFunc(h.navigationEditor)},
		{Pattern: "POST /admin/site/navigation/editor", Handler: http.HandlerFunc(h.saveNavigationEditor)},
		{Pattern: "GET /{path...}", Public: true, Handler: http.HandlerFunc(h.redirectFallback)},
	}
	if h.FontsEnabled {
		routes = append(routes,
			Route{Pattern: "GET /admin/site/fonts", Handler: http.HandlerFunc(h.fontPicker)},
			Route{Pattern: "POST /admin/site/fonts", Handler: http.HandlerFunc(h.localizeFont)},
		)
	}
	if h.FormsEnabled {
		routes = append(routes,
			Route{Pattern: "GET /admin/site/inbox/{form}", Handler: http.HandlerFunc(h.inbox)},
			Route{Pattern: "GET /admin/site/forms", Handler: http.HandlerFunc(h.forms)},
			Route{Pattern: "POST /admin/site/forms", Handler: http.HandlerFunc(h.saveForm)},
			Route{Pattern: "POST /admin/site/forms/{id}/working-hours", Handler: http.HandlerFunc(h.saveFormWorkingHours)},
			Route{Pattern: "POST /admin/site/forms/{id}/delete", Handler: http.HandlerFunc(h.deleteForm)},
			Route{Pattern: "POST /admin/site/inbox/{id}/status", Handler: http.HandlerFunc(h.inboxStatus)},
			Route{Pattern: "POST /admin/site/inbox/{id}/lead", Handler: http.HandlerFunc(h.leadUpdate)},
			Route{Pattern: "POST /admin/site/inbox/{id}/notes", Handler: http.HandlerFunc(h.leadNoteCreate)},
			Route{Pattern: "GET /admin/site/inbox/{id}/notes", Handler: http.HandlerFunc(h.leadNotes)},
			Route{Pattern: "GET /admin/site/inbox/{form}/export.csv", Handler: http.HandlerFunc(h.exportInboxCSV)},
			Route{Pattern: "GET /admin/site/inbox/{form}/export.xlsx", Handler: http.HandlerFunc(h.exportInboxXLSX)},
			Route{Pattern: "GET /admin/site/mail", Handler: http.HandlerFunc(h.mailSpool)},
			Route{Pattern: "POST /admin/site/mail/{id}/retry", Handler: http.HandlerFunc(h.mailSpoolRetry)},
			Route{Pattern: "POST /forms/{form}/submit", Public: true, Handler: http.HandlerFunc(h.submitForm)},
			Route{Pattern: "POST /forms/{form}/view", Public: true, Handler: http.HandlerFunc(h.recordFormView)},
		)
		if h.assistantEnabled() {
			routes = append(routes, Route{Pattern: "POST /admin/site/inbox/{form}/ai-summarize", Handler: http.HandlerFunc(h.summarizeInboxAI)})
		}
	}
	return routes
}

type privateSettingReader interface {
	GetSetting(context.Context, database.Executor, string) (site.Setting, bool, error)
}

func randomID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}

func (h Handler) saveSetting(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value site.Setting
	if !decodeJSON(w, r, &value) {
		return
	}
	if err := h.Service.SaveSetting(r.Context(), value); err != nil {
		http.Error(w, "invalid site setting", siteStatus(err))
		return
	}
	if h.InvalidateDiscovery != nil {
		h.InvalidateDiscovery()
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) saveNavigation(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var items []site.NavigationItem
	if !decodeJSON(w, r, &items) {
		return
	}
	if err := h.Service.SaveNavigation(r.Context(), r.PathValue("locale"), items); err != nil {
		http.Error(w, "invalid navigation", siteStatus(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) admin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionSiteManage)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := security.DecodeJSONStrict(w, r, target, 128<<10); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	return true
}

func siteStatus(err error) int {
	if errors.Is(err, site.ErrIdempotencyConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, site.ErrIdempotencyStoreRequired) || errors.Is(err, site.ErrLeadStoreRequired) || errors.Is(err, site.ErrMailQueueRequired) || errors.Is(err, site.ErrRedirectStoreRequired) || errors.Is(err, mailer.ErrSpoolLeaseInvalid) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, site.ErrStoreRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadRequest
}

func (h Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "site settings unavailable", http.StatusServiceUnavailable)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		http.Error(w, "site settings unavailable", http.StatusInternalServerError)
		return
	}
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(result)
}

func (h Handler) navigation(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		http.Error(w, "navigation unavailable", http.StatusServiceUnavailable)
		return
	}
	items, err := h.Store.ListNavigation(r.Context(), h.DB, r.PathValue("locale"))
	if err != nil {
		http.Error(w, "navigation unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(items)
}

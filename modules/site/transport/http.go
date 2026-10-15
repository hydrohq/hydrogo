package transport

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	formstore "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/geolocation"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Route struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

type Handler struct {
	DB                  database.Executor
	Store               site.Store
	Service             *site.Service
	Redirects           *site.RedirectRegistry
	SitemapOrigin       string
	SitemapProvider     func(context.Context) ([]site.SitemapEntry, error)
	SitemapStream       site.SitemapStream
	SitemapCategories   []string
	InvalidateDiscovery func()
	NavigationLocales   []string
	RequireTenantScope  bool

	// Forms (only when the forms module is enabled).
	FormsEnabled      bool
	AllowedForms      map[string]struct{}
	Media             *media.Service
	SubmissionLimiter security.RateLimiter
	FormViewLimiter   security.RateLimiter
	FormMetrics       analytics.FormMetrics
	VisitorHasher     *analytics.VisitorHasher
	Turnstile         interface {
		Verify(context.Context, string, string) (bool, error)
	}
	GeoResolver interface {
		ResolveRequest(*http.Request) geolocation.Location
	}
	Realtime interface {
		Publish(context.Context, realtime.Event) error
	}
}

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "GET /api/v1/site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /api/v1/site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /api/v1/site/redirects", Public: true, Handler: http.HandlerFunc(h.publicRedirects)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
		{Pattern: "GET /sitemap-index.xml", Public: true, Handler: http.HandlerFunc(h.sitemapIndex)},
		{Pattern: "GET /sitemap-pages.xml", Public: true, Handler: http.HandlerFunc(h.sitemapPart)},
		{Pattern: "GET /sitemap-articles.xml", Public: true, Handler: http.HandlerFunc(h.sitemapArticlePart)},

		{Pattern: "GET /api/v1/admin/site/settings", Handler: http.HandlerFunc(h.adminSettings)},
		{Pattern: "PUT /api/v1/admin/site/settings", Handler: http.HandlerFunc(h.adminSaveSettings)},
		{Pattern: "GET /api/v1/admin/site/maintenance", Handler: http.HandlerFunc(h.adminMaintenance)},
		{Pattern: "PUT /api/v1/admin/site/maintenance", Handler: http.HandlerFunc(h.adminSaveMaintenance)},
		{Pattern: "GET /api/v1/admin/site/staff-notice", Handler: http.HandlerFunc(h.adminStaffNotice)},
		{Pattern: "PUT /api/v1/admin/site/staff-notice", Handler: http.HandlerFunc(h.adminSaveStaffNotice)},
		{Pattern: "GET /api/v1/admin/site/social-links", Handler: http.HandlerFunc(h.adminSocialLinks)},
		{Pattern: "PUT /api/v1/admin/site/social-links", Handler: http.HandlerFunc(h.adminSaveSocialLinks)},
		{Pattern: "GET /api/v1/admin/site/business-profile", Handler: http.HandlerFunc(h.adminBusinessProfile)},
		{Pattern: "PUT /api/v1/admin/site/business-profile", Handler: http.HandlerFunc(h.adminSaveBusinessProfile)},
		{Pattern: "GET /api/v1/admin/site/translations/{locale}", Handler: http.HandlerFunc(h.adminTranslations)},
		{Pattern: "PUT /api/v1/admin/site/translations/{locale}", Handler: http.HandlerFunc(h.adminSaveTranslations)},
		{Pattern: "GET /api/v1/admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.adminNavigation)},
		{Pattern: "PUT /api/v1/admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.adminSaveNavigation)},
		{Pattern: "GET /api/v1/admin/site/redirects", Handler: http.HandlerFunc(h.adminRedirects)},
		{Pattern: "POST /api/v1/admin/site/redirects", Handler: http.HandlerFunc(h.adminCreateRedirect)},
		{Pattern: "POST /api/v1/admin/site/redirects/import", Handler: http.HandlerFunc(h.adminImportRedirects)},
		{Pattern: "PUT /api/v1/admin/site/redirects/{id}", Handler: http.HandlerFunc(h.adminUpdateRedirect)},
		{Pattern: "DELETE /api/v1/admin/site/redirects/{id}", Handler: http.HandlerFunc(h.adminDeleteRedirect)},
		{Pattern: "GET /api/v1/admin/site/not-found", Handler: http.HandlerFunc(h.adminNotFound)},
		{Pattern: "POST /api/v1/admin/site/not-found/convert", Handler: http.HandlerFunc(h.adminConvertNotFound)},
		{Pattern: "DELETE /api/v1/admin/site/not-found", Handler: http.HandlerFunc(h.adminDeleteNotFound)},
	}
	if h.FormsEnabled {
		routes = append(routes,
			Route{Pattern: "GET /api/v1/forms/{form}", Public: true, Handler: http.HandlerFunc(h.publicForm)},
			Route{Pattern: "POST /api/v1/forms/{form}/submit", Public: true, Handler: http.HandlerFunc(h.submitForm)},
			Route{Pattern: "POST /api/v1/forms/{form}/view", Public: true, Handler: http.HandlerFunc(h.recordFormView)},
			Route{Pattern: "GET /api/v1/admin/site/forms", Handler: http.HandlerFunc(h.adminForms)},
			Route{Pattern: "POST /api/v1/admin/site/forms", Handler: http.HandlerFunc(h.adminCreateForm)},
			Route{Pattern: "GET /api/v1/admin/site/forms/{id}", Handler: http.HandlerFunc(h.adminForm)},
			Route{Pattern: "PUT /api/v1/admin/site/forms/{id}", Handler: http.HandlerFunc(h.adminUpdateForm)},
			Route{Pattern: "DELETE /api/v1/admin/site/forms/{id}", Handler: http.HandlerFunc(h.adminDeleteForm)},
			Route{Pattern: "GET /api/v1/admin/site/inbox", Handler: http.HandlerFunc(h.adminInbox)},
			Route{Pattern: "GET /api/v1/admin/site/inbox/export.csv", Handler: http.HandlerFunc(h.adminInboxExportCSV)},
			Route{Pattern: "GET /api/v1/admin/site/inbox/export.xlsx", Handler: http.HandlerFunc(h.adminInboxExportXLSX)},
			Route{Pattern: "POST /api/v1/admin/site/inbox/purge", Handler: http.HandlerFunc(h.adminPurgeInbox)},
			Route{Pattern: "GET /api/v1/admin/site/inbox/{id}", Handler: http.HandlerFunc(h.adminInboxItem)},
			Route{Pattern: "PATCH /api/v1/admin/site/inbox/{id}", Handler: http.HandlerFunc(h.adminUpdateInboxItem)},
			Route{Pattern: "GET /api/v1/admin/site/inbox/{id}/notes", Handler: http.HandlerFunc(h.adminInboxNotes)},
			Route{Pattern: "POST /api/v1/admin/site/inbox/{id}/notes", Handler: http.HandlerFunc(h.adminAddInboxNote)},
		)
	}
	return routes
}

// requireSiteManage answers 403 unless the signed-in user may manage the
// site and the service is wired.
func (h Handler) requireSiteManage(w http.ResponseWriter, r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	if !auth.Can(user, auth.PermissionSiteManage) {
		api.WriteError(w, r, http.StatusForbidden, "site management permission required", nil)
		return false
	}
	if h.Service == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "site service unavailable", nil)
		return false
	}
	return true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	return api.DecodeJSON(w, r, target, 256<<10)
}

// writeSiteError maps module errors onto the API envelope.
func writeSiteError(w http.ResponseWriter, r *http.Request, err error, message string) {
	switch {
	case errors.Is(err, sql.ErrNoRows), errors.Is(err, formstore.ErrFormNotFound):
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
	case errors.Is(err, site.ErrIdempotencyConflict):
		api.WriteError(w, r, http.StatusConflict, message, nil)
	case errors.Is(err, site.ErrInvalidSetting), errors.Is(err, site.ErrInvalidNavigation), errors.Is(err, site.ErrInvalidRedirect), errors.Is(err, site.ErrInvalidInboxState), errors.Is(err, site.ErrInvalidSubmission), errors.Is(err, site.ErrInvalidIdempotencyKey):
		api.WriteError(w, r, http.StatusUnprocessableEntity, message, nil)
	case errors.Is(err, site.ErrStoreRequired), errors.Is(err, site.ErrRedirectStoreRequired), errors.Is(err, site.ErrLeadStoreRequired), errors.Is(err, site.ErrMailQueueRequired), errors.Is(err, site.ErrIdempotencyStoreRequired), errors.Is(err, mailer.ErrSpoolLeaseInvalid), errors.Is(err, database.ErrDatabaseRequired):
		api.WriteError(w, r, http.StatusServiceUnavailable, message, nil)
	default:
		api.WriteError(w, r, http.StatusUnprocessableEntity, message, nil)
	}
}

func randomID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}

func requestIP(r *http.Request) string { return requestmeta.ClientIP(r) }

func (h Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "site settings unavailable", nil)
		return
	}
	settings, err := h.Store.ListPublicSettings(r.Context(), h.DB)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "site settings unavailable", nil)
		return
	}
	result := make(map[string]string, len(settings))
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}
	api.WriteData(w, http.StatusOK, result, nil)
}

func (h Handler) navigation(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Store) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "navigation unavailable", nil)
		return
	}
	items, err := h.Store.ListNavigation(r.Context(), h.DB, r.PathValue("locale"))
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "navigation unavailable", nil)
		return
	}
	if items == nil {
		items = []site.NavigationItem{}
	}
	api.WriteData(w, http.StatusOK, items, nil)
}

type publicRedirect struct {
	Pattern    string `json:"pattern"`
	Kind       string `json:"kind"`
	TargetPath string `json:"target_path"`
	Status     int    `json:"status"`
}

// publicRedirects lists the active rules so a headless frontend can apply
// them; hit counts and ids stay private.
func (h Handler) publicRedirects(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "redirects unavailable", nil)
		return
	}
	rules, err := h.Service.ListRedirectRules(r.Context(), 1000)
	if err != nil {
		writeSiteError(w, r, err, "redirects unavailable")
		return
	}
	result := make([]publicRedirect, 0, len(rules))
	for _, rule := range rules {
		result = append(result, publicRedirect{Pattern: rule.Pattern, Kind: rule.Kind, TargetPath: rule.TargetPath, Status: rule.Status})
	}
	api.WriteData(w, http.StatusOK, result, nil)
}

package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
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
	RequireTenantScope  bool
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
		{Pattern: "GET /sitemap-index.xml", Public: true, Handler: http.HandlerFunc(h.sitemapIndex)},
		{Pattern: "GET /sitemap-pages.xml", Public: true, Handler: http.HandlerFunc(h.sitemapPart)},
		{Pattern: "GET /sitemap-articles.xml", Public: true, Handler: http.HandlerFunc(h.sitemapArticlePart)},
		{Pattern: "POST /admin/site/settings", Handler: http.HandlerFunc(h.saveSetting)},
		{Pattern: "POST /admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.saveNavigation)},
	}
}

type privateSettingReader interface {
	GetSetting(context.Context, database.Executor, string) (site.Setting, bool, error)
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

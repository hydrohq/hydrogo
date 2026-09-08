package transport

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Route struct {
	Pattern string
	Public  bool
	Handler http.Handler
}

type Handler struct {
	DB              database.Executor
	Store           site.Store
	Service         *site.Service
	SitemapOrigin   string
	SitemapProvider func(context.Context) ([]site.SitemapEntry, error)
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
		{Pattern: "POST /admin/site/settings", Handler: http.HandlerFunc(h.saveSetting)},
		{Pattern: "POST /admin/site/redirects", Handler: http.HandlerFunc(h.saveRedirect)},
		{Pattern: "POST /admin/site/navigation/{locale}", Handler: http.HandlerFunc(h.saveNavigation)},
	}
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
		http.Error(w, "invalid site setting", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) saveRedirect(w http.ResponseWriter, r *http.Request) {
	if !h.admin(r) || h.Service == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value site.Redirect
	if !decodeJSON(w, r, &value) {
		return
	}
	if err := h.Service.SaveRedirect(r.Context(), value); err != nil {
		http.Error(w, "invalid redirect", http.StatusBadRequest)
		return
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
		http.Error(w, "invalid navigation", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) admin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return false
	}
	return true
}

func (h Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	if h.SitemapProvider == nil || h.SitemapOrigin == "" {
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return
	}
	entries, err := h.SitemapProvider(r.Context())
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	data, err := site.RenderSitemap(entries)
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(data)
}

func (h Handler) publicSettings(w http.ResponseWriter, r *http.Request) {
	if h.DB == nil || h.Store == nil {
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
	if h.DB == nil || h.Store == nil {
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

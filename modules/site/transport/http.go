package transport

import (
	"context"
	"encoding/json"
	"net/http"

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
	SitemapOrigin   string
	SitemapProvider func(context.Context) ([]site.SitemapEntry, error)
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /site/settings", Public: true, Handler: http.HandlerFunc(h.publicSettings)},
		{Pattern: "GET /site/navigation/{locale}", Public: true, Handler: http.HandlerFunc(h.navigation)},
		{Pattern: "GET /sitemap.xml", Public: true, Handler: http.HandlerFunc(h.sitemap)},
	}
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

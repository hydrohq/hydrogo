package transport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

// GET /api/v1/news?locale=&limit= lists published items for a locale.
func (h Handler) publicList(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w, r) {
		return
	}
	repository, ok := h.Service.Repository.(publishedListRepository)
	if !ok {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news list is unavailable", nil)
		return
	}
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if locale == "" {
		api.WriteError(w, r, http.StatusBadRequest, "locale is required", nil)
		return
	}
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			api.WriteError(w, r, http.StatusBadRequest, "limit must be between 1 and 100", nil)
			return
		}
		limit = parsed
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	var (
		items []news.Item
		err   error
	)
	if filterRepo, ok := h.Service.Repository.(publishedCategoryFilterRepository); ok && category != "" {
		items, err = filterRepo.ListPublishedFiltered(r.Context(), h.DB, locale, category, limit)
	} else {
		items, err = repository.ListPublished(r.Context(), h.DB, locale, limit)
	}
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news is unavailable", nil)
		return
	}
	api.WriteList(w, items, api.PaginationMeta{Limit: limit, HasMore: len(items) == limit})
}

// GET /api/v1/news/{locale}/{slug}
func (h Handler) publicGet(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.PathValue("locale"), r.PathValue("slug"))
}

// GET /api/v1/news/by-slug?locale=&slug=
func (h Handler) publicGetByQuery(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"))
}

func (h Handler) publicGetWithRoute(w http.ResponseWriter, r *http.Request, rawLocale, rawSlug string) {
	if !h.ready(w, r) {
		return
	}
	locale := canonical.Locale(rawLocale)
	slug, err := canonical.Path(rawSlug)
	if locale == "" || err != nil || slug == "" {
		api.WriteError(w, r, http.StatusNotFound, "news item not found", nil)
		return
	}
	item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			api.WriteError(w, r, http.StatusServiceUnavailable, "news is unavailable", nil)
			return
		}
		api.WriteError(w, r, http.StatusNotFound, "news item not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, item, nil)
}

// GET /api/v1/news/{id}/preview?token= returns the item a preview link was
// issued for, in any status, as long as the revision still matches.
func (h Handler) publicPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if h.PreviewSigner == nil || !h.ready(w, r) {
		if h.PreviewSigner == nil {
			api.WriteError(w, r, http.StatusNotFound, "preview not found", nil)
		}
		return
	}
	claims, err := h.PreviewSigner.Verify(r.URL.Query().Get("token"), time.Now().UTC())
	if err != nil || claims.ResourceType != "news" || claims.ResourceID != r.PathValue("id") {
		api.WriteError(w, r, http.StatusNotFound, "preview not found", nil)
		return
	}
	if h.RequireTenantScope {
		scope, e := tenant.Require(r.Context())
		if e != nil || scope.TenantID != claims.TenantID {
			api.WriteError(w, r, http.StatusNotFound, "preview not found", nil)
			return
		}
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, claims.ResourceID)
	if err != nil || item.Revision != claims.Revision {
		api.WriteError(w, r, http.StatusNotFound, "preview not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, item, nil)
}

func previewable(status news.Status) bool {
	return status == news.StatusPublished || status == news.StatusDraft || status == news.StatusScheduled || status == news.StatusRetracted
}

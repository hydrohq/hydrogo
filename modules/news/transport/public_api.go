package transport

import (
	"context"
	"encoding/json"
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

type contentViewIncrementer interface {
	IncrementContentView(context.Context, string, string, string, string) (int64, error)
}

type contentViewReader interface {
	ContentViewTotal(context.Context, string, string, string) (int64, error)
}

type viewInput struct {
	Slug string `json:"slug"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (h Handler) publicRecordView(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	var in viewInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid json body", nil)
		return
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = strings.TrimSpace(in.ID)
	}
	if slug == "" {
		api.WriteError(w, r, http.StatusBadRequest, "slug is required", nil)
		return
	}
	contentType := strings.TrimSpace(in.Type)
	if contentType == "" {
		contentType = "news"
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	}
	locale := h.defaultLocale()
	contentID := slug
	if contentType == "news" && h.Service.Repository != nil && h.DB != nil {
		if item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug); err == nil && item.ID != "" {
			contentID = item.ID
		}
	}
	var total int64 = 1
	if incrementer, ok := h.ContentViews.(contentViewIncrementer); ok && h.ContentViews != nil {
		if count, err := incrementer.IncrementContentView(r.Context(), tenantID, contentType, contentID, locale); err == nil {
			total = count
		}
	}
	api.WriteData(w, http.StatusOK, map[string]any{
		"slug":        slug,
		"views":       total,
		"incremented": true,
	}, nil)
}

func (h Handler) publicGetViews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, r, http.StatusMethodNotAllowed, "method not allowed", nil)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	}
	contentType := strings.TrimSpace(r.URL.Query().Get("type"))
	if contentType == "" {
		contentType = "news"
	}
	locale := h.defaultLocale()
	slugsParam := strings.TrimSpace(r.URL.Query().Get("slugs"))
	if slugsParam != "" {
		rawList := strings.Split(slugsParam, ",")
		slugs := make([]string, 0, len(rawList))
		for _, s := range rawList {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				slugs = append(slugs, trimmed)
			}
		}
		result := make(map[string]int64, len(slugs))
		if len(slugs) > 0 && h.ContentViews != nil {
			lookupIDs := make([]string, len(slugs))
			idToSlug := make(map[string]string, len(slugs))
			for i, slug := range slugs {
				contentID := slug
				if contentType == "news" && h.Service.Repository != nil && h.DB != nil {
					if item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug); err == nil && item.ID != "" {
						contentID = item.ID
					}
				}
				lookupIDs[i] = contentID
				idToSlug[contentID] = slug
			}
			if counts, err := h.ContentViews.ContentViewCounts(r.Context(), tenantID, contentType, lookupIDs); err == nil {
				for id, count := range counts {
					if origSlug, ok := idToSlug[id]; ok {
						result[origSlug] = count
					}
				}
			}
		}
		api.WriteData(w, http.StatusOK, result, nil)
		return
	}

	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		slug = strings.TrimSpace(r.URL.Query().Get("id"))
	}
	if slug == "" {
		api.WriteError(w, r, http.StatusBadRequest, "slug is required", nil)
		return
	}
	contentID := slug
	if contentType == "news" && h.Service.Repository != nil && h.DB != nil {
		if item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug); err == nil && item.ID != "" {
			contentID = item.ID
		}
	}
	var total int64
	if reader, ok := h.ContentViews.(contentViewReader); ok && h.ContentViews != nil {
		if count, err := reader.ContentViewTotal(r.Context(), tenantID, contentType, contentID); err == nil {
			total = count
		}
	} else if h.ContentViews != nil {
		if counts, err := h.ContentViews.ContentViewCounts(r.Context(), tenantID, contentType, []string{contentID}); err == nil {
			total = counts[contentID]
		}
	}
	api.WriteData(w, http.StatusOK, map[string]any{
		"slug":  slug,
		"views": total,
	}, nil)
}

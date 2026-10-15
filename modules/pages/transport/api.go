package transport

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

// pageSummary is the intentionally small public discovery projection. Detail
// content remains behind the detail endpoints so list clients do not mistake
// empty draft fields for published content.
type pageSummary struct {
	ID                    string    `json:"id"`
	Locale                string    `json:"locale"`
	Slug                  string    `json:"slug"`
	Title                 string    `json:"title"`
	Revision              int       `json:"revision"`
	PublicationGeneration int64     `json:"publication_generation"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// GET /api/v1/pages?locale=&limit= lists published pages as summaries.
func (h Handler) listPublished(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			api.WriteError(w, r, http.StatusBadRequest, "limit must be between 1 and 100", nil)
			return
		}
		limit = parsed
	}
	items, err := h.Content.ListPublished(r.Context(), h.DB, r.URL.Query().Get("locale"), limit)
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	visible := items[:0]
	for _, item := range items {
		if h.publicPageAllowed(item) {
			visible = append(visible, item)
		}
	}
	api.WriteList(w, summarizePages(visible), api.PaginationMeta{Limit: limit, HasMore: len(items) == limit})
}

func summarizePages(items []pages.Page) []pageSummary {
	summaries := make([]pageSummary, 0, len(items))
	for _, page := range items {
		summaries = append(summaries, pageSummary{ID: page.ID, Locale: page.Locale, Slug: page.Slug, Title: page.Title, Revision: page.Revision, PublicationGeneration: page.PublicationGeneration, UpdatedAt: page.UpdatedAt})
	}
	return summaries
}

// GET /api/v1/pages/{locale}/{slug} returns one published page.
func (h Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	h.getPublishedWithRoute(w, r, r.PathValue("locale"), r.PathValue("slug"), true)
}

// GET /api/v1/pages/by-slug?locale=&slug= is the query-string form of the
// detail endpoint for clients that cannot build path segments safely.
func (h Handler) getPublishedByQuery(w http.ResponseWriter, r *http.Request) {
	h.getPublishedWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"), false)
}

func (h Handler) getPublishedWithRoute(w http.ResponseWriter, r *http.Request, locale, slug string, recordMiss bool) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, locale, slug)
	if err != nil {
		if recordMiss && pageReadStatus(err) == http.StatusNotFound {
			h.recordNotFound(r)
		}
		writePageReadError(w, r, err)
		return
	}
	if !h.publicPageAllowed(page) {
		api.WriteError(w, r, http.StatusNotFound, "page not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, page, nil)
}

func pageReadStatus(err error) int {
	if errors.Is(err, service.ErrDependencyRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusNotFound
}

// GET /api/v1/pages/{id}/preview?token= returns the draft a preview link was
// issued for. The token binds the page, its revision and the tenant, so a
// shared link stops working as soon as the draft changes.
func (h Handler) publicPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if h.PreviewSigner == nil {
		api.WriteError(w, r, http.StatusNotFound, "preview not found", nil)
		return
	}
	claims, err := h.PreviewSigner.Verify(r.URL.Query().Get("token"), time.Now().UTC())
	if err != nil || claims.PageID != r.PathValue("id") {
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
	page, err := h.Content.Get(r.Context(), h.DB, claims.PageID)
	if err != nil || (page.Status != pages.StatusDraft && page.Status != pages.StatusPublished) || page.Revision != claims.Revision {
		api.WriteError(w, r, http.StatusNotFound, "preview not found", nil)
		return
	}
	api.WriteData(w, http.StatusOK, page, nil)
}

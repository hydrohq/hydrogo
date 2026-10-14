package transport

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type pageInput struct {
	Locale string        `json:"locale"`
	Slug   string        `json:"slug"`
	Title  string        `json:"title"`
	Body   string        `json:"body"`
	Blocks []pages.Block `json:"blocks"`
	Tags   []string      `json:"tags,omitempty"`
}

type updateInput struct {
	pageInput
	Version int `json:"version"`
}

// pageSummary is the intentionally small public discovery projection. Detail
// content remains behind the by-slug endpoint so list clients do not mistake
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

func (h Handler) listPublished(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	items, err := h.Content.ListPublished(r.Context(), h.DB, r.URL.Query().Get("locale"), limit)
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	visible := items[:0]
	for _, item := range items {
		if h.publicPageAllowed(item) {
			visible = append(visible, item)
		}
	}
	api.WriteJSON(w, http.StatusOK, summarizePages(visible))
}

func summarizePages(items []pages.Page) []pageSummary {
	summaries := make([]pageSummary, 0, len(items))
	for _, page := range items {
		summaries = append(summaries, pageSummary{ID: page.ID, Locale: page.Locale, Slug: page.Slug, Title: page.Title, Revision: page.Revision, PublicationGeneration: page.PublicationGeneration, UpdatedAt: page.UpdatedAt})
	}
	return summaries
}

func (h Handler) getPublished(w http.ResponseWriter, r *http.Request) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), r.PathValue("slug"))
	if err != nil {
		if pageStatus(err) == http.StatusNotFound {
			h.recordNotFound(r)
		}
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	api.WriteJSON(w, http.StatusOK, page)
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input pageInput
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "page service unavailable", http.StatusServiceUnavailable)
		return
	}
	page := pages.Page{ID: id, Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks, Tags: input.Tags}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Content.CreateDraft(r.Context(), page); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	h.writePage(w, r, page.ID)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	if !auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.writePage(w, r, r.PathValue("id"))
}

func (h Handler) preview(w http.ResponseWriter, r *http.Request) {
	if !auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	h.writePage(w, r, r.PathValue("id"))
}

func (h Handler) createPreviewLink(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) || h.PreviewSigner == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil || page.Status != pages.StatusDraft {
		http.Error(w, "draft unavailable", http.StatusNotFound)
		return
	}
	tenantID := ""
	if h.RequireTenantScope {
		scope, e := tenant.Require(r.Context())
		if e != nil {
			http.Error(w, "preview unavailable", http.StatusForbidden)
			return
		}
		tenantID = scope.TenantID
	}
	now := time.Now().UTC()
	token, err := h.PreviewSigner.Issue(preview.Claims{PageID: page.ID, TenantID: tenantID, Revision: page.Revision}, now, 24*time.Hour)
	if err != nil {
		http.Error(w, "preview unavailable", http.StatusServiceUnavailable)
		return
	}
	target := "/preview/pages/" + url.PathEscape(page.ID) + "?token=" + url.QueryEscape(token)
	if h.PreviewOrigin != "" {
		target = strings.TrimRight(h.PreviewOrigin, "/") + target
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = ui.DraftPreviewLinkResult(target, now.Add(24*time.Hour).Format(time.RFC3339)).Render(r.Context(), w)
		return
	}
	api.WriteJSON(w, http.StatusCreated, map[string]any{"url": target, "expires_at": now.Add(24 * time.Hour)})
}

func (h Handler) writePage(w http.ResponseWriter, r *http.Request, id string) {
	page, err := h.Content.Get(r.Context(), h.DB, id)
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	api.WriteJSON(w, http.StatusOK, page)
}

func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input updateInput
	if err := decodeJSON(w, r, &input); err != nil {
		return
	}
	page := pages.Page{ID: r.PathValue("id"), Locale: input.Locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks, Tags: input.Tags}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Content.UpdateDraft(r.Context(), page, input.Version); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	h.writePage(w, r, page.ID)
}

func (h Handler) recordPublishedSlugChange(ctx context.Context, previous pages.Page, locale, slug string) error {
	if h.RecordSlugRedirect == nil || previous.Status != pages.StatusPublished || previous.Slug == slug {
		return nil
	}
	if locale != previous.Locale {
		return nil
	}
	if h.DefaultLocale != "" && previous.Locale != h.DefaultLocale {
		return nil
	}
	return h.RecordSlugRedirect(ctx, previous.Locale, previous.Slug, slug)
}

func (h Handler) recordPreviousPublishedSlug(ctx context.Context, oldLocale, oldSlug string, page pages.Page) error {
	return h.recordPublishedSlugChange(ctx, pages.Page{Locale: oldLocale, Slug: oldSlug, Status: pages.StatusPublished}, page.Locale, page.Slug)
}

func (h Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	expectedVersion, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil || expectedVersion <= 0 {
		http.Error(w, "version query parameter is required", http.StatusBadRequest)
		return
	}
	oldLocale, oldSlug, err := h.Content.PublishedSlug(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		http.Error(w, "page publication unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := h.Content.Publish(r.Context(), r.PathValue("id"), expectedVersion); err != nil {
		http.Error(w, pageError(err), pageStatus(err))
		return
	}
	if oldSlug != "" {
		if page, getErr := h.Content.Get(r.Context(), h.DB, r.PathValue("id")); getErr == nil {
			if err := h.recordPreviousPublishedSlug(r.Context(), oldLocale, oldSlug, page); err != nil {
				http.Error(w, "page published, but its old link could not be redirected", http.StatusServiceUnavailable)
				return
			}
		}
	}
	h.writePage(w, r, r.PathValue("id"))
}

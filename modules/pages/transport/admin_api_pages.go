package transport

import (
	"context"
	"encoding/csv"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

const (
	maxAdminListFetch = 1000
	maxBulkIDs        = 500
	maxJSONBodyBytes  = 2 << 20
)

type adminListRepository interface {
	ListAdmin(context.Context, database.Executor, int) ([]pages.Page, error)
}

type pageInput struct {
	Locale         string        `json:"locale"`
	Slug           string        `json:"slug"`
	Title          string        `json:"title"`
	Body           string        `json:"body"`
	Blocks         []pages.Block `json:"blocks"`
	Tags           []string      `json:"tags,omitempty"`
	LayoutTemplate string        `json:"layout_template,omitempty"`
}

type updateInput struct {
	pageInput
	Version int `json:"version"`
}

type versionInput struct {
	Version int `json:"version"`
}

// pageListRow is the admin list projection: no body or blocks, plus the view
// count when analytics is on.
type pageListRow struct {
	ID                string       `json:"id"`
	Locale            string       `json:"locale"`
	Slug              string       `json:"slug"`
	Title             string       `json:"title"`
	Status            pages.Status `json:"status"`
	Version           int          `json:"version"`
	Revision          int          `json:"revision"`
	PublishedRevision *int         `json:"published_revision,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
	Views             *int64       `json:"views,omitempty"`
}

func (h Handler) adminRepository(w http.ResponseWriter, r *http.Request) (adminListRepository, bool) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Content.Pages) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "pages are unavailable", nil)
		return nil, false
	}
	repository, ok := h.Content.Pages.(adminListRepository)
	if !ok {
		api.WriteError(w, r, http.StatusServiceUnavailable, "page admin list is unavailable", nil)
		return nil, false
	}
	return repository, true
}

// GET /api/v1/admin/pages?limit&offset&search&status&locale&sort
func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	repository, ok := h.adminRepository(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	limit, offset, err := listPaging(query)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}
	status := strings.TrimSpace(query.Get("status"))
	if status != "" && status != string(pages.StatusDraft) && status != string(pages.StatusPublished) {
		api.WriteError(w, r, http.StatusBadRequest, "status must be draft or published", nil)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, maxAdminListFetch)
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	search := strings.ToLower(strings.TrimSpace(query.Get("search")))
	locale := canonical.Locale(query.Get("locale"))
	filtered := items[:0]
	for _, item := range items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if locale != "" && canonical.Locale(item.Locale) != locale {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(item.Title), search) && !strings.Contains(strings.ToLower(item.Slug), search) {
			continue
		}
		filtered = append(filtered, item)
	}
	var views map[string]int64
	if h.ContentViews != nil && h.AnalyticsEnabled && len(filtered) > 0 {
		ids := make([]string, 0, len(filtered))
		for _, item := range filtered {
			ids = append(ids, item.ID)
		}
		tenantID := "default"
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			tenantID = scope.TenantID
		}
		if counts, countsErr := h.ContentViews.ContentViewCounts(r.Context(), tenantID, "page", ids); countsErr == nil {
			views = counts
		}
	}
	sortPages(filtered, views, query.Get("sort"))
	rows := make([]pageListRow, 0, limit)
	for i := offset; i < len(filtered) && len(rows) < limit; i++ {
		item := filtered[i]
		row := pageListRow{ID: item.ID, Locale: item.Locale, Slug: item.Slug, Title: item.Title, Status: item.Status, Version: item.Version, Revision: item.Revision, PublishedRevision: item.PublishedRevision, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
		if views != nil {
			count := views[item.ID]
			row.Views = &count
		}
		rows = append(rows, row)
	}
	api.WriteList(w, rows, api.PaginationMeta{Limit: limit, Offset: offset, Total: len(filtered), HasMore: offset+len(rows) < len(filtered)})
}

func listPaging(query url.Values) (int, int, error) {
	limit, offset := 25, 0
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, 0, errInvalidLimit
		}
		limit = parsed
	}
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > maxAdminListFetch {
			return 0, 0, errInvalidOffset
		}
		offset = parsed
	}
	return limit, offset, nil
}

type listError string

func (e listError) Error() string { return string(e) }

const (
	errInvalidLimit  listError = "limit must be between 1 and 100"
	errInvalidOffset listError = "offset is out of range"
)

func sortPages(items []pages.Page, views map[string]int64, sortKey string) {
	less := func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) }
	switch sortKey {
	case "title":
		less = func(i, j int) bool { return items[i].Title < items[j].Title }
	case "-title":
		less = func(i, j int) bool { return items[i].Title > items[j].Title }
	case "updated_at":
		less = func(i, j int) bool { return items[i].UpdatedAt.Before(items[j].UpdatedAt) }
	case "views", "-views":
		if views == nil {
			break
		}
		less = func(i, j int) bool {
			a, b := views[items[i].ID], views[items[j].ID]
			if a == b {
				return items[i].ID < items[j].ID
			}
			if sortKey == "views" {
				return a < b
			}
			return a > b
		}
	}
	sort.SliceStable(items, less)
}

// GET /api/v1/admin/pages/{id}
func (h Handler) adminGet(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	h.writePage(w, r, r.PathValue("id"), http.StatusOK)
}

func (h Handler) writePage(w http.ResponseWriter, r *http.Request, id string, status int) {
	page, err := h.Content.Get(r.Context(), h.DB, id)
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	api.WriteData(w, status, page, nil)
}

// POST /api/v1/admin/pages
func (h Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var input pageInput
	if !api.DecodeJSON(w, r, &input, maxJSONBodyBytes) {
		return
	}
	blocks := input.Blocks
	if len(blocks) == 0 && input.LayoutTemplate != "" {
		resolved, err := pages.ResolveLayoutTemplate(input.LayoutTemplate, h.LayoutTemplates)
		if err != nil {
			api.WriteValidation(w, r, map[string]string{"layout_template": "invalid"})
			return
		}
		blocks = resolved
	}
	id, err := newID()
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "pages are unavailable", nil)
		return
	}
	locale := input.Locale
	if locale == "" {
		locale = h.defaultLocale()
	}
	page := pages.Page{ID: id, Locale: locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: blocks, Tags: input.Tags, Status: pages.StatusDraft}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		api.WriteValidation(w, r, map[string]string{"media": "unavailable"})
		return
	}
	if err := h.Content.CreateDraft(r.Context(), page); err != nil {
		writePageError(w, r, err)
		return
	}
	h.writePage(w, r, page.ID, http.StatusCreated)
}

// PUT /api/v1/admin/pages/{id}
func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var input updateInput
	if !api.DecodeJSON(w, r, &input, maxJSONBodyBytes) {
		return
	}
	if input.Version < 1 {
		api.WriteValidation(w, r, map[string]string{"version": "required"})
		return
	}
	locale := input.Locale
	if locale == "" {
		locale = h.defaultLocale()
	}
	page := pages.Page{ID: r.PathValue("id"), Locale: locale, Slug: input.Slug, Title: input.Title, Body: input.Body, Blocks: input.Blocks, Tags: input.Tags}
	if err := h.validatePageMedia(r.Context(), page); err != nil {
		api.WriteValidation(w, r, map[string]string{"media": "unavailable"})
		return
	}
	if err := h.Content.UpdateDraft(r.Context(), page, input.Version); err != nil {
		writePageError(w, r, err)
		return
	}
	h.writePage(w, r, page.ID, http.StatusOK)
}

// POST /api/v1/admin/pages/{id}/publish {version}
func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var input versionInput
	if !api.DecodeJSON(w, r, &input, maxJSONBodyBytes) {
		return
	}
	if input.Version < 1 {
		api.WriteValidation(w, r, map[string]string{"version": "required"})
		return
	}
	id := r.PathValue("id")
	oldLocale, oldSlug, err := h.Content.PublishedSlug(r.Context(), h.DB, id)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "the previously published link could not be read", nil)
		return
	}
	if err := h.Content.Publish(r.Context(), id, input.Version); err != nil {
		writePageError(w, r, err)
		return
	}
	if oldSlug != "" {
		if page, getErr := h.Content.Get(r.Context(), h.DB, id); getErr == nil {
			if err := h.recordPreviousPublishedSlug(r.Context(), oldLocale, oldSlug, page); err != nil {
				api.WriteError(w, r, http.StatusServiceUnavailable, "page published, but its old link could not be redirected", nil)
				return
			}
		}
	}
	if user := currentUser(r); user != nil && h.Notifications != nil {
		_ = h.Notifications.Add(r.Context(), h.DB, user.ID, "Page published", "Your page is now visible to visitors.", "/admin/pages/"+url.PathEscape(id))
	}
	h.writePage(w, r, id, http.StatusOK)
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

type translateInput struct {
	Locale  string `json:"locale"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Version int    `json:"version"`
}

// POST /api/v1/admin/pages/{id}/translate creates or updates the page that
// carries the same slug in another locale.
func (h Handler) adminTranslate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var input translateInput
	if !api.DecodeJSON(w, r, &input, maxJSONBodyBytes) {
		return
	}
	source, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	locale := canonical.Locale(input.Locale)
	if locale == "" || locale == source.Locale {
		api.WriteValidation(w, r, map[string]string{"locale": "invalid"})
		return
	}
	repository, ok := h.adminRepository(w, r)
	if !ok {
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, maxAdminListFetch)
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	target := pages.Page{}
	for _, item := range items {
		if item.Locale == locale && item.Slug == source.Slug {
			target, err = h.Content.Get(r.Context(), h.DB, item.ID)
			if err != nil {
				writePageReadError(w, r, err)
				return
			}
			break
		}
	}
	target.Locale, target.Slug, target.Title, target.Body = locale, source.Slug, strings.TrimSpace(input.Title), input.Body
	target.Status = pages.StatusDraft
	status := http.StatusOK
	if target.ID == "" {
		id, idErr := newID()
		if idErr != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "pages are unavailable", nil)
			return
		}
		target.ID = "page-" + id
		target.Blocks = source.Blocks
		if err := h.Content.CreateDraft(r.Context(), target); err != nil {
			writePageError(w, r, err)
			return
		}
		status = http.StatusCreated
	} else {
		if input.Version != target.Version {
			api.WriteError(w, r, http.StatusConflict, "the translation was changed by someone else; reload and try again", nil)
			return
		}
		if err := h.Content.UpdateDraft(r.Context(), target, input.Version); err != nil {
			writePageError(w, r, err)
			return
		}
	}
	h.writePage(w, r, target.ID, status)
}

type bulkInput struct {
	Action string   `json:"action"`
	IDs    []string `json:"ids"`
}

// POST /api/v1/admin/pages/bulk {action:"export", ids} streams the selected
// pages as CSV.
func (h Handler) adminBulk(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var input bulkInput
	if !api.DecodeJSON(w, r, &input, maxJSONBodyBytes) {
		return
	}
	if input.Action != "export" {
		api.WriteValidation(w, r, map[string]string{"action": "invalid"})
		return
	}
	ids, ok := normalizeBulkIDs(input.IDs)
	if !ok {
		api.WriteValidation(w, r, map[string]string{"ids": "invalid"})
		return
	}
	repository, ok := h.adminRepository(w, r)
	if !ok {
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, 5000)
	if err != nil {
		writePageReadError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pages.csv"`)
	out := csv.NewWriter(w)
	_ = out.Write([]string{"locale", "slug", "title", "status"})
	for _, item := range items {
		if _, selected := ids[item.ID]; selected {
			_ = out.Write([]string{item.Locale, item.Slug, item.Title, string(item.Status)})
		}
	}
	out.Flush()
}

func normalizeBulkIDs(raw []string) (map[string]struct{}, bool) {
	if len(raw) == 0 || len(raw) > maxBulkIDs {
		return nil, false
	}
	ids := make(map[string]struct{}, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" || len(id) > 128 {
			return nil, false
		}
		ids[id] = struct{}{}
	}
	return ids, true
}

// POST /api/v1/admin/pages/{id}/preview-link issues a signed, revision-bound
// link to GET /api/v1/pages/{id}/preview.
func (h Handler) createPreviewLink(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) || h.PreviewSigner == nil {
		writeForbidden(w, r)
		return
	}
	page, err := h.Content.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil || (page.Status != pages.StatusDraft && page.Status != pages.StatusPublished) {
		api.WriteError(w, r, http.StatusNotFound, "draft not found", nil)
		return
	}
	tenantID := ""
	if h.RequireTenantScope {
		scope, e := tenant.Require(r.Context())
		if e != nil {
			writeForbidden(w, r)
			return
		}
		tenantID = scope.TenantID
	}
	now := time.Now().UTC()
	const ttl = 24 * time.Hour
	token, err := h.PreviewSigner.Issue(preview.Claims{PageID: page.ID, TenantID: tenantID, Revision: page.Revision}, now, ttl)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "preview is unavailable", nil)
		return
	}
	target := publicBase + "/" + url.PathEscape(page.ID) + "/preview?token=" + url.QueryEscape(token)
	if h.PreviewOrigin != "" {
		target = strings.TrimRight(h.PreviewOrigin, "/") + target
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusCreated, map[string]any{"url": target, "expires_at": now.Add(ttl)}, nil)
}

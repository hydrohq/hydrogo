package transport

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

// newsListRow is the admin list projection without the body.
type newsListRow struct {
	ID          string      `json:"id"`
	Locale      string      `json:"locale"`
	Slug        string      `json:"slug"`
	Title       string      `json:"title"`
	Category    string      `json:"category,omitempty"`
	AuthorName  string      `json:"author_name,omitempty"`
	Featured    bool        `json:"featured"`
	Status      news.Status `json:"status"`
	Version     int         `json:"version"`
	Revision    int         `json:"revision"`
	ScheduledAt *time.Time  `json:"scheduled_at,omitempty"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Views       *int64      `json:"views,omitempty"`
}

// adminItem is the editor payload: the item plus the transitions the
// workflow allows from its status and its taxonomy terms.
type adminItem struct {
	news.Item
	Transitions []string          `json:"transitions"`
	Terms       []v1.TaxonomyTerm `json:"terms"`
}

func (h Handler) transitions(item news.Item) []string {
	result := make([]string, 0, 3)
	if h.Service.CanTransition(item.Status, v1.EditorialScheduled) {
		result = append(result, "schedule")
	}
	if h.Service.CanTransition(item.Status, v1.EditorialPublished) {
		result = append(result, "publish")
	}
	if h.Service.CanTransition(item.Status, v1.EditorialRetracted) {
		result = append(result, "retract")
	}
	return result
}

func validStatus(value string) bool {
	switch news.Status(value) {
	case news.StatusDraft, news.StatusScheduled, news.StatusPublished, news.StatusRetracted:
		return true
	}
	return false
}

// GET /api/v1/admin/news?limit&offset&search&status&locale&category&sort
func (h Handler) adminList(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	if !h.ready(w, r) {
		return
	}
	repository, ok := h.Service.Repository.(adminListRepository)
	if !ok {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news admin list is unavailable", nil)
		return
	}
	query := r.URL.Query()
	limit, offset, err := listPaging(query)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}
	status := strings.TrimSpace(query.Get("status"))
	if status != "" && !validStatus(status) {
		api.WriteError(w, r, http.StatusBadRequest, "status must be draft, scheduled, published or retracted", nil)
		return
	}
	items, err := repository.ListAdmin(r.Context(), h.DB, maxAdminListFetch)
	if err != nil {
		writeNewsReadError(w, r, err)
		return
	}
	search := strings.ToLower(strings.TrimSpace(query.Get("search")))
	locale := canonical.Locale(query.Get("locale"))
	category := strings.TrimSpace(query.Get("category"))
	filtered := items[:0]
	for _, item := range items {
		if status != "" && string(item.Status) != status {
			continue
		}
		if locale != "" && canonical.Locale(item.Locale) != locale {
			continue
		}
		if category != "" && item.Category != category {
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
		if counts, countsErr := h.ContentViews.ContentViewCounts(r.Context(), tenantID, "news", ids); countsErr == nil {
			views = counts
		}
	}
	sortItems(filtered, views, query.Get("sort"))
	rows := make([]newsListRow, 0, limit)
	for i := offset; i < len(filtered) && len(rows) < limit; i++ {
		item := filtered[i]
		row := newsListRow{ID: item.ID, Locale: item.Locale, Slug: item.Slug, Title: item.Title, Category: item.Category, AuthorName: item.AuthorName, Featured: item.Featured, Status: item.Status, Version: item.Version, Revision: item.Revision, ScheduledAt: item.ScheduledAt, ExpiresAt: item.ExpiresAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
		if views != nil {
			count := views[item.ID]
			row.Views = &count
		}
		rows = append(rows, row)
	}
	api.WriteList(w, rows, api.PaginationMeta{Limit: limit, Offset: offset, Total: len(filtered), HasMore: offset+len(rows) < len(filtered)})
}

type listError string

func (e listError) Error() string { return string(e) }

func listPaging(query url.Values) (int, int, error) {
	limit, offset := 25, 0
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, 0, listError("limit must be between 1 and 100")
		}
		limit = parsed
	}
	if raw := query.Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > maxAdminListFetch {
			return 0, 0, listError("offset is out of range")
		}
		offset = parsed
	}
	return limit, offset, nil
}

func sortItems(items []news.Item, views map[string]int64, sortKey string) {
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

// GET /api/v1/admin/news/{id}
func (h Handler) adminGet(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) {
		writeForbidden(w, r)
		return
	}
	h.writeAdminItem(w, r, r.PathValue("id"), http.StatusOK)
}

func (h Handler) writeAdminItem(w http.ResponseWriter, r *http.Request, id string, status int) {
	if !h.ready(w, r) {
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		writeNewsReadError(w, r, err)
		return
	}
	payload := adminItem{Item: item, Transitions: h.transitions(item), Terms: []v1.TaxonomyTerm{}}
	if taxonomy, ok := h.Service.Repository.(taxonomyAdminRepository); ok {
		terms, termsErr := taxonomy.ListTaxonomy(r.Context(), h.DB, item.ID)
		if termsErr != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "news taxonomy is unavailable", nil)
			return
		}
		if terms != nil {
			payload.Terms = terms
		}
	}
	api.WriteData(w, status, payload, nil)
}

func (h Handler) itemFromInput(id string, value input) news.Item {
	locale := value.Locale
	if locale == "" {
		locale = h.defaultLocale()
	}
	return news.Item{ID: id, Locale: locale, Slug: value.Slug, Title: value.Title, Body: value.Body, CoverMediaID: value.CoverMediaID, Excerpt: value.Excerpt, KeyTakeaways: value.KeyTakeaways, Category: value.Category, AuthorName: value.AuthorName, Featured: value.Featured, Tags: value.Tags, SEOTitle: value.SEOTitle, SEODescription: value.SEODescription, ExpiresAt: value.ExpiresAt, Status: news.StatusDraft}
}

// POST /api/v1/admin/news
func (h Handler) adminCreate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var value input
	if !api.DecodeJSON(w, r, &value, maxJSONBodyBytes) {
		return
	}
	id, err := newID()
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news is unavailable", nil)
		return
	}
	item := h.itemFromInput(id, value)
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		api.WriteValidation(w, r, map[string]string{"media": "unavailable"})
		return
	}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		writeNewsError(w, r, err)
		return
	}
	if value.TermIDs != nil {
		if err := h.replaceTerms(r, item.ID, value.TermIDs); err != nil {
			writeNewsError(w, r, err)
			return
		}
	}
	h.writeAdminItem(w, r, item.ID, http.StatusCreated)
}

// PUT /api/v1/admin/news/{id}
func (h Handler) adminUpdate(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var value updateInput
	if !api.DecodeJSON(w, r, &value, maxJSONBodyBytes) {
		return
	}
	if value.Version < 1 {
		api.WriteValidation(w, r, map[string]string{"version": "required"})
		return
	}
	item := h.itemFromInput(r.PathValue("id"), value.input)
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		api.WriteValidation(w, r, map[string]string{"media": "unavailable"})
		return
	}
	if err := h.Service.UpdateDraft(r.Context(), item, value.Version); err != nil {
		writeNewsError(w, r, err)
		return
	}
	if value.TermIDs != nil {
		if err := h.replaceTerms(r, item.ID, value.TermIDs); err != nil {
			writeNewsError(w, r, err)
			return
		}
	}
	h.writeAdminItem(w, r, item.ID, http.StatusOK)
}

type scheduleInput struct {
	Version     int       `json:"version"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// POST /api/v1/admin/news/{id}/schedule {scheduled_at, version}
func (h Handler) adminSchedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var value scheduleInput
	if !api.DecodeJSON(w, r, &value, maxJSONBodyBytes) {
		return
	}
	if value.Version < 1 {
		api.WriteValidation(w, r, map[string]string{"version": "required"})
		return
	}
	if value.ScheduledAt.IsZero() {
		api.WriteValidation(w, r, map[string]string{"scheduled_at": "required"})
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), value.Version, value.ScheduledAt.UTC()); err != nil {
		writeNewsError(w, r, err)
		return
	}
	h.writeAdminItem(w, r, r.PathValue("id"), http.StatusOK)
}

// POST /api/v1/admin/news/{id}/publish {version}
func (h Handler) adminPublish(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "publish")
}

// POST /api/v1/admin/news/{id}/retract {version}
func (h Handler) adminRetract(w http.ResponseWriter, r *http.Request) {
	h.adminTransition(w, r, "retract")
}

func (h Handler) adminTransition(w http.ResponseWriter, r *http.Request, action string) {
	if !editorAllowed(r) {
		writeForbidden(w, r)
		return
	}
	var value versionInput
	if !api.DecodeJSON(w, r, &value, maxJSONBodyBytes) {
		return
	}
	if value.Version < 1 {
		api.WriteValidation(w, r, map[string]string{"version": "required"})
		return
	}
	var err error
	if action == "publish" {
		err = h.Service.Publish(r.Context(), r.PathValue("id"), value.Version)
	} else {
		err = h.Service.Retract(r.Context(), r.PathValue("id"), value.Version)
	}
	if err != nil {
		writeNewsError(w, r, err)
		return
	}
	h.writeAdminItem(w, r, r.PathValue("id"), http.StatusOK)
}

// POST /api/v1/admin/news/{id}/preview-link issues a short-lived, revision-
// bound link to GET /api/v1/news/{id}/preview.
func (h Handler) createPreviewLink(w http.ResponseWriter, r *http.Request) {
	if !contentReadAllowed(r) || h.PreviewSigner == nil {
		writeForbidden(w, r)
		return
	}
	if !h.ready(w, r) {
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		writeNewsReadError(w, r, err)
		return
	}
	if !previewable(item.Status) {
		api.WriteError(w, r, http.StatusConflict, "this item cannot be previewed", nil)
		return
	}
	claims := preview.Claims{ResourceType: "news", ResourceID: item.ID, Revision: item.Revision}
	if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
		claims.TenantID = scope.TenantID
	} else if h.RequireTenantScope {
		writeForbidden(w, r)
		return
	}
	now := time.Now().UTC()
	const ttl = 15 * time.Minute
	token, err := h.PreviewSigner.Issue(claims, now, ttl)
	if err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "preview is unavailable", nil)
		return
	}
	target := publicBase + "/" + url.PathEscape(item.ID) + "/preview?token=" + url.QueryEscape(token)
	if h.PublicOrigin != "" {
		target = strings.TrimRight(strings.TrimSpace(h.PublicOrigin), "/") + target
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusCreated, map[string]any{"url": target, "expires_at": now.Add(ttl)}, nil)
}

func (h Handler) presenceTenant(w http.ResponseWriter, r *http.Request) (string, bool) {
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		return scope.TenantID, true
	}
	if h.RequireTenantScope {
		writeForbidden(w, r)
		return "", false
	}
	return "default", true
}

// POST /api/v1/admin/news/{id}/presence
func (h Handler) editPresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	resource := r.PathValue("id")
	if h.Presence == nil || !editorAllowed(r) || user == nil {
		writeForbidden(w, r)
		return
	}
	if len(resource) > 128 || strings.ContainsAny(resource, "\x00\r\n") {
		api.WriteError(w, r, http.StatusBadRequest, "invalid resource", nil)
		return
	}
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
	if name == "" {
		name = "Editor"
	}
	if err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "news:" + resource, UserID: user.ID, Email: user.Email, Name: name}); err != nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "presence is unavailable", nil)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, h.Presence.Active(tenantID, "news:"+resource, user.ID), nil)
}

// GET /api/v1/admin/news/{id}/presence
func (h Handler) editPresenceList(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if h.Presence == nil || !editorAllowed(r) || user == nil {
		writeForbidden(w, r)
		return
	}
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, h.Presence.Active(tenantID, "news:"+r.PathValue("id"), user.ID), nil)
}

// DELETE /api/v1/admin/news/{id}/presence
func (h Handler) editPresenceLeave(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	resource := r.PathValue("id")
	if h.Presence == nil || !editorAllowed(r) || user == nil {
		writeForbidden(w, r)
		return
	}
	tenantID, ok := h.presenceTenant(w, r)
	if !ok {
		return
	}
	h.Presence.Leave(tenantID, "news:"+resource, user.ID)
	w.WriteHeader(http.StatusNoContent)
}

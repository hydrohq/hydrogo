package transport

import (
	"context"
	"crypto/rand"
	stdsql "database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/api"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

type Handler struct {
	Service     news.Service
	DB          database.Executor
	Media       publicMediaURLProvider
	MediaLookup interface {
		Get(context.Context, database.Executor, string) (media.Asset, error)
	}
	DefaultLocale      string
	PublicOrigin       string
	RequireTenantScope bool
	Presence           *realtime.Presence
}

type publicMediaURLProvider interface {
	PublicURL(string) (string, error)
}

const (
	maxNewsTaxonomyFormBytes = 16 << 10
	maxNewsItemFormBytes     = 256 << 10
	maxNewsWorkflowFormBytes = 4 << 10
)

func (h Handler) defaultLocale() string {
	if h.DefaultLocale != "" {
		return h.DefaultLocale
	}
	return "tr"
}

type publishedListRepository interface {
	ListPublished(context.Context, database.Executor, string, int) ([]news.Item, error)
}

type relatedListRepository interface {
	ListRelatedPublished(context.Context, database.Executor, string, string, int) ([]news.Item, error)
}

type adminListRepository interface {
	ListAdmin(context.Context, database.Executor, int) ([]news.Item, error)
}

type taxonomyAdminRepository interface {
	ListTaxonomy(context.Context, database.Executor, string) ([]v1.TaxonomyTerm, error)
	ListTaxonomyTerms(context.Context, database.Executor) ([]v1.TaxonomyTerm, error)
	ReplaceTaxonomy(context.Context, database.Executor, string, []v1.TaxonomyTerm) error
	UpsertTaxonomyTerm(context.Context, database.Executor, v1.TaxonomyTerm) error
	DeleteTaxonomyTerm(context.Context, database.Executor, string) error
}

type transactionStarter interface {
	BeginTx(context.Context, *stdsql.TxOptions) (*stdsql.Tx, error)
}

type input struct {
	Locale         string     `json:"locale"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	CoverMediaID   string     `json:"cover_media_id"`
	Excerpt        string     `json:"excerpt"`
	KeyTakeaways   string     `json:"key_takeaways"`
	Category       string     `json:"category"`
	AuthorName     string     `json:"author_name"`
	Featured       bool       `json:"featured"`
	Tags           string     `json:"tags"`
	SEOTitle       string     `json:"seo_title"`
	SEODescription string     `json:"seo_description"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

type updateInput struct {
	input
	Version int `json:"version"`
}

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "POST /api/news", Protected: true, Handler: http.HandlerFunc(h.create)},
		{Pattern: "GET /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.get)},
		{Pattern: "PUT /api/news/{id}", Protected: true, Handler: http.HandlerFunc(h.update)},
		{Pattern: "POST /api/news/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.schedule)},
		{Pattern: "POST /api/news/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.publish)},
		{Pattern: "POST /api/news/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.retract)},
		{Pattern: "GET /api/news", Handler: http.HandlerFunc(h.publicList)},
		{Pattern: "GET /api/news/by-slug", Handler: http.HandlerFunc(h.publicGetByQuery)},
		{Pattern: "GET /public/news/{locale}/{slug}", Handler: http.HandlerFunc(h.publicGet)},
	}
	if h.Presence != nil {
		routes = append(routes,
			Route{Pattern: "POST /api/news/{id}/presence", Protected: true, Handler: http.HandlerFunc(h.editPresenceHeartbeat)},
			Route{Pattern: "GET /api/news/{id}/presence", Protected: true, Handler: http.HandlerFunc(h.editPresenceList)},
		)
	}
	return routes
}

func (h Handler) publicList(w http.ResponseWriter, r *http.Request) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	repository, ok := h.Service.Repository.(publishedListRepository)
	if !ok {
		http.Error(w, "news list unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(r.URL.Query().Get("locale"))
	if locale == "" {
		http.Error(w, "locale is required", http.StatusBadRequest)
		return
	}
	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			http.Error(w, "limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	items, err := repository.ListPublished(r.Context(), h.DB, locale, limit)
	if err != nil {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	api.WriteJSON(w, http.StatusOK, items)
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value input
	if !decodeJSON(w, r, &value) {
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "news service unavailable", http.StatusServiceUnavailable)
		return
	}
	item := news.Item{ID: id, Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, CoverMediaID: value.CoverMediaID, Excerpt: value.Excerpt, KeyTakeaways: value.KeyTakeaways, Category: value.Category, AuthorName: value.AuthorName, Featured: value.Featured, Tags: value.Tags, SEOTitle: value.SEOTitle, SEODescription: value.SEODescription, ExpiresAt: value.ExpiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateDraft(r.Context(), item); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	if !auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) update(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value updateInput
	if !decodeJSON(w, r, &value) {
		return
	}
	item := news.Item{ID: r.PathValue("id"), Locale: value.Locale, Slug: value.Slug, Title: value.Title, Body: value.Body, CoverMediaID: value.CoverMediaID, Excerpt: value.Excerpt, KeyTakeaways: value.KeyTakeaways, Category: value.Category, AuthorName: value.AuthorName, Featured: value.Featured, Tags: value.Tags, SEOTitle: value.SEOTitle, SEODescription: value.SEODescription, ExpiresAt: value.ExpiresAt, Status: news.StatusDraft}
	if err := h.validateNewsMedia(r.Context(), item); err != nil {
		http.Error(w, errUnavailableMediaReference.Error(), http.StatusBadRequest)
		return
	}
	if err := h.Service.UpdateDraft(r.Context(), item, value.Version); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, item.ID)
}

func (h Handler) schedule(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var value struct {
		Version     int    `json:"version"`
		ScheduledAt string `json:"scheduled_at"`
	}
	if !decodeJSON(w, r, &value) {
		return
	}
	at, err := time.Parse(time.RFC3339, value.ScheduledAt)
	if err != nil {
		http.Error(w, "invalid schedule", http.StatusBadRequest)
		return
	}
	if err := h.Service.Schedule(r.Context(), r.PathValue("id"), value.Version, at); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := h.Service.Publish(r.Context(), r.PathValue("id"), version); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) retract(w http.ResponseWriter, r *http.Request) {
	if !editorAllowed(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	if err := h.Service.Retract(r.Context(), r.PathValue("id"), version); err != nil {
		api.WriteError(w, r, newsErrorStatus(err), "news operation failed", nil)
		return
	}
	h.write(w, r, r.PathValue("id"))
}

func (h Handler) publicGet(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.PathValue("locale"), r.PathValue("slug"))
}

func (h Handler) publicGetByQuery(w http.ResponseWriter, r *http.Request) {
	h.publicGetWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"))
}

func (h Handler) publicGetWithRoute(w http.ResponseWriter, r *http.Request, rawLocale, rawSlug string) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	locale := canonical.Locale(rawLocale)
	slug, err := canonical.Path(rawSlug)
	if locale == "" || err != nil || slug == "" {
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	item, err := h.Service.GetPublishedByRoute(r.Context(), h.DB, locale, slug)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	api.WriteJSON(w, http.StatusOK, item)
}

func (h Handler) write(w http.ResponseWriter, r *http.Request, id string) {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		http.Error(w, "news unavailable", http.StatusServiceUnavailable)
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		if errors.Is(err, database.ErrDatabaseRequired) {
			http.Error(w, "news unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "news not found", http.StatusNotFound)
		return
	}
	api.WriteJSON(w, http.StatusOK, item)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionContentWrite)
}

func contentReadAllowed(r *http.Request) bool {
	return auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead)
}

func (h Handler) editPresenceHeartbeat(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	resource := r.PathValue("id")
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active || len(resource) > 128 || strings.ContainsAny(resource, "\x00\r\n") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	name := strings.TrimSpace(strings.SplitN(user.Email, "@", 2)[0])
	if name == "" {
		name = "Editor"
	}
	if err := h.Presence.Heartbeat(realtime.PresenceSession{TenantID: tenantID, Resource: "news:" + resource, UserID: user.ID, Name: name}); err != nil {
		http.Error(w, "presence unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) editPresenceList(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if h.Presence == nil || !editorAllowed(r) || user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID := "default"
	if scope, err := tenant.FromContext(r.Context()); err == nil {
		tenantID = scope.TenantID
	} else if h.RequireTenantScope {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteJSON(w, http.StatusOK, h.Presence.Active(tenantID, "news:"+r.PathValue("id"), user.ID))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return false
	}
	return true
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func newsErrorStatus(err error) int {
	if errors.Is(err, news.ErrRepositoryRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		return http.StatusServiceUnavailable
	}
	if errors.Is(err, news.ErrInvalidItem) {
		return http.StatusBadRequest
	}
	if errors.Is(err, news.ErrConflict) || errors.Is(err, news.ErrScheduleStale) {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

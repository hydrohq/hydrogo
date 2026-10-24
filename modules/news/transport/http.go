package transport

import (
	"context"
	"crypto/rand"
	stdsql "database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
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
	PreviewSigner      *preview.Signer
	ContentViews       interface {
		ContentViewCounts(context.Context, string, string, []string) (map[string]int64, error)
	}
	AnalyticsEnabled bool
}

type publicMediaURLProvider interface {
	PublicURL(string) (string, error)
}

const (
	adminBase        = "/api/v1/admin/news"
	publicBase       = "/api/v1/news"
	maxJSONBodyBytes = 1 << 20
	// maxAdminListFetch is the repository's admin list ceiling; the list
	// endpoint filters and pages within the most recent items.
	maxAdminListFetch = 100
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

type publishedCategoryFilterRepository interface {
	ListPublishedFiltered(context.Context, database.Executor, string, string, int) ([]news.Item, error)
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
	TermIDs        []string   `json:"term_ids,omitempty"`
}

type updateInput struct {
	input
	Version int `json:"version"`
}

type versionInput struct {
	Version int `json:"version"`
}

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "GET " + publicBase, Handler: http.HandlerFunc(h.publicList)},
		{Pattern: "GET " + publicBase + "/by-slug", Handler: http.HandlerFunc(h.publicGetByQuery)},
		{Pattern: "POST " + publicBase + "/views", Handler: http.HandlerFunc(h.publicRecordView)},
		{Pattern: "GET " + publicBase + "/views", Handler: http.HandlerFunc(h.publicGetViews)},
		{Pattern: "POST /api/views", Handler: http.HandlerFunc(h.publicRecordView)},
		{Pattern: "GET /api/views", Handler: http.HandlerFunc(h.publicGetViews)},
		{Pattern: "GET " + publicBase + "/{locale}/{slug}", Handler: http.HandlerFunc(h.publicGet)},

		{Pattern: "GET " + adminBase, Protected: true, Handler: http.HandlerFunc(h.adminList)},
		{Pattern: "POST " + adminBase, Protected: true, Handler: http.HandlerFunc(h.adminCreate)},
		{Pattern: "GET " + adminBase + "/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomyCatalog)},
		{Pattern: "POST " + adminBase + "/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomyTerm)},
		{Pattern: "DELETE " + adminBase + "/taxonomy/{id}", Protected: true, Handler: http.HandlerFunc(h.adminTaxonomyTermDelete)},
		{Pattern: "GET " + adminBase + "/{id}", Protected: true, Handler: http.HandlerFunc(h.adminGet)},
		{Pattern: "PUT " + adminBase + "/{id}", Protected: true, Handler: http.HandlerFunc(h.adminUpdate)},
		{Pattern: "POST " + adminBase + "/{id}/schedule", Protected: true, Handler: http.HandlerFunc(h.adminSchedule)},
		{Pattern: "POST " + adminBase + "/{id}/publish", Protected: true, Handler: http.HandlerFunc(h.adminPublish)},
		{Pattern: "POST " + adminBase + "/{id}/retract", Protected: true, Handler: http.HandlerFunc(h.adminRetract)},
		{Pattern: "GET " + adminBase + "/{id}/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminItemTaxonomy)},
		{Pattern: "PUT " + adminBase + "/{id}/taxonomy", Protected: true, Handler: http.HandlerFunc(h.adminReplaceItemTaxonomy)},
	}
	if h.Presence != nil {
		routes = append(routes,
			Route{Pattern: "POST " + adminBase + "/{id}/presence", Protected: true, Handler: http.HandlerFunc(h.editPresenceHeartbeat)},
			Route{Pattern: "GET " + adminBase + "/{id}/presence", Protected: true, Handler: http.HandlerFunc(h.editPresenceList)},
			Route{Pattern: "POST " + adminBase + "/{id}/presence/leave", Protected: true, Handler: http.HandlerFunc(h.editPresenceLeave)},
		)
	}
	if h.PreviewSigner != nil {
		routes = append(routes,
			Route{Pattern: "POST " + adminBase + "/{id}/preview-link", Protected: true, Handler: http.HandlerFunc(h.createPreviewLink)},
			Route{Pattern: "GET " + publicBase + "/{id}/preview", Handler: http.HandlerFunc(h.publicPreview)},
		)
	}
	return routes
}

func (h Handler) ready(w http.ResponseWriter, r *http.Request) bool {
	if database.RequireExecutor(h.DB) != nil || guards.IsNil(h.Service.Repository) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news is unavailable", nil)
		return false
	}
	return true
}

// writeItem answers with the stored item, which carries the version the
// client needs for its next mutation.
func (h Handler) writeItem(w http.ResponseWriter, r *http.Request, id string, status int) {
	if !h.ready(w, r) {
		return
	}
	item, err := h.Service.Repository.Get(r.Context(), h.DB, id)
	if err != nil {
		writeNewsReadError(w, r, err)
		return
	}
	api.WriteData(w, status, item, nil)
}

func writeNewsReadError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, database.ErrDatabaseRequired) || errors.Is(err, news.ErrRepositoryRequired) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "news is unavailable", nil)
		return
	}
	api.WriteError(w, r, http.StatusNotFound, "news item not found", nil)
}

// writeNewsError maps service errors onto the API envelope.
func writeNewsError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, news.ErrRepositoryRequired), errors.Is(err, database.ErrDatabaseRequired):
		api.WriteError(w, r, http.StatusServiceUnavailable, "news is unavailable", nil)
	case errors.Is(err, news.ErrInvalidItem):
		api.WriteValidation(w, r, map[string]string{"item": "invalid"})
	case errors.Is(err, news.ErrInvalidWorkflowTransition):
		api.WriteError(w, r, http.StatusConflict, "this transition is not allowed from the item's current status", nil)
	case errors.Is(err, news.ErrConflict), errors.Is(err, news.ErrScheduleStale):
		api.WriteError(w, r, http.StatusConflict, "the news item was changed by someone else; reload and try again", nil)
	case errors.Is(err, news.ErrTaxonomyTermInUse), errors.Is(err, news.ErrTaxonomyTermHasChildren):
		api.WriteError(w, r, http.StatusConflict, "the term is still referenced by items or child terms", nil)
	case errors.Is(err, news.ErrTaxonomyTermIDTaken):
		api.WriteError(w, r, http.StatusConflict, "this term id is already in use", nil)
	case errors.Is(err, stdsql.ErrNoRows):
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
	default:
		api.WriteError(w, r, http.StatusInternalServerError, "the news operation could not be completed", nil)
	}
}

func writeForbidden(w http.ResponseWriter, r *http.Request) {
	api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionContentWrite)
}

func contentReadAllowed(r *http.Request) bool {
	return auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead)
}

func currentUser(r *http.Request) *auth.User {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		return nil
	}
	return user
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

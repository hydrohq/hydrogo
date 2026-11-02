package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
	"github.com/ardaozyildirim/hydrogo/modules/seo"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Handler struct {
	DB      database.Executor
	Content service.ContentService
	Media   interface {
		Get(context.Context, database.Executor, string) (media.Asset, error)
	}
	DefaultLocale      string
	SupportedLocales   []string
	LayoutTemplates    []pages.LayoutTemplate
	AllowDemoFrontend  bool
	SEOOrigin          string
	PublicSEOSettings  func(context.Context) PublicSEOSettings
	OGRenderer         *seo.OGRenderer
	DiscoveryProvider  seo.DiscoveryProvider
	DiscoveryCache     *seo.DiscoveryCache
	PreviewSigner      *preview.Signer
	PreviewOrigin      string
	RequireTenantScope bool
	RecordSlugRedirect func(context.Context, string, string, string) error
	RecordNotFound     func(context.Context, string) error
	Notifications      *adminstudiosql.Store
	// WeeklyDigestEnabled exposes the digest preference when mail is configured.
	WeeklyDigestEnabled bool
	Presence            *realtime.Presence
	Assistant           ai.Assistant
	AIAudit             audit.Writer
	// AIDB, Dialect and AILimiter enforce the tenant AI token quota and
	// rate-limit the suggestion endpoint; without them the assistant runs
	// unbounded. AIDB is the query-capable handle quota accounting needs.
	AIDB         database.QueryExecutor
	Dialect      string
	AILimiter    security.RateLimiter
	ContentViews interface {
		ContentViewCounts(context.Context, string, string, []string) (map[string]int64, error)
	}
	AnalyticsEnabled bool
}

type PublicSEOSettings struct {
	Brand, FaviconURL, SocialImageURL string
	BusinessProfile                   *site.BusinessProfile
}

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

const (
	adminBase  = "/api/v1/admin/pages"
	publicBase = "/api/v1/pages"
)

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "GET " + publicBase, Handler: http.HandlerFunc(h.listPublished)},
		{Pattern: "GET " + publicBase + "/by-slug", Handler: http.HandlerFunc(h.getPublishedByQuery)},
		{Pattern: "GET " + publicBase + "/{locale}/{slug}", Handler: http.HandlerFunc(h.getPublished)},
		{Pattern: "GET /og/{locale}/{slug}", Handler: http.HandlerFunc(h.openGraphImage)},

		{Pattern: "GET " + adminBase, Handler: http.HandlerFunc(h.adminList), Protected: true},
		{Pattern: "POST " + adminBase, Handler: http.HandlerFunc(h.adminCreate), Protected: true},
		{Pattern: "GET " + adminBase + "/blocks", Handler: http.HandlerFunc(h.adminBlockCatalog), Protected: true},
		{Pattern: "GET " + adminBase + "/layouts", Handler: http.HandlerFunc(h.adminLayouts), Protected: true},
		{Pattern: "POST " + adminBase + "/bulk", Handler: http.HandlerFunc(h.adminBulk), Protected: true},
		{Pattern: "POST " + adminBase + "/import", Handler: http.HandlerFunc(h.adminImport), Protected: true},
		{Pattern: "GET " + adminBase + "/{id}", Handler: http.HandlerFunc(h.adminGet), Protected: true},
		{Pattern: "PUT " + adminBase + "/{id}", Handler: http.HandlerFunc(h.adminUpdate), Protected: true},
		{Pattern: "POST " + adminBase + "/{id}/publish", Handler: http.HandlerFunc(h.adminPublish), Protected: true},
		{Pattern: "POST " + adminBase + "/{id}/translate", Handler: http.HandlerFunc(h.adminTranslate), Protected: true},
		{Pattern: "GET " + adminBase + "/{id}/revisions", Handler: http.HandlerFunc(h.adminRevisions), Protected: true},
		{Pattern: "GET " + adminBase + "/{id}/revisions/{revision}", Handler: http.HandlerFunc(h.adminRevision), Protected: true},
		{Pattern: "POST " + adminBase + "/{id}/revisions/{revision}/restore", Handler: http.HandlerFunc(h.adminRestoreRevision), Protected: true},
	}
	if h.Notifications != nil {
		routes = append(routes,
			Route{Pattern: "GET /api/v1/admin/notifications", Handler: http.HandlerFunc(h.adminNotifications), Protected: true},
			Route{Pattern: "POST /api/v1/admin/notifications/read-all", Handler: http.HandlerFunc(h.adminNotificationsReadAll), Protected: true},
			Route{Pattern: "POST /api/v1/admin/notifications/{id}/dismiss", Handler: http.HandlerFunc(h.adminNotificationDismiss), Protected: true},
			Route{Pattern: "PUT /api/v1/admin/notifications/digest", Handler: http.HandlerFunc(h.adminNotificationDigest), Protected: true},
		)
	}
	if h.Presence != nil {
		routes = append(routes,
			Route{Pattern: "POST " + adminBase + "/{id}/presence", Handler: http.HandlerFunc(h.editPresenceHeartbeat), Protected: true},
			Route{Pattern: "GET " + adminBase + "/{id}/presence", Handler: http.HandlerFunc(h.editPresenceList), Protected: true},
			Route{Pattern: "POST " + adminBase + "/{id}/presence/leave", Handler: http.HandlerFunc(h.editPresenceLeave), Protected: true},
		)
	}
	if h.PreviewSigner != nil {
		routes = append(routes,
			Route{Pattern: "POST " + adminBase + "/{id}/preview-link", Handler: http.HandlerFunc(h.createPreviewLink), Protected: true},
			Route{Pattern: "GET " + publicBase + "/{id}/preview", Handler: http.HandlerFunc(h.publicPreview)},
		)
	}
	routes = append(routes,
		Route{Pattern: "GET " + adminBase + "/ai/status", Handler: http.HandlerFunc(h.adminAIStatus), Protected: true},
	)
	if h.assistantEnabled() {
		routes = append(routes,
			Route{Pattern: "POST " + adminBase + "/{id}/ai-suggest", Handler: http.HandlerFunc(h.adminAISuggest), Protected: true},
			Route{Pattern: "POST " + adminBase + "/{id}/ai-applied", Handler: http.HandlerFunc(h.adminAIApplied), Protected: true},
		)
	}
	for _, route := range seo.DiscoveryRoutes(h.DiscoveryProvider, h.DiscoveryCache) {
		routes = append(routes, Route{Pattern: route.Pattern, Handler: route.Handler})
	}
	return routes
}

func (h Handler) assistantEnabled() bool {
	return h.Assistant.Enabled && h.Assistant.Provider != nil && h.AIAudit != nil
}

func (h Handler) defaultLocale() string {
	if h.DefaultLocale != "" {
		return h.DefaultLocale
	}
	return "tr"
}

func (h Handler) publicPageAllowed(page pages.Page) bool {
	return h.AllowDemoFrontend || !strings.HasPrefix(page.ID, "demo-")
}

func (h Handler) recordNotFound(r *http.Request) {
	if h.RecordNotFound != nil && len(r.URL.Path) <= 2048 {
		_ = h.RecordNotFound(r.Context(), r.URL.Path)
	}
}

func (h Handler) publicSEOSettings(ctx context.Context) PublicSEOSettings {
	if h.PublicSEOSettings == nil {
		return PublicSEOSettings{}
	}
	return h.PublicSEOSettings(ctx)
}

func (h Handler) pageOGInput(page pages.Page, settings PublicSEOSettings) seo.OGInput {
	input := seo.OGInput{Title: page.Title, Category: "Page"}
	input.Brand = settings.Brand
	return input
}

func (h Handler) openGraphImage(w http.ResponseWriter, r *http.Request) {
	if h.OGRenderer == nil {
		http.NotFound(w, r)
		return
	}
	// Clients request /og/{locale}/{slug}.png; ServeMux wildcards must own a
	// whole segment, so the .png suffix is parsed here instead of the pattern.
	slug := strings.TrimSuffix(r.PathValue("slug"), ".png")
	page, err := h.Content.GetPublished(r.Context(), h.DB, r.PathValue("locale"), slug)
	if err != nil || !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	data, err := h.OGRenderer.Render(h.pageOGInput(page, h.publicSEOSettings(r.Context())))
	if err != nil {
		http.Error(w, "image unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data) // #nosec G705 -- PNG bytes rendered by the server and sent as image/png (nosniff is set)
}

// writePageError maps service errors onto the API envelope: 422 with field
// codes for invalid input, 409 for stale versions and route clashes, 503 when
// a dependency is missing.
func writePageError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, pages.ErrInvalidBlocks):
		api.WriteValidation(w, r, map[string]string{"blocks": "invalid"})
	case errors.Is(err, service.ErrInvalidPage):
		api.WriteValidation(w, r, map[string]string{"page": "invalid"})
	case errors.Is(err, pages.ErrRouteConflict):
		api.WriteError(w, r, http.StatusConflict, "another published page already uses this slug", map[string]string{"slug": "taken"})
	case errors.Is(err, service.ErrConflict):
		api.WriteError(w, r, http.StatusConflict, "the page was changed by someone else; reload and try again", nil)
	case errors.Is(err, service.ErrDependencyRequired), errors.Is(err, database.ErrDatabaseRequired):
		api.WriteError(w, r, http.StatusServiceUnavailable, "pages are unavailable", nil)
	default:
		api.WriteError(w, r, http.StatusInternalServerError, "the page operation could not be completed", nil)
	}
}

func writePageReadError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, service.ErrDependencyRequired) || errors.Is(err, database.ErrDatabaseRequired) {
		api.WriteError(w, r, http.StatusServiceUnavailable, "pages are unavailable", nil)
		return
	}
	api.WriteError(w, r, http.StatusNotFound, "page not found", nil)
}

func writeForbidden(w http.ResponseWriter, r *http.Request) {
	api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func editorAllowed(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return auth.Can(user, auth.PermissionContentWrite)
}

func contentReadAllowed(r *http.Request) bool {
	return auth.Can(auth.UserFromContext(r.Context()), auth.PermissionContentRead)
}

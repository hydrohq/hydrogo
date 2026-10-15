package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
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
}

type PublicSEOSettings struct {
	Brand, FaviconURL, SocialImageURL string
	BusinessProfile                   *site.BusinessProfile
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

func (h Handler) Routes() []Route {
	routes := []Route{
		{Pattern: "POST /api/pages", Handler: http.HandlerFunc(h.create), Protected: true},
		{Pattern: "GET /api/pages", Handler: http.HandlerFunc(h.listPublished)},
		{Pattern: "GET /api/pages/{id}", Handler: http.HandlerFunc(h.get), Protected: true},
		{Pattern: "GET /api/pages/{id}/preview", Handler: http.HandlerFunc(h.preview), Protected: true},
		{Pattern: "PUT /api/pages/{id}", Handler: http.HandlerFunc(h.update), Protected: true},
		{Pattern: "POST /api/pages/{id}/publish", Handler: http.HandlerFunc(h.publish), Protected: true},
		{Pattern: "GET /public/pages/{locale}/{slug}", Handler: http.HandlerFunc(h.getPublished)},
		{Pattern: "GET /api/pages/by-slug", Handler: http.HandlerFunc(h.getPublishedByQuery)},
		{Pattern: "GET /og/{locale}/{slug}", Handler: http.HandlerFunc(h.openGraphImage)},
	}
	if h.PreviewSigner != nil {
	}
	for _, route := range seo.DiscoveryRoutes(h.DiscoveryProvider, h.DiscoveryCache) {
		routes = append(routes, Route{Pattern: route.Pattern, Handler: route.Handler})
	}
	return routes
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
	_, _ = w.Write(data)
}

func (h Handler) getPublishedByQuery(w http.ResponseWriter, r *http.Request) {
	h.getPublishedWithRoute(w, r, r.URL.Query().Get("locale"), r.URL.Query().Get("slug"))
}

func (h Handler) getPublishedWithRoute(w http.ResponseWriter, r *http.Request, locale, slug string) {
	page, err := h.Content.GetPublished(r.Context(), h.DB, locale, slug)
	if err != nil {
		http.Error(w, "page lookup failed", pageReadStatus(err))
		return
	}
	if !h.publicPageAllowed(page) {
		http.NotFound(w, r)
		return
	}
	api.WriteJSON(w, http.StatusOK, page)
}

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if err := security.DecodeJSONStrict(w, r, target, 1<<20); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return err
	}
	return nil
}

func pageError(err error) string {
	if errors.Is(err, service.ErrInvalidPage) {
		return "Bilgiler eksik veya geçersiz. Alanları kontrol edip yeniden deneyin."
	}
	if errors.Is(err, service.ErrConflict) {
		return "Bu içerik başka biri tarafından güncellendi. Sayfayı yenileyip tekrar deneyin."
	}
	if errors.Is(err, pages.ErrRouteConflict) {
		return "Bu sayfa bağlantısı kullanımda. Farklı bir bağlantı adı seçin."
	}
	return "İşlem tamamlanamadı. Lütfen yeniden deneyin."
}

func pageStatus(err error) int {
	if errors.Is(err, service.ErrInvalidPage) {
		return http.StatusBadRequest
	}
	if errors.Is(err, service.ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, pages.ErrRouteConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, service.ErrDependencyRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}

func pageReadStatus(err error) int {
	if errors.Is(err, service.ErrDependencyRequired) {
		return http.StatusServiceUnavailable
	}
	return http.StatusNotFound
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

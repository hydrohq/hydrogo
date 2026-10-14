package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/canonical"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	blockrender "github.com/ardaozyildirim/hydrogo/modules/pages/render"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
	"github.com/ardaozyildirim/hydrogo/modules/seo"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

type Handler struct {
	DB      database.Executor
	Content service.ContentService
	Theme   *themev1.ThemeBundle
	Media   interface {
		Get(context.Context, database.Executor, string) (media.Asset, error)
	}
	AdminCollection     adminstudio.Collection
	LayoutTemplates     []pages.LayoutTemplate
	DefaultLocale       string
	SupportedLocales    []string
	AllowDemoFrontend   bool
	SEOOrigin           string
	PublicSEOSettings   func(context.Context) PublicSEOSettings
	NavigationProvider  func(context.Context, string) ([]site.NavigationItem, error)
	OGRenderer          *seo.OGRenderer
	DiscoveryProvider   seo.DiscoveryProvider
	DiscoveryCache      *seo.DiscoveryCache
	Notifications       *adminstudiosql.Store
	Presence            *realtime.Presence
	WeeklyDigestEnabled bool
	PreviewSigner       *preview.Signer
	PreviewOrigin       string
	RequireTenantScope  bool
	SearchEnabled       bool
	ContentViews        interface {
		ContentViewCounts(context.Context, string, string, []string) (map[string]int64, error)
	}
	Feedback interface {
		FeedbackTotals(context.Context, string, string, string) (analytics.FeedbackTotals, error)
		FeedbackCounts(context.Context, string, string, []string) (map[string]analytics.FeedbackTotals, error)
	}
	AnalyticsEnabled      bool
	RecordSlugRedirect    func(context.Context, string, string, string) error
	RecordNotFound        func(context.Context, string) error
	PublicSupport         adminstudio.SupportConfig
	PublicContactSettings func(context.Context) ([]site.Setting, error)
	Assistant             ai.Assistant
	AIAudit               audit.Writer
}

type PublicSEOSettings struct {
	Brand, FontCSS, AppearanceCSS, FaviconURL, SocialImageURL string
	BusinessProfile                                           *site.BusinessProfile
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

func (h Handler) feedbackRenderOptions(r *http.Request) blockrender.RenderOptions {
	options := blockrender.RenderOptions{CSRF: security.CSRFTokenFromContext(r.Context())}
	if !h.AnalyticsEnabled || h.Feedback == nil {
		return options
	}
	options.FeedbackEnabled = true
	options.FeedbackCounts = func(contentIDs []string) map[string][2]int64 {
		tenantID := "default"
		if scope, err := tenant.FromContext(r.Context()); err == nil {
			tenantID = scope.TenantID
		} else if h.RequireTenantScope {
			return map[string][2]int64{}
		}
		totals, err := h.Feedback.FeedbackCounts(r.Context(), tenantID, "faq", contentIDs)
		if err != nil {
			return map[string][2]int64{}
		}
		counts := make(map[string][2]int64, len(totals))
		for id, value := range totals {
			counts[id] = [2]int64{value.Up, value.Down}
		}
		return counts
	}
	return options
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
		{Pattern: "GET /public/pages/{locale}/{slug}/html", Handler: http.HandlerFunc(h.getPublishedHTML)},
		{Pattern: "GET /og/{locale}/{slug}", Handler: http.HandlerFunc(h.openGraphImage)},
		{Pattern: "GET /admin/pages/new", Handler: http.HandlerFunc(h.adminNewPage), Protected: true},
		{Pattern: "GET /admin/pages/import", Handler: http.HandlerFunc(h.adminImport), Protected: true},
		{Pattern: "POST /admin/pages/import", Handler: http.HandlerFunc(h.adminImport), Protected: true},
		{Pattern: "POST /admin/pages/bulk", Handler: http.HandlerFunc(h.adminBulk), Protected: true},
		{Pattern: "GET /admin/notifications", Handler: http.HandlerFunc(h.adminNotifications), Protected: true},
		{Pattern: "POST /admin/notifications", Handler: http.HandlerFunc(h.adminNotifications), Protected: true},
		{Pattern: "GET /admin/notifications/unread", Handler: http.HandlerFunc(h.adminUnreadNotifications), Protected: true},
		{Pattern: "GET /admin/guide", Handler: http.HandlerFunc(h.adminGuide), Protected: true},
		{Pattern: "GET /admin/pages/{id}", Handler: http.HandlerFunc(h.adminPage), Protected: true},
		{Pattern: "GET /admin/pages", Handler: http.HandlerFunc(h.adminList), Protected: true},
		{Pattern: "POST /admin/pages", Handler: http.HandlerFunc(h.adminCreateForm), Protected: true},
		{Pattern: "POST /admin/pages/{id}", Handler: http.HandlerFunc(h.adminUpdate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/publish", Handler: http.HandlerFunc(h.adminPublish), Protected: true},
		{Pattern: "POST /admin/pages/{id}/translate", Handler: http.HandlerFunc(h.adminTranslate), Protected: true},
		{Pattern: "POST /admin/pages/{id}/revisions/{revision}/restore", Handler: http.HandlerFunc(h.restoreRevision), Protected: true},
	}
	if h.Presence != nil {
		routes = append(routes,
			Route{Pattern: "POST /admin/pages/{id}/presence", Handler: http.HandlerFunc(h.editPresenceHeartbeat), Protected: true},
			Route{Pattern: "GET /admin/pages/{id}/presence", Handler: http.HandlerFunc(h.editPresenceList), Protected: true},
		)
	}
	if h.PreviewSigner != nil {
		routes = append(routes, Route{Pattern: "POST /admin/pages/{id}/preview-link", Handler: http.HandlerFunc(h.createPreviewLink), Protected: true}, Route{Pattern: "GET /preview/pages/{id}", Handler: http.HandlerFunc(h.publicDraftPreview)})
	}
	for _, route := range seo.DiscoveryRoutes(h.DiscoveryProvider, h.DiscoveryCache) {
		routes = append(routes, Route{Pattern: route.Pattern, Handler: route.Handler})
	}
	if h.assistantEnabled() {
		routes = append(routes,
			Route{Pattern: "POST /admin/pages/{id}/ai-suggest", Handler: http.HandlerFunc(h.adminAISuggest), Protected: true},
			Route{Pattern: "POST /admin/pages/{id}/ai-applied", Handler: http.HandlerFunc(h.adminAIApplied), Protected: true},
		)
	}
	return routes
}

func (h Handler) assistantEnabled() bool {
	return h.Assistant.Enabled && h.Assistant.Provider != nil && h.AIAudit != nil
}

func (h Handler) getPublishedHTML(w http.ResponseWriter, r *http.Request) {
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
	renderBlocks := make([]pages.Block, len(page.Blocks))
	for i, block := range page.Blocks {
		renderBlocks[i], err = h.withPublicMedia(r.Context(), block)
		if err != nil {
			http.Error(w, "page media lookup failed", http.StatusInternalServerError)
			return
		}
	}
	blocks, err := blockrender.BlocksWithOptions(renderBlocks, h.feedbackRenderOptions(r))
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	seoMeta, err := h.pageSEOMeta(r.Context(), page)
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	designHead := ""
	showConsentNotice := h.AnalyticsEnabled && !hasAnalyticsConsent(r)
	if h.AnalyticsEnabled {
		w.Header().Set("Vary", "Cookie")
	}
	if pageHasDesignSystemBlocks(page.Blocks) || showConsentNotice || h.PublicSupport.Enabled || h.PublicContactSettings != nil {
		var head bytes.Buffer
		if err := ui.DesignSystemHead().Render(r.Context(), &head); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		designHead = head.String()
	}
	supportMarkup := h.socialLinksMarkup(r.Context()) + h.floatingContact(r.Context(), page.Locale)
	searchMarkup := ""
	if h.SearchEnabled {
		var search bytes.Buffer
		if err := ui.PublicSearchBox().Render(r.Context(), &search); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		searchMarkup = search.String()
	}
	designHead += seoMeta
	articleHTML, tocHTML, err := ui.PrepareArticleHTML(ui.RichTextHTML(page.Body))
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	readingHead, err := renderString(r.Context(), publicReadingHead())
	if err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	designHead += readingHead
	var readingTools bytes.Buffer
	if err := ui.PublicArticleTools().Render(r.Context(), &readingTools); err != nil {
		http.Error(w, "page render failed", http.StatusInternalServerError)
		return
	}
	consentMarkup := ""
	if showConsentNotice {
		var notice bytes.Buffer
		props := designsystem.CookieNoticeProps{Action: "/analytics/consent", CSRF: security.CSRFTokenFromContext(r.Context()), ReturnTo: r.URL.Path, Variant: "bar"}
		if strings.HasPrefix(canonical.Locale(page.Locale), "en") {
			props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Privacy choices", "Allow anonymous, daily page-view totals? HydroGo does not use advertising cookies.", "Allow analytics", "Decline"
		} else {
			props.Heading, props.Message, props.AcceptLabel, props.RejectLabel = "Gizlilik tercihiniz", "Anonim günlük sayfa görüntülenme toplamlarına izin verilsin mi? HydroGo reklam çerezi kullanmaz.", "Analitiğe izin ver", "Reddet"
		}
		if err := designsystem.CookieNotice(props).Render(r.Context(), &notice); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		consentMarkup = notice.String()
	}
	for i, block := range renderBlocks {
		if block.Type != "faq" {
			continue
		}
		var payload struct {
			Items []ui.FAQItem `json:"items"`
		}
		if json.Unmarshal(block.Payload, &payload) != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		var schema bytes.Buffer
		if err := ui.FAQSchema(fmt.Sprintf("page-faq-%d", i), payload.Items).Render(r.Context(), &schema); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		designHead += schema.String()
	}
	relatedMarkup := template.HTML("")
	if related, relatedErr := h.Content.Related(r.Context(), h.DB, page, 4); relatedErr == nil {
		var relatedHTML bytes.Buffer
		if renderErr := pages.RelatedPages(page.ID, page.Locale, related).Render(r.Context(), &relatedHTML); renderErr == nil {
			relatedMarkup = template.HTML(relatedHTML.String())
		}
	}
	allowHeadStyles(w.Header(), designHead)
	if h.Theme != nil {
		article, err := renderString(r.Context(), publicPageArticle(page.Title, readingTools.String(), tocHTML, articleHTML))
		if err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		var content strings.Builder
		content.WriteString(searchMarkup)
		content.WriteString(article)
		if editorAllowed(r) {
			content.WriteString(renderInPlaceEditLink(r, page.ID))
		}
		for _, block := range renderBlocks {
			var themed strings.Builder
			if err := h.Theme.RenderBlock(&themed, block); err == nil {
				styled, styleErr := blockrender.StyleBlockHTML(block, themed.String())
				if styleErr != nil {
					http.Error(w, "page render failed", http.StatusInternalServerError)
					return
				}
				content.WriteString(styled)
			} else if errors.Is(err, themev1.ErrThemeBlockUnavailable) {
				fallback, fallbackErr := blockrender.BlocksWithOptions([]pages.Block{block}, h.feedbackRenderOptions(r))
				if fallbackErr != nil {
					http.Error(w, "page render failed", http.StatusInternalServerError)
					return
				}
				content.WriteString(fallback)
			} else {
				http.Error(w, "page render failed", http.StatusInternalServerError)
				return
			}
		}
		content.WriteString(string(relatedMarkup))
		content.WriteString(consentMarkup)
		content.WriteString(supportMarkup)
		navigation := make([]themev1.ThemeNavigationItem, 0)
		if h.NavigationProvider != nil {
			items, navErr := h.NavigationProvider(r.Context(), page.Locale)
			if navErr != nil {
				http.Error(w, "page render failed", http.StatusInternalServerError)
				return
			}
			for _, item := range items {
				navigation = append(navigation, themev1.ThemeNavigationItem{ID: item.ID, ParentID: item.ParentID, Label: item.Label, Href: item.Href, Target: item.Target, Position: item.Position})
			}
		}
		navigationHTML, navErr := themev1.RenderThemeNavigation(navigation)
		if navErr != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
			return
		}
		navigationHTML = template.HTML(string(navigationHTML) + searchMarkup)
		if err := h.Theme.RenderPage(w, themev1.PageTemplateData{Title: page.Title, Slug: page.Slug, Locale: page.Locale, Direction: canonical.Direction(page.Locale), Content: template.HTML(content.String()), Navigation: navigation, Slots: map[string]template.HTML{"content": template.HTML(content.String()), "head": template.HTML(designHead), "navigation": navigationHTML}}); err != nil {
			http.Error(w, "page render failed", http.StatusInternalServerError)
		}
		return
	}
	_ = publicPageDocument(publicPageView{
		Locale: page.Locale, Direction: canonical.Direction(page.Locale), Title: page.Title,
		Head: designHead, Search: searchMarkup, EditLink: renderInPlaceEditLink(r, page.ID), Tools: readingTools.String(),
		Article: articleHTML, Blocks: blocks, Related: string(relatedMarkup), Consent: consentMarkup, Support: supportMarkup,
	}).Render(r.Context(), w)
}

func (h Handler) recordNotFound(r *http.Request) {
	if h.RecordNotFound != nil && len(r.URL.Path) <= 2048 {
		_ = h.RecordNotFound(r.Context(), r.URL.Path)
	}
}

func (h Handler) socialLinksMarkup(ctx context.Context) string {
	if h.PublicContactSettings == nil {
		return ""
	}
	settings, err := h.PublicContactSettings(ctx)
	if err != nil {
		return ""
	}
	profiles := site.SocialProfilesFromSettings(settings)
	links := make([]designsystem.SocialLink, 0, len(profiles))
	for _, profile := range profiles {
		links = append(links, designsystem.SocialLink{Platform: profile.Platform, URL: profile.URL})
	}
	var output bytes.Buffer
	if designsystem.SocialLinks(designsystem.SocialLinksProps{Links: links}).Render(ctx, &output) != nil {
		return ""
	}
	return output.String()
}

func (h Handler) floatingContact(ctx context.Context, locale string) string {
	props := designsystem.FloatingContactProps{Phone: h.PublicSupport.Phone, WhatsAppURL: h.PublicSupport.WhatsAppURL, AriaLabel: "İletişim seçenekleri", PhoneLabel: "Ara", WhatsAppLabel: "WhatsApp", Position: "bottom-right"}
	enabled := h.PublicSupport.Enabled
	if h.PublicContactSettings != nil {
		settings, err := h.PublicContactSettings(ctx)
		if err != nil {
			return ""
		}
		if site.ContactWidgetConfigured(settings) {
			value := site.ContactWidgetFromSettings(settings)
			enabled = value.Enabled
			props.Phone, props.WhatsAppNumber, props.Message = value.Phone, value.WhatsAppNumber, value.Message
			props.Position, props.MobileOnly = value.Position, value.MobileOnly
			props.WhatsAppURL = ""
		}
	}
	if !enabled {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		props.AriaLabel, props.PhoneLabel = "Contact options", "Call"
	}
	var output bytes.Buffer
	if err := designsystem.FloatingContact(props).Render(ctx, &output); err != nil {
		return ""
	}
	return output.String()
}

func hasAnalyticsConsent(r *http.Request) bool {
	cookie, err := r.Cookie(analytics.ConsentCookie)
	return err == nil && strings.TrimSpace(cookie.Value) != ""
}

func renderInPlaceEditLink(r *http.Request, pageID string) string {
	if !editorAllowed(r) {
		return ""
	}
	var link bytes.Buffer
	if err := ui.InPlaceEditLink("/admin/pages/"+url.PathEscape(pageID)).Render(r.Context(), &link); err != nil {
		return ""
	}
	return link.String()
}

func (h Handler) pageSEOMeta(ctx context.Context, page pages.Page) (string, error) {
	settings := PublicSEOSettings{}
	if h.PublicSEOSettings != nil {
		settings = h.PublicSEOSettings(ctx)
	}
	head := seoHeadView{AppearanceCSS: settings.AppearanceCSS}
	if validLocalFontCSS(settings.FontCSS) {
		head.FontCSS = settings.FontCSS
	}
	if site.ValidAppearanceAssetURL(settings.FaviconURL) {
		head.FaviconURL = settings.FaviconURL
	}
	if settings.BusinessProfile != nil {
		if jsonLD, err := site.BusinessJSONLD(*settings.BusinessProfile); err == nil {
			head.BusinessJSONLD = json.RawMessage(jsonLD)
		}
	}
	origin := strings.TrimSpace(h.SEOOrigin)
	if origin == "" {
		return renderString(ctx, seoHead(head))
	}
	ogInput := h.pageOGInput(page, settings)
	imagePath := "/og/" + url.PathEscape(page.Locale) + "/" + url.PathEscape(page.Slug) + ".png?v=" + seo.OGVersion(ogInput)
	view, err := site.BuildSEO(site.SEOInput{Origin: origin, Locale: page.Locale, Path: "/" + page.Slug, Title: page.Title, OGImage: strings.TrimRight(origin, "/") + imagePath})
	if err != nil {
		return "", err
	}
	if translations, ok := h.Content.Pages.(interface {
		ListPublishedTranslations(context.Context, database.Executor, string, int) ([]pages.Page, error)
	}); ok {
		variants, listErr := translations.ListPublishedTranslations(ctx, h.DB, page.Slug, 100)
		if listErr != nil {
			return "", listErr
		}
		for _, variant := range variants {
			locale := canonical.Locale(variant.Locale)
			if locale == "" || locale != variant.Locale || variant.Slug != page.Slug {
				continue
			}
			href := strings.TrimRight(origin, "/") + "/public/pages/" + url.PathEscape(locale) + "/" + url.PathEscape(variant.Slug) + "/html"
			head.Alternates = append(head.Alternates, seoAlternate{Locale: locale, Href: href})
		}
	}
	head.OGImage = view.OGImage
	if site.ValidAppearanceAssetURL(settings.SocialImageURL) && settings.SocialImageURL != "" {
		head.OGImage = strings.TrimRight(origin, "/") + settings.SocialImageURL
	}
	head.Social = true
	head.Canonical, head.Description = view.CanonicalURL, view.Description
	head.OGTitle, head.OGDescription = view.OGTitle, view.OGDescription
	return renderString(ctx, seoHead(head))
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

func validLocalFontCSS(value string) bool {
	value = strings.TrimSpace(value)
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != value || !strings.HasPrefix(value, "/fonts/font-") || !strings.HasSuffix(value, ".css") || strings.ContainsAny(value, "\\\r\n\x00") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(value, "/fonts/font-"), ".css")
	if name == "" {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
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

func pageHasDesignSystemBlocks(blocks []pages.Block) bool {
	return len(blocks) > 0
}

func (h Handler) withPublicMedia(ctx context.Context, block pages.Block) (pages.Block, error) {
	if h.Media == nil || block.Type != "hero" && block.Type != "before_after" {
		return block, nil
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(block.Payload, &payload); err != nil {
		return block, err
	}
	keys := map[string]string{"media_asset_id": "media_url"}
	if block.Type == "before_after" {
		keys = map[string]string{"before_image_asset_id": "before_image_url", "after_image_asset_id": "after_image_url"}
	}
	changed := false
	for idField, urlField := range keys {
		var assetID string
		if err := json.Unmarshal(payload[idField], &assetID); err != nil || assetID == "" {
			continue
		}
		asset, err := h.Media.Get(ctx, h.DB, assetID)
		if err != nil || asset.Status != "ready" || asset.Visibility != media.VisibilityPublic {
			continue
		}
		mediaURL := "/media/" + asset.ID
		if provider, ok := h.Media.(interface{ PublicURL(string) (string, error) }); ok {
			if publicURL, urlErr := provider.PublicURL(asset.StorageKey); urlErr == nil && publicURL != "" {
				mediaURL = publicURL
			}
		}
		encodedURL, err := json.Marshal(mediaURL)
		if err != nil {
			return block, err
		}
		payload[urlField] = encodedURL
		changed = true
	}
	if !changed {
		return block, nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return block, err
	}
	block.Payload = encoded
	return block, nil
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

func (h Handler) publicDraftPreview(w http.ResponseWriter, r *http.Request) {
	if h.PreviewSigner == nil {
		http.NotFound(w, r)
		return
	}
	claims, err := h.PreviewSigner.Verify(r.URL.Query().Get("token"), time.Now().UTC())
	if err != nil || claims.PageID != r.PathValue("id") {
		http.NotFound(w, r)
		return
	}
	if h.RequireTenantScope {
		scope, e := tenant.Require(r.Context())
		if e != nil || scope.TenantID != claims.TenantID {
			http.NotFound(w, r)
			return
		}
	}
	page, err := h.Content.Get(r.Context(), h.DB, claims.PageID)
	if err != nil || (page.Status != pages.StatusDraft && page.Status != pages.StatusPublished) || page.Revision != claims.Revision {
		http.NotFound(w, r)
		return
	}
	blocks, err := blockrender.Blocks(page.Blocks)
	if err != nil {
		http.Error(w, "draft preview unavailable", http.StatusInternalServerError)
		return
	}
	article, _, err := ui.PrepareArticleHTML(ui.RichTextHTML(page.Body))
	if err != nil {
		http.Error(w, "draft preview unavailable", http.StatusInternalServerError)
		return
	}
	body, err := renderString(r.Context(), draftPreviewBody(page.Title, article, blocks))
	if err != nil {
		http.Error(w, "draft preview unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The page editor's device preview embeds this response in a same-origin
	// iframe; everything else keeps the strict frame-ancestors 'none'.
	security.AllowSameOriginFrame(w.Header())
	if h.Theme != nil {
		if err := h.Theme.RenderPage(w, themev1.PageTemplateData{Title: page.Title, Slug: page.Slug, Locale: page.Locale, Direction: canonical.Direction(page.Locale), Content: template.HTML(body), Slots: map[string]template.HTML{"content": template.HTML(body)}}); err != nil {
			return
		}
		return
	}
	_ = draftPreviewDocument(page.Locale, page.Title, body).Render(r.Context(), w)
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

package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/richtext"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	pagetransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
	"github.com/ardaozyildirim/hydrogo/modules/seo"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// initAdminStudio builds the admin studio registry (served in the admin
// schema) and the pages and news HTTP handlers.
func (a *serveApp) initAdminStudio() error {
	studioRegistry, err := adminstudio.NewRegistry([]adminstudio.Collection{
		{ID: "pages", Label: "Sayfalar", Description: "Taslak, bloklar ve yayınlanmış içerik.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}, {ID: "blocks", Label: "Bloklar", Kind: adminstudio.FieldRelation}}, Source: adminstudio.SourceModule, APIPath: "/api/v1/admin/pages"},
		{ID: "news", Label: "Haberler", Description: "Taslak, zamanlama ve yayın akışı.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "cover_media_id", Label: "Kapak görseli (herkese açık asset)", Kind: adminstudio.FieldText, Widget: "media"}, {ID: "excerpt", Label: "Kısa özet", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "key_takeaways", Label: "Önemli Noktalar (Key Takeaways)", Kind: adminstudio.FieldText, Widget: "list"}, {ID: "category", Label: "Kategori", Kind: adminstudio.FieldText}, {ID: "author_name", Label: "Yazar adı", Kind: adminstudio.FieldText}, {ID: "featured", Label: "Öne çıkan", Kind: adminstudio.FieldText, Widget: "checkbox"}, {ID: "tags", Label: "Etiketler", Kind: adminstudio.FieldText}, {ID: "seo_title", Label: "SEO başlığı", Kind: adminstudio.FieldText}, {ID: "seo_description", Label: "SEO açıklaması", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}}, Source: adminstudio.SourceModule, APIPath: "/api/v1/admin/news"},
	})
	if err != nil {
		return fmt.Errorf("admin studio registry: %w", err)
	}
	a.studioRegistry = studioRegistry
	previewSigner, err := newPreviewSigner(a.cfg.Security.PreviewSigningKey)
	if err != nil {
		return fmt.Errorf("preview signer: %w", err)
	}
	if err := a.initPagesHTTP(previewSigner); err != nil {
		return err
	}
	return a.initNewsHTTP(previewSigner)
}

// localizeStudioCollection switches the body to the rich-text editor and the
// locale field to the configured locales (hidden when i18n is off).
func (a *serveApp) localizeStudioCollection(collection *adminstudio.Collection) {
	cfg := a.cfg
	for i := range collection.Fields {
		if collection.Fields[i].ID == "body" {
			collection.Fields[i].Widget = "richtext"
		}
		if collection.Fields[i].ID == "locale" {
			if !cfg.I18N.Enabled {
				collection.Fields[i].Widget = "hidden"
				collection.Fields[i].VisibleInList = false
			} else if len(cfg.I18N.Locales) > 0 {
				collection.Fields[i].Options = cfg.I18N.Locales
			}
		}
	}
}

// newPreviewSigner uses the configured HYDRO_PREVIEW_SIGNING_KEY. Without
// one it signs with a random per-process key, so the editor's draft preview
// works out of the box; links then stop verifying after a restart or on
// another instance, which is why multi-instance sites should set the key.
func newPreviewSigner(configured string) (*preview.Signer, error) {
	if len(configured) >= 32 {
		return &preview.Signer{Key: []byte(configured)}, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return &preview.Signer{Key: key}, nil
}

func (a *serveApp) initPagesHTTP(previewSigner *preview.Signer) error {
	cfg, db, modules := a.cfg, a.db, a.modules
	requireTenantScope := a.requireTenantScope
	pageStore, siteStore, newsRouteStore := a.pageStore, a.siteStore, a.newsRouteStore
	redirectRegistry, siteService := a.redirectRegistry, &a.siteService
	if !moduleEnabled(cfg.Modules, "pages") {
		return nil
	}
	pageLLMSCache := &seo.DiscoveryCache{}
	a.pageLLMSCache = pageLLMSCache
	var recordSlugRedirect func(context.Context, string, string, string) error
	var recordNotFound func(context.Context, string) error
	if modules["site"] {
		recordSlugRedirect = func(redirectCtx context.Context, locale, oldSlug, newSlug string) error {
			tenantID := ""
			if requireTenantScope {
				scope, scopeErr := tenant.Require(redirectCtx)
				if scopeErr != nil {
					return scopeErr
				}
				tenantID = scope.TenantID
			}
			source, target := "/"+oldSlug, "/"+newSlug
			fingerprint := sha256.Sum256([]byte(tenantID + "\x00" + locale + "\x00" + source))
			rule := site.RedirectRule{ID: "slug-" + hex.EncodeToString(fingerprint[:16]), TenantID: tenantID, Pattern: source, Kind: "exact", TargetPath: target, Status: http.StatusMovedPermanently, CreatedAt: time.Now().UTC()}
			candidate := &site.RedirectRegistry{RequireTenantScope: requireTenantScope}
			if err := candidate.Replace(tenantID, redirectRegistry.List(tenantID)); err != nil {
				return err
			}
			if err := candidate.Upsert(rule); err != nil {
				return err
			}
			if err := siteService.SaveRedirectRule(redirectCtx, rule); err != nil {
				return err
			}
			return redirectRegistry.Upsert(rule)
		}
	}
	if modules["site"] {
		recordNotFound = func(missCtx context.Context, path string) error {
			return siteService.RecordNotFound(missCtx, path, time.Now().UTC())
		}
	}
	notificationStore := adminstudiosql.Store{Dialect: cfg.Database.Dialect}
	pageHTTP := pagetransport.Handler{DB: db, Content: a.contentService, Media: a.mediaService, DefaultLocale: cfg.I18N.DefaultLocale, SupportedLocales: cfg.I18N.Locales, AllowDemoFrontend: demoFrontendEnabled() || a.devMode, SEOOrigin: cfg.HTTP.PublicOrigin, PreviewSigner: previewSigner, PreviewOrigin: cfg.HTTP.PublicOrigin, RequireTenantScope: requireTenantScope, RecordSlugRedirect: recordSlugRedirect, RecordNotFound: recordNotFound, OGRenderer: &seo.OGRenderer{CacheDir: cfg.SEO.OGCacheDir}, DiscoveryCache: pageLLMSCache, Notifications: &notificationStore, WeeklyDigestEnabled: modules["forms"] && a.smtpSender != nil, Presence: a.editPresence, Assistant: a.contentAssistant, AIAudit: a.contentAIAudit, AIDB: db, Dialect: string(cfg.Database.Dialect), AILimiter: a.newRateLimiter(10, time.Minute), ContentViews: a.analyticsStore, AnalyticsEnabled: modules["analytics"], PublicSEOSettings: func(seoCtx context.Context) pagetransport.PublicSEOSettings {
		var result pagetransport.PublicSEOSettings
		if !modules["site"] {
			return result
		}
		settings, err := siteStore.ListPublicSettings(seoCtx, db)
		if err != nil {
			return result
		}
		for _, setting := range settings {
			switch setting.Key {
			case "site.name", "site.title":
				if result.Brand == "" {
					result.Brand = setting.Value
				}
			case "appearance.favicon_url":
				result.FaviconURL = setting.Value
			case "appearance.social_image_url":
				result.SocialImageURL = setting.Value
			}
		}
		if profile, ok := site.BusinessProfileFromSettings(settings); ok {
			result.BusinessProfile = &profile
		}
		return result
	}}
	pageHTTP.DiscoveryProvider = func(providerCtx context.Context) (seo.Discovery, error) {
		published, err := pageStore.ListPublishedRoutes(providerCtx, db)
		if err != nil {
			return seo.Discovery{}, err
		}
		discovery := seo.Discovery{Origin: cfg.HTTP.PublicOrigin, Name: "Website", Pages: make([]seo.DiscoveryPage, 0, len(published))}
		if modules["site"] {
			if settings, err := siteStore.ListPublicSettings(providerCtx, db); err == nil {
				for _, setting := range settings {
					switch setting.Key {
					case "site.name", "site.title":
						discovery.Name = setting.Value
					case "site.description":
						discovery.Description = setting.Value
					case "seo.llms_excluded_paths":
						discovery.ExcludePath = strings.FieldsFunc(setting.Value, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' })
					}
				}
			}
		}
		for _, page := range published {
			discovery.Pages = append(discovery.Pages, seo.DiscoveryPage{Title: page.Title, Path: "/" + page.Locale + "/" + page.Slug, Text: richtext.PlainText(page.Body)})
		}
		if modules["news"] {
			if err := newsRouteStore.StreamPublishedRoutes(providerCtx, db, func(item newsservice.Item) error {
				title := item.SEOTitle
				if strings.TrimSpace(title) == "" {
					title = item.Title
				}
				text := item.Body
				if strings.TrimSpace(text) == "" {
					text = item.Excerpt
				}
				discovery.Pages = append(discovery.Pages, seo.DiscoveryPage{Title: title, Path: "/public/news/" + item.Locale + "/" + item.Slug, Description: item.SEODescription, Text: richtext.PlainText(text)})
				return nil
			}); err != nil {
				return seo.Discovery{}, err
			}
		}
		return discovery, nil
	}
	if modules["site"] {
		a.siteHTTP.InvalidateDiscovery = pageLLMSCache.Invalidate
	}
	a.pageHTTP = pageHTTP
	return nil
}

func (a *serveApp) initNewsHTTP(previewSigner *preview.Signer) error {
	cfg, db, mediaService := a.cfg, a.db, a.mediaService
	if !moduleEnabled(cfg.Modules, "news") {
		return nil
	}
	newsHTTP := newstransport.Handler{Service: a.newsService, DB: db, Media: mediaService, MediaLookup: mediaService, DefaultLocale: cfg.I18N.DefaultLocale, PublicOrigin: cfg.HTTP.PublicOrigin, Presence: a.editPresence, PreviewSigner: previewSigner, RequireTenantScope: a.requireTenantScope, ContentViews: a.analyticsStore, AnalyticsEnabled: a.modules["analytics"]}
	a.newsHTTP = newsHTTP
	return nil
}

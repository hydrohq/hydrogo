package main

import (
	"context"
	"fmt"
	"os"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	formsruntime "github.com/ardaozyildirim/hydrogo/internal/forms"
	formstore "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/geolocation"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/storage"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	pagesstore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
	"golang.org/x/sync/errgroup"
)

// initMedia wires uploads, the optimizer, object storage and the media HTTP
// handler.
func (a *serveApp) initMedia() error {
	ctx, cfg, db, modules := a.ctx, a.cfg, a.db, a.modules
	requireTenantScope, budget := a.requireTenantScope, a.budget
	mediaStore := mediastore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	var mediaService media.Service
	var configuredWatermark *media.WatermarkOptions
	if cfg.Media.WatermarkText != "" {
		position, opacity := cfg.Media.WatermarkPosition, cfg.Media.WatermarkOpacity
		if position == "" {
			position = "bottom-right"
		}
		if opacity == 0 {
			opacity = .6
		}
		configuredWatermark = &media.WatermarkOptions{Text: cfg.Media.WatermarkText, Position: position, Opacity: opacity, Margin: cfg.Media.WatermarkMargin}
	}
	if modules["media"] {
		uploader, err := media.NewUploaderWithConcurrency(cfg.Media.Dir, budget.MediaMaxBytes, budget.WorkerConcurrency)
		if err != nil {
			return err
		}
		uploader.QuotaGuard = storage.QuotaGuard{MinFreeBytes: cfg.Media.MinFreeBytes}
		var mediaOptimizer *media.Optimizer
		if !a.opts.cgi {
			if configuredWatermark != nil {
				mediaOptimizer = media.NewOptimizerWithWatermark(cfg.Media.Dir, minInt(budget.WorkerConcurrency, 2), mediaStore, db, os.Getenv("HYDRO_MEDIA_FFMPEG"), *configuredWatermark)
			} else {
				mediaOptimizer = media.NewOptimizer(cfg.Media.Dir, minInt(budget.WorkerConcurrency, 2), mediaStore, db, os.Getenv("HYDRO_MEDIA_FFMPEG"))
			}
		}
		var blobStore media.BlobStore
		if cfg.Media.StorageDriver == "s3" {
			blobStore, err = media.NewS3BlobStore(media.S3Config{Endpoint: cfg.Media.S3Endpoint, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, UseSSL: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle, PublicBaseURL: cfg.Media.CDNBaseURL})
			if err != nil {
				return fmt.Errorf("configure media object storage: %w", err)
			}
		}
		if mediaOptimizer != nil {
			mediaOptimizer.SetBlobStore(blobStore)
		}
		mediaService = media.Service{
			Transactions:     a.transactioner,
			Uploader:         uploader,
			BlobStore:        blobStore,
			Assets:           mediaStore,
			Optimizer:        mediaOptimizer,
			AdminSettings:    mediaStore,
			DefaultWatermark: configuredWatermark,
			QuotaChecker:     mediaQuotaChecker{db: db, dialect: string(cfg.Database.Dialect)},
		}
		if mediaOptimizer != nil {
			go func() { <-ctx.Done(); mediaOptimizer.Close() }()
		}
	}
	mediaHTTP := media.HTTPHandler{Service: mediaService, DB: db, HotlinkProtection: cfg.Media.HotlinkProtection, CanonicalOrigin: cfg.HTTP.PublicOrigin, AllowedRefererHosts: cfg.Media.AllowedRefererHosts, SettingsRepository: mediaStore, SettingsCache: media.NewAdminSettingsCache(), SettingsDefaults: media.AdminSettings{HotlinkProtection: cfg.Media.HotlinkProtection, AllowedRefererHosts: cfg.Media.AllowedRefererHosts, WatermarkEnabled: configuredWatermark != nil, WatermarkText: cfg.Media.WatermarkText, WatermarkPosition: cfg.Media.WatermarkPosition, WatermarkOpacity: cfg.Media.WatermarkOpacity, WatermarkMargin: cfg.Media.WatermarkMargin}, RequireTenantScope: requireTenantScope}
	if modules["media"] {
		mediaHTTP.Garbage = &media.GarbageCollector{DB: db, Repository: mediaStore, References: mediastore.MediaReferences{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Pages: modules["pages"], News: modules["news"], Site: modules["site"], Forms: modules["forms"]}, BlobStore: mediaService.BlobStore, StorageDir: cfg.Media.Dir, Retention: 7 * 24 * time.Hour}
	}
	a.mediaService, a.mediaHTTP = mediaService, mediaHTTP
	return nil
}

// initSite wires the content stores, the form mail spool and the site module:
// settings, redirects, translation overrides, sitemaps, navigation and forms.
func (a *serveApp) initSite() error {
	ctx, cfg, db, modules, stderr := a.ctx, a.cfg, a.db, a.modules, a.stderr
	requireTenantScope, tenantScopes, transactioner := a.requireTenantScope, a.tenantScopes, a.transactioner
	analyticsStore, noCookieHasher, smtpSender := a.analyticsStore, a.noCookieHasher, a.smtpSender
	pageStore := pagesstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	siteStore := sitestore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	newsRouteStore := newsstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	a.pageStore, a.siteStore, a.newsRouteStore = pageStore, siteStore, newsRouteStore
	var spoolStore mailstore.Spool
	var mailWorker mailer.SpoolWorker
	if modules["forms"] {
		spoolStore = mailstore.Spool{DB: db, Tx: transactioner, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, MaxPending: a.budget.QueueEntries}
		if smtpSender != nil {
			mailWorker = mailer.SpoolWorker{Store: spoolStore, Sender: smtpSender, WorkerID: "mail-spool-worker-0", Lease: time.Minute, TenantScopes: tenantScopes}
			if stderr != nil {
				mailWorker.OnError = func(err error) { _, _ = fmt.Fprintf(stderr, "hydro mail spool: %v\n", err) }
			}
		}
	}
	a.spoolStore, a.mailWorker = spoolStore, mailWorker
	if !modules["site"] {
		return nil
	}
	a.siteService = site.Service{Transactions: transactioner, Store: siteStore, Audit: auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, RequireTenantScope: requireTenantScope}
	siteService := &a.siteService
	if modules["forms"] && noCookieHasher != nil {
		siteService.FormMetrics = analyticsStore
	}
	redirectRegistry := &site.RedirectRegistry{RequireTenantScope: requireTenantScope}
	a.redirectRegistry = redirectRegistry
	redirectScopes := tenantScopes
	if !requireTenantScope {
		redirectScopes = []tenant.Scope{{}}
	}
	var group errgroup.Group
	for _, scope := range redirectScopes {
		group.Go(func() error {
			redirectCtx := ctx
			if requireTenantScope {
				redirectCtx = tenant.WithScope(redirectCtx, scope)
			}
			settings, settingsErr := siteStore.ListPublicSettings(redirectCtx, db)
			if settingsErr != nil {
				return fmt.Errorf("load site translation overrides: %w", settingsErr)
			}
			translationSettings := make(map[string]string, len(settings))
			for _, setting := range settings {
				translationSettings[setting.Key] = setting.Value
			}
			i18n.ReplaceOverrides(scope.TenantID, i18n.OverrideSettings(translationSettings))
			rules, loadErr := siteService.ListRedirectRules(redirectCtx, 1000)
			if loadErr != nil {
				return fmt.Errorf("load site redirects: %w", loadErr)
			}
			if loadErr = redirectRegistry.Replace(scope.TenantID, rules); loadErr != nil {
				return fmt.Errorf("validate site redirects: %w", loadErr)
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	if modules["forms"] {
		siteService.Forms = formstore.Store{Executor: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		siteService.MailQueue = spoolStore
	}
	sitemapOrigin := cfg.HTTP.PublicOrigin
	if sitemapOrigin == "" && cfg.HTTP.Address != "" {
		sitemapOrigin = "http://" + cfg.HTTP.Address
	}
	a.linkScanOrigin = cfg.HTTP.PublicOrigin
	linkScanManager := linkcheck.NewManager(linkcheck.Options{MaxPages: 500, MaxTargets: 5000, MaxFindings: 1000, Concurrency: 4, RequestDelay: 100 * time.Millisecond, Retention: 35 * 24 * time.Hour})
	a.linkScanManager = linkScanManager
	siteHTTP := sitetransport.Handler{DB: db, Store: siteStore, Service: siteService, Redirects: redirectRegistry, RequireTenantScope: requireTenantScope, SitemapOrigin: sitemapOrigin, SitemapProvider: func(providerCtx context.Context) ([]site.SitemapEntry, error) {
		published, err := pageStore.ListPublishedRoutes(providerCtx, db)
		if err != nil {
			return nil, err
		}
		entries := make([]site.SitemapEntry, 0, len(published))
		for _, page := range published {
			seo, err := site.BuildSEO(site.SEOInput{Origin: sitemapOrigin, Locale: page.Locale, Path: page.Slug, Title: page.Title})
			if err != nil {
				return nil, err
			}
			entries = append(entries, site.SitemapEntry{URL: seo.CanonicalURL, LastModified: page.UpdatedAt})
		}
		return entries, nil
	}}
	siteHTTP.SitemapCategories = []string{}
	if moduleEnabled(cfg.Modules, "pages") {
		siteHTTP.SitemapCategories = append(siteHTTP.SitemapCategories, "pages")
	}
	if moduleEnabled(cfg.Modules, "news") {
		siteHTTP.SitemapCategories = append(siteHTTP.SitemapCategories, "articles")
	}
	siteHTTP.SitemapStream = func(providerCtx context.Context, category string, visit func(site.SitemapEntry) error) error {
		switch category {
		case "pages":
			if !modules["pages"] {
				return nil
			}
			return pageStore.StreamPublishedRoutes(providerCtx, db, func(page pages.Page) error {
				seoView, err := site.BuildSEO(site.SEOInput{Origin: sitemapOrigin, Locale: page.Locale, Path: page.Slug, Title: page.Title})
				if err != nil {
					return err
				}
				return visit(site.SitemapEntry{URL: seoView.CanonicalURL, LastModified: page.UpdatedAt, Category: "pages"})
			})
		case "articles":
			if !modules["news"] {
				return nil
			}
			return newsRouteStore.StreamPublishedRoutes(providerCtx, db, func(item newsservice.Item) error {
				seoView, err := site.BuildSEO(site.SEOInput{Origin: sitemapOrigin, Path: "/public/news/" + item.Locale + "/" + item.Slug, Title: item.Title})
				if err != nil {
					return err
				}
				return visit(site.SitemapEntry{URL: seoView.CanonicalURL, LastModified: item.UpdatedAt, Category: "articles"})
			})
		default:
			return site.ErrInvalidSEOOrigin
		}
	}
	siteHTTP.NavigationLocales = cfg.I18N.Locales
	if modules["forms"] {
		siteHTTP.FormsEnabled = true
		siteHTTP.AllowedForms = map[string]struct{}{"contact": {}}
		siteHTTP.Media = &a.mediaService
		siteHTTP.SubmissionLimiter = a.newRateLimiter(3, time.Hour)
		siteHTTP.FormViewLimiter = a.newRateLimiter(60, time.Minute)
		if noCookieHasher != nil {
			siteHTTP.FormMetrics = analyticsStore
			siteHTTP.VisitorHasher = noCookieHasher
		}
		if a.realtimeHub != nil {
			siteHTTP.Realtime = a.realtimeHub
		}
		if cfg.Forms.TurnstileSecretKey != "" {
			siteHTTP.Turnstile = formsruntime.TurnstileVerifier{Secret: cfg.Forms.TurnstileSecretKey}
		}
		proxyRanges, geoErr := geolocation.ParseTrustedProxyRanges(cfg.GeoIP.TrustedProxyCIDRs)
		if geoErr != nil {
			return fmt.Errorf("configure form geolocation: %w", geoErr)
		}
		resolver := geolocation.Resolver{TrustedProxyRanges: proxyRanges}
		if cfg.GeoIP.DatabasePath != "" {
			geoDB, openErr := geolocation.OpenMaxMind(cfg.GeoIP.DatabasePath)
			if openErr != nil {
				return fmt.Errorf("open form geolocation database: %w", openErr)
			}
			a.onClose(func() error { _ = geoDB.Close(); return nil })
			resolver.Database = geoDB
		}
		siteHTTP.GeoResolver = resolver
	}
	a.siteHTTP = siteHTTP
	return nil
}

// initNews wires the news service and its scheduled-publication worker.
func (a *serveApp) initNews() error {
	cfg, db := a.cfg, a.db
	requireTenantScope := a.requireTenantScope
	if moduleEnabled(cfg.Modules, "news") {
		a.newsService = newsservice.Service{Transactions: a.transactioner, Repository: a.newsRouteStore, Realtime: a.realtimeHub, Audit: auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, PublicCache: a.publicNewsCache, WorkflowRole: func(ctx context.Context) string {
			if user := auth.UserFromContext(ctx); user != nil {
				return string(user.Role)
			}
			return ""
		}}
		a.newsWorker = newsservice.Worker{
			Durable:      jobstore.Queue{Store: jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: a.budget.QueueEntries}, DB: db},
			Service:      a.newsService,
			Lease:        time.Minute,
			Retry:        time.Minute,
			TenantScopes: a.tenantScopes,
		}
	}
	return nil
}

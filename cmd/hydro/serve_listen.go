package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// listen builds the server configuration and serves it as a single CGI
// request, a FastCGI listener or the standalone HTTP(S) server.
func (a *serveApp) listen(routes []router.Route) error {
	serverConfig, err := a.serverConfig(routes)
	if err != nil {
		return err
	}
	if a.opts.cgi {
		handler := server.NewHandler(serverConfig)
		handler = server.StaticDiskCacheMiddleware(server.StaticCacheOptions{
			BaseDir:       a.staticCacheDir,
			SessionCookie: auth.SessionCookieName,
		})(handler)

		// 2. Comprehensive Lazy Cron runner for CGI request lifecycles
		handler = server.LazyCron(handler, a.cgiLazyCron)
		return server.ServeCGI(handler)
	}
	// NOTE: serve() has always built these mailer routes only after
	// serverConfig captured the route table, so the server never registers
	// them. Kept as-is by the serve() split; wiring them up is a separate change.
	_ = a.mailerRoutes()
	if a.opts.fastCGI {
		return server.ServeFastCGI(a.ctx, "tcp", a.opts.fastCGIAddress, serverConfig)
	}
	return server.Serve(a.ctx, serverConfig)
}

// cgiLazyCron runs scheduled work at the end of CGI requests, where no
// background workers exist.
func (a *serveApp) cgiLazyCron(cronCtx context.Context) {
	cfg, db, modules := a.cfg, a.db, a.modules
	staticCacheDir, sessionStore, webhookEndpoints := a.staticCacheDir, a.sessionStore, a.webhookEndpoints
	// Bounded cron context
	cronCtx, cancel := context.WithTimeout(cronCtx, 5*time.Second)
	defer cancel()

	// A. Scheduled news publications
	if modules["news"] && !cfg.SafeMode {
		published := false
		if err := a.newsWorker.EnqueueDue(cronCtx, db, time.Now().UTC()); err == nil {
			for i := 0; i < 4; i++ {
				worked, err := a.newsWorker.RunOnce(cronCtx, "cgi-lazy-cron", time.Now().UTC())
				if err != nil || !worked {
					break
				}
				published = true
			}
		}
		if published && staticCacheDir != "" {
			_ = server.PurgeStaticDiskCache(staticCacheDir)
		}
	}

	// B. Session cleanup
	if sessionStore != nil {
		_, _ = sessionStore.DeleteExpired(cronCtx, time.Now().UTC())
	}

	// C. Outbox webhook dispatch
	if !cfg.SafeMode && len(webhookEndpoints) > 0 {
		outboxWorker := webhooks.Worker{
			DB:           db,
			Store:        outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope},
			TenantScopes: a.tenantScopes,
			Dispatcher:   webhooks.Dispatcher{Endpoints: webhookEndpoints},
			WorkerID:     "cgi-lazy-cron",
			Lease:        10 * time.Second,
			RetryAfter:   30 * time.Second,
		}
		_, _ = outboxWorker.RunOnce(cronCtx, time.Now().UTC())
	}
	if !cfg.SafeMode && modules["forms"] && modules["pages"] && a.smtpSender != nil {
		_, _ = runAdminWeeklyDigestOnce(cronCtx, time.Now(), db, a.transactioner, adminstudiosql.Store{Dialect: cfg.Database.Dialect}, a.userStore, a.spoolStore, a.tenantScopes, a.requireTenantScope, modules["analytics"], modules["members"])
		_, _ = a.mailWorker.RunOnce(cronCtx)
	}
}

// mailerRoutes are the email test, branding, verification and DNS
// diagnostics routes. See the note in listen: they are not registered.
func (a *serveApp) mailerRoutes() []router.Route {
	cfg, db, modules, smtpSender := a.cfg, a.db, a.modules, a.smtpSender
	userStore, auditRecorder := a.userStore, a.auditRecorder
	newRateLimiter := a.newRateLimiter
	var routes []router.Route
	if smtpSender != nil && modules["auth"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/mailer/test-preview", Access: router.Protected, Handler: mailer.TestPreviewHandler{Sender: smtpSender}}, router.Route{Pattern: "POST /admin/api/mailer/test-preview", Access: router.Protected, Handler: mailer.TestPreviewHandler{Sender: smtpSender}})
		brandingHandler := mailer.EmailBrandingHandler{Settings: mailstore.EmailBrandingStore{DB: db, Dialect: cfg.Database.Dialect}, Audit: auditRecorder}
		routes = append(routes, router.Route{Pattern: "GET /admin/mailer/branding", Access: router.Protected, Handler: brandingHandler}, router.Route{Pattern: "POST /admin/mailer/branding", Access: router.Protected, Handler: brandingHandler})
		verifyNotifier := auth.EmailVerificationNotifierFunc(func(notifyCtx context.Context, email, link string) error {
			return (mailer.NotificationSender{Sender: smtpSender}).SendEmailVerification(notifyCtx, email, link)
		})
		verificationHandler := auth.EmailVerificationHandler{Service: auth.EmailVerificationService{Users: userStore, Tokens: userStore}, Transactions: a.transactioner, Notifier: verifyNotifier, PublicOrigin: cfg.HTTP.PublicOrigin, Limiter: newRateLimiter(3, time.Hour), AdminUsers: userStore, Audit: auditRecorder}
		routes = append(routes, verificationHandler.Routes()...)
	}
	if modules["auth"] && strings.TrimSpace(cfg.Mail.From) != "" {
		routes = append(routes, router.Route{Pattern: "GET /admin/mailer/dns-diagnostics", Access: router.Protected, Handler: mailer.DNSDiagnosticsHandler{From: cfg.Mail.From, Selector: cfg.Mail.DKIMSelector}})
	}
	return routes
}

// serverConfig assembles server.Config: security headers and CSP hashes,
// panic reporting, maintenance mode, error pages and readiness checks.
func (a *serveApp) serverConfig(routes []router.Route) (server.Config, error) {
	cfg, db, modules := a.cfg, a.db, a.modules
	siteStore, authHTTP, budget := a.siteStore, a.authHTTP, a.budget
	var panicReporter *observability.SentryReporter
	if cfg.Observability.SentryDSN != "" {
		var err error
		panicReporter, err = observability.NewSentryReporter(cfg.Observability.SentryDSN, nil, 64)
		if err != nil {
			return server.Config{}, fmt.Errorf("Sentry configuration: %w", err)
		}
		a.onClose(func() error { panicReporter.Close(); return nil })
	}
	adminScriptHashes, err := ui.AdminScriptHashes()
	if err != nil {
		return server.Config{}, fmt.Errorf("Admin script CSP catalog: %w", err)
	}
	adminScriptHashes = append(adminScriptHashes, media.AdminScriptHashes()...)
	if modules["news"] {
		adminScriptHashes = append(adminScriptHashes, newstransport.AdminScriptHashes()...)
	}
	return server.Config{
		Address:  cfg.HTTP.Address,
		SafeMode: cfg.SafeMode,
		CustomCodeProvider: func(r *http.Request) (string, string) {
			if !modules["site"] || cfg.SafeMode {
				return "", ""
			}
			setting, found, settingErr := siteStore.GetSetting(r.Context(), db, site.CustomCodeSettingKey)
			if settingErr != nil || !found {
				return "", ""
			}
			code := site.CustomCodeFromSetting(setting)
			return code.Head, code.Body
		},
		AutoTLS:         cfg.HTTP.AutoTLS,
		TLSHost:         cfg.HTTP.Domain,
		TLSCacheDir:     filepath.Join(filepath.Dir(cfg.Media.Dir), "certs"),
		RealtimeHub:     a.realtimeHub,
		PublicHTMLCache: a.publicHTMLCache,
		ShutdownTimeout: budget.ShutdownTimeout,
		AuthSvc:         a.authSvc,
		Metrics:         a.metrics,
		Logger:          a.logger,
		OnPanic: func(ctx context.Context, request *http.Request, panicType string, stack []byte) {
			if panicReporter != nil {
				panicReporter.CapturePanic(ctx, request, panicType, stack)
			}
		},
		OnHTTPError: func(request *http.Request, status int) {
			if panicReporter != nil {
				panicReporter.CaptureHTTPError(request, status)
			}
		},
		EnableHSTS:            cfg.Security.EnableHSTS,
		SecurityHeadersPreset: cfg.Security.HeadersPreset,
		SecurityScriptHashes:  adminScriptHashes,
		AdminSupport:          adminstudio.SupportConfig{Enabled: cfg.AdminSupport.Enabled, AgencyName: cfg.AdminSupport.AgencyName, Email: cfg.AdminSupport.Email, Phone: cfg.AdminSupport.Phone, WhatsAppURL: cfg.AdminSupport.WhatsAppURL, SLABadge: cfg.AdminSupport.SLABadge, GuideURL: cfg.AdminSupport.GuideURL, SafeModeActive: cfg.SafeMode, SiteModuleEnabled: modules["site"], CommerceModuleEnabled: modules["commerce"], CollectionsModuleEnabled: modules["collections"]},
		AdminSupportResolver: func(r *http.Request) adminstudio.SupportConfig {
			if !modules["site"] {
				return adminstudio.SupportConfig{}
			}
			setting, found, err := siteStore.GetSetting(r.Context(), db, site.StaffNoticeSettingKey)
			if err != nil || !found {
				return adminstudio.SupportConfig{}
			}
			notice := site.StaffBroadcastNoticeFromSetting(setting)
			if !notice.ActiveAt(time.Now().UTC()) {
				return adminstudio.SupportConfig{}
			}
			return adminstudio.SupportConfig{StaffNoticeActive: true, StaffNoticeMessage: notice.Message, StaffNoticeSeverity: notice.Severity, StaffNoticeDismissKey: notice.DismissKey()}
		},
		IPBlocklist:          a.ipBlocklist,
		AnomalyFilterEnabled: cfg.Security.EnableAnomalyFilter,
		AnomalyBypassPaths:   cfg.Security.AnomalyBypassPaths,
		OnSecurityAnomaly: func(r *http.Request, rule string) {
			if authHTTP.Audit != nil {
				target := rule + " from " + r.RemoteAddr
				_ = authHTTP.Audit.Record(r.Context(), "security.request_rejected", target)
			}
		},
		SecureCookie:        cfg.Security.SecureCookies,
		HTTPBodyLimit:       budget.HTTPBodyBytes,
		MaxInFlightRequests: budget.MaxInFlightHTTP,
		PublicOrigin:        cfg.HTTP.PublicOrigin,
		PublicAPIOrigins:    cfg.Security.PublicAPIOrigins,
		PublicErrorPage: func(errorCtx context.Context, status int) ([]byte, bool) {
			if !modules["site"] {
				return nil, false
			}
			settings, settingsErr := siteStore.ListPublicSettings(errorCtx, db)
			if settingsErr != nil {
				settings = nil
			}
			page := site.ErrorPageFromSettings(settings, status)
			links := make([]designsystem.PublicErrorLink, 0, len(page.Links))
			for _, link := range page.Links {
				links = append(links, designsystem.PublicErrorLink{Label: link.Label, URL: link.URL})
			}
			var body bytes.Buffer
			props := designsystem.PublicErrorPageProps{Status: status, Headline: page.Headline, Message: page.Message, ImageURL: page.ImageURL, ActionLabel: page.ActionLabel, ActionURL: page.ActionURL, Links: links}
			if err := designsystem.PublicErrorPage(props).Render(errorCtx, &body); err != nil {
				return nil, false
			}
			return body.Bytes(), true
		},
		Maintenance: server.MaintenanceConfig{
			Enabled: func(r *http.Request) bool {
				if !modules["site"] {
					return false
				}
				setting, found, err := siteStore.GetSetting(r.Context(), db, site.MaintenanceSettingKey)
				return err == nil && found && site.MaintenanceFromSetting(setting).ActiveAt(time.Now().UTC())
			},
			Allowlist: func(r *http.Request) []string {
				if !modules["site"] {
					return nil
				}
				setting, found, err := siteStore.GetSetting(r.Context(), db, site.MaintenanceSettingKey)
				if err != nil || !found {
					return nil
				}
				return site.MaintenanceFromSetting(setting).Allowlist
			},
			PreviewKey: func(r *http.Request, key string) bool {
				if !modules["site"] || len(key) < 16 || len(key) > 128 {
					return false
				}
				setting, found, err := siteStore.GetSetting(r.Context(), db, site.MaintenanceSettingKey)
				return err == nil && found && site.MaintenanceFromSetting(setting).AllowsPreviewKey(key)
			},
			SecureCookie: cfg.Security.SecureCookies,
			Render: func(r *http.Request) ([]byte, error) {
				headline, message := "We’ll be back soon", "We’re making a few improvements. Please try again shortly."
				if setting, found, err := siteStore.GetSetting(r.Context(), db, site.MaintenanceSettingKey); err == nil && found {
					settings := site.MaintenanceFromSetting(setting)
					if settings.Headline != "" {
						headline = settings.Headline
					}
					if settings.Message != "" {
						message = settings.Message
					}
				}
				var body bytes.Buffer
				err := designsystem.MaintenancePage(headline, message).Render(r.Context(), &body)
				return body.Bytes(), err
			},
		},
		TenantResolver: a.tenantResolver,
		Routes:         routes,
		Ready: func(checkCtx context.Context) error {
			if err := db.PingContext(checkCtx); err != nil {
				return err
			}
			return database.CheckCompatibilityForModules(checkCtx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
		},
	}, nil
}

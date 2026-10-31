package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

// listen builds the server configuration and serves it as a single CGI
// request, a FastCGI listener or the standalone HTTP(S) server.
func (a *serveApp) listen(routes []router.Route) error {
	serverConfig, err := a.serverConfig(append(routes, a.mailerRoutes()...))
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
		if os.Getenv("REQUEST_METHOD") != "" {
			return server.ServeCGI(handler)
		}
		// Under Apache mod_fcgid or FastCGI wrapper, REQUEST_METHOD is not present
		// in process environment but supplied over FastCGI protocol on standard input.
		return server.ServeFastCGIStdin(a.ctx, handler, serverConfig.ShutdownTimeout)
	}
	if a.opts.fastCGI {
		if a.opts.fastCGIAddress != "" {
			return server.ServeFastCGI(a.ctx, "tcp", a.opts.fastCGIAddress, serverConfig)
		}
		handler := server.NewHandler(serverConfig)
		handler = server.StaticDiskCacheMiddleware(server.StaticCacheOptions{
			BaseDir:       a.staticCacheDir,
			SessionCookie: auth.SessionCookieName,
		})(handler)
		handler = server.LazyCron(handler, a.cgiLazyCron)
		return server.ServeFastCGIStdin(a.ctx, handler, serverConfig.ShutdownTimeout)
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
	if a.memberSessions != nil {
		_, _ = a.memberSessions.DeleteExpired(cronCtx, time.Now().UTC())
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

// mailerRoutes are the e-mail verification routes (admin-only; the handler
// checks the admin role).
func (a *serveApp) mailerRoutes() []router.Route {
	cfg, modules, smtpSender := a.cfg, a.modules, a.smtpSender
	userStore, auditRecorder := a.userStore, a.auditRecorder
	newRateLimiter := a.newRateLimiter
	var routes []router.Route
	if smtpSender != nil && modules["auth"] {
		verifyNotifier := auth.EmailVerificationNotifierFunc(func(notifyCtx context.Context, email, link string) error {
			return (mailer.NotificationSender{Sender: smtpSender}).SendEmailVerification(notifyCtx, email, link)
		})
		verificationHandler := auth.EmailVerificationHandler{Service: auth.EmailVerificationService{Users: userStore, Tokens: userStore}, Transactions: a.transactioner, Notifier: verifyNotifier, PublicOrigin: cfg.HTTP.PublicOrigin, Limiter: newRateLimiter(3, time.Hour), AdminUsers: userStore, Audit: auditRecorder}
		routes = append(routes, verificationHandler.Routes()...)
	}
	return routes
}

// serverConfig assembles server.Config: security headers, panic reporting,
// maintenance mode and readiness checks.
// readiness is the check behind /readyz and the admin health endpoint.
func (a *serveApp) readiness(checkCtx context.Context) error {
	if err := a.db.PingContext(checkCtx); err != nil {
		return err
	}
	return database.CheckCompatibilityForModules(checkCtx, a.db, a.cfg.Database.Dialect, migrations.FS, enabledModuleSet(a.cfg.Modules))
}

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
	clientIP, err := requestmeta.NewIPExtractor(cfg.HTTP.BehindProxy, cfg.HTTP.TrustedProxies)
	if err != nil {
		return server.Config{}, err
	}
	return server.Config{
		ClientIP:        clientIP,
		Address:         cfg.HTTP.Address,
		SafeMode:        cfg.SafeMode,
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
		AdminSupport:          adminstudio.SupportConfig{Enabled: cfg.AdminSupport.Enabled, AgencyName: cfg.AdminSupport.AgencyName, Email: cfg.AdminSupport.Email, Phone: cfg.AdminSupport.Phone, WhatsAppURL: cfg.AdminSupport.WhatsAppURL, SLABadge: cfg.AdminSupport.SLABadge, GuideURL: cfg.AdminSupport.GuideURL, SafeModeActive: cfg.SafeMode, SiteModuleEnabled: modules["site"], CommerceModuleEnabled: modules["commerce"], CollectionsModuleEnabled: modules["collections"], Modules: modules},
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
		CSRFExemptPaths:      append(commerceCSRFExemptPaths(modules, cfg.Commerce), "/api/v1/public/onboard"),
		OnSecurityAnomaly: func(r *http.Request, rule string) {
			if authHTTP.Audit != nil {
				target := rule + " from " + requestmeta.ClientIP(r)
				details, _ := json.Marshal(map[string]any{
					"ip":         requestmeta.ClientIP(r),
					"user_agent": r.UserAgent(),
					"method":     r.Method,
					"path":       r.URL.Path,
					"query":      r.URL.RawQuery,
					"rule":       rule,
					"status":     http.StatusForbidden,
				})
				if detailed, ok := authHTTP.Audit.(auth.DetailedAuditRecorder); ok {
					_ = detailed.RecordDetails(r.Context(), "security.request_rejected", target, nil, details)
				} else {
					_ = authHTTP.Audit.Record(r.Context(), "security.request_rejected", target)
				}
			}
		},
		AdminRequestAudit:   adminRequestAudit(cfg.Audit.LogRequestBodies, authHTTP.Audit, a.logger),
		SecureCookie:        cfg.Security.SecureCookies,
		HTTPBodyLimit:       budget.HTTPBodyBytes,
		MaxInFlightRequests: budget.MaxInFlightHTTP,
		PublicOrigin:        cfg.HTTP.PublicOrigin,
		PublicAPIOrigins:    cfg.Security.PublicAPIOrigins,
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
		},
		TenantResolver: a.tenantResolver,
		TenantGovernor: a.tenantGovernor,
		Routes:         routes,
		Ready:          a.readiness,
	}, nil
}

// adminRequestAudit returns the request-body audit middleware when the
// operator enabled audit.log_request_bodies, and nil otherwise.
func adminRequestAudit(enabled bool, recorder auth.AuditRecorder, logger *slog.Logger) func(http.Handler) http.Handler {
	detailed, ok := recorder.(auth.DetailedAuditRecorder)
	if !enabled || !ok {
		return nil
	}
	return audit.RequestLog{
		Recorder: detailed,
		// Uploads are multipart and skipped by content type; these endpoints are
		// high-volume reports from the admin UI itself, not user actions.
		SkipPath: func(path string) bool { return path == "/api/v1/admin/client-errors" },
		// A dropped audit write is itself worth knowing about: the chain's
		// "tamper-evident" guarantee depends on writes actually landing.
		OnWriteError: func(err error) {
			if logger != nil {
				logger.Error("admin request audit write failed", "error", err)
			}
		},
	}.Middleware
}

package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	aicore "github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	analyticsstore "github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	pushservice "github.com/ardaozyildirim/hydrogo/internal/push"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	securitystore "github.com/ardaozyildirim/hydrogo/internal/security/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	membertransport "github.com/ardaozyildirim/hydrogo/modules/members/transport"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	pagesstore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	pageservice "github.com/ardaozyildirim/hydrogo/modules/pages/service"
	pagetransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
	"github.com/ardaozyildirim/hydrogo/modules/seo"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
)

// serveApp carries the dependencies serve() wires together: configuration,
// the database, stores, services and module handlers. The init* methods fill
// it in a fixed order (see serve), the route methods read it, and close
// releases resources in reverse order of acquisition.
type serveApp struct {
	ctx     context.Context
	stop    context.CancelFunc
	stderr  io.Writer
	opts    serveOptions
	cfg     config.Config
	devMode bool
	closers []func() error
	// workers tracks every background goroutine startWorkers launches, so
	// Serve can wait for them to actually exit after the HTTP server stops
	// instead of only cancelling their context and returning immediately --
	// a scheduled backup's VACUUM INTO mid-flight would otherwise be cut off
	// with a partial file left behind.
	workers sync.WaitGroup

	tenantResolver     *tenant.HostResolver
	tenantScopes       []tenant.Scope
	requireTenantScope bool
	modules            map[string]bool

	// Database and security core.
	budget        capacity.Budget
	db            *sql.DB
	transactioner database.Transactor
	ipBlocklist   *security.IPBlocklist
	banStore      securitystore.BanStore
	redisClient   redisRuntime

	// Accounts.
	userStore       *authstore.UserStore
	userAdmin       auth.UserAdminRepository
	sessionStore    *authstore.SessionStore
	authSvc         *auth.AuthService
	mfaService      *auth.MFAService
	recoveryService *auth.PasswordRecovery
	auditRecorder   auth.AuditRecorder
	authHTTP        auth.HTTPHandler
	mobileAuthHTTP  auth.MobileHTTPHandler
	memberStore     memberstore.Store
	memberSessions  expiredSessionCleaner
	memberService   members.Service
	memberHTTP      membertransport.Handler
	memberOAuthHTTP *membertransport.OAuthHandler
	smtpSender      mailer.Sender

	// Optional modules.
	contentAssistant aicore.Assistant
	contentAIAudit   auditstore.Store
	pushDispatcher   *pushservice.Dispatcher
	realtimeHub      *realtime.Hub
	editPresence     *realtime.Presence
	analyticsStore   analyticsstore.Store
	contentViewQueue *analytics.ContentViewQueue
	searchQueryQueue *analytics.SearchQueryQueue
	noCookieQueue    *analytics.NoCookieQueue
	noCookieHasher   *analytics.VisitorHasher

	// Content.
	publicPageCache  pageCache
	publicHTMLCache  *server.PublicHTMLCache
	staticCacheDir   string
	contentService   pageservice.ContentService
	mediaService     media.Service
	mediaHTTP        media.HTTPHandler
	pageStore        pagesstore.Store
	siteStore        sitestore.Store
	newsRouteStore   newsstore.Store
	spoolStore       mailstore.Spool
	mailWorker       mailer.SpoolWorker
	siteService      site.Service
	siteHTTP         sitetransport.Handler
	linkScanManager  *linkcheck.Manager
	linkScanOrigin   string
	redirectRegistry *site.RedirectRegistry
	newsService      newsservice.Service
	newsWorker       newsservice.Worker
	studioRegistry   adminstudio.Registry
	// contentTypes are the loaded YAML content types and contentLocales the
	// site locales they are served in; both feed the admin schema.
	collectionService collections.Service
	contentTypes      []contenttypes.Definition
	contentLocales    contenttypes.Locales
	pageHTTP          pagetransport.Handler
	pageLLMSCache     *seo.DiscoveryCache
	newsHTTP          newstransport.Handler

	// Runtime.
	metrics          *observability.Metrics
	hookOnce         sync.Once
	hookStatus       *hookStatus
	hookCall         contenttypes.HookCaller
	webhookEndpoints []webhooks.Endpoint
	logger           *slog.Logger
}

// onClose registers a cleanup step; close runs them last-in, first-out, like
// the defers they replace.
func (a *serveApp) onClose(fn func() error) {
	a.closers = append(a.closers, fn)
}

func (a *serveApp) close() error {
	var err error
	for i := len(a.closers) - 1; i >= 0; i-- {
		if closeErr := a.closers[i](); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}
	a.closers = nil
	return err
}

// newRateLimiter returns a Redis-backed limiter when Redis is configured and
// an in-memory one otherwise, tenant-scoped when tenancy is on.
func (a *serveApp) newRateLimiter(limit int, window time.Duration) security.RateLimiter {
	return newRuntimeRateLimiter(a.redisClient, a.cfg.Redis.KeyPrefix, limit, window, a.requireTenantScope)
}

// activeStudioCollections lists the admin studio collections of enabled modules.
func (a *serveApp) activeStudioCollections() []adminstudio.Collection {
	return adminstudio.FilterActiveCollections(a.studioRegistry.Collections(), a.modules)
}

// spawn runs fn in a goroutine tracked by a.workers, so Serve can wait for
// every background worker to actually exit after the HTTP server stops.
func (a *serveApp) spawn(fn func()) {
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		fn()
	}()
}

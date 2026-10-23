package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// initMetrics creates the runtime metrics and their cache, job and database
// stats providers.
func (a *serveApp) initMetrics() {
	cfg, db := a.cfg, a.db
	metrics := observability.NewMetrics()
	a.metrics = metrics
	metrics.SetWorkerConcurrency(a.budget.WorkerConcurrency)
	a.newsWorker.OnRetry = metrics.RecordWorkerRetry
	a.newsWorker.OnDeadLetter = metrics.RecordWorkerDeadLetter
	contentService := a.contentService
	metrics.SetCacheStatsProvider(func() observability.CacheStats {
		if contentService.PublicCache == nil {
			return observability.CacheStats{}
		}
		stats := contentService.PublicCache.Stats()
		return observability.CacheStats{Hits: stats.Hits, Misses: stats.Misses, Loads: stats.Loads, LoadErrors: stats.LoadErrors, Evictions: stats.Evictions, Expirations: stats.Expirations}
	})
	jobActive := "0"
	jobDead := "1"
	if cfg.Database.Dialect == database.Postgres {
		jobActive = "FALSE"
		jobDead = "TRUE"
	}
	metrics.SetJobStatsProvider(func() observability.JobStats {
		stats := observability.JobStats{}
		statsCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := db.QueryRowContext(statsCtx, "SELECT COUNT(*) FROM jobs WHERE dead_lettered = "+jobActive).Scan(&stats.Depth); err != nil {
			return observability.JobStats{}
		}
		if err := db.QueryRowContext(statsCtx, "SELECT COUNT(*) FROM jobs WHERE lease_until IS NOT NULL").Scan(&stats.Leased); err != nil {
			return observability.JobStats{Depth: stats.Depth}
		}
		if err := db.QueryRowContext(statsCtx, "SELECT COUNT(*) FROM jobs WHERE dead_lettered = "+jobDead).Scan(&stats.DeadLettered); err != nil {
			return observability.JobStats{Depth: stats.Depth, Leased: stats.Leased}
		}
		return stats
	})
	metrics.SetDatabaseStatsProvider(func() observability.DatabaseStats {
		stats := db.Stats()
		return observability.DatabaseStats{MaxOpenConnections: stats.MaxOpenConnections, OpenConnections: stats.OpenConnections, InUseConnections: stats.InUse, IdleConnections: stats.Idle, WaitCount: stats.WaitCount, WaitDuration: stats.WaitDuration}
	})
}

type expiredSessionCleaner interface {
	DeleteExpired(context.Context, time.Time) (int64, error)
}

type sessionCleanupObserver interface {
	RecordSessionCleanup(int64)
	RecordSessionCleanupError()
}

// newServeLogger logs JSON, or the readable development format in dev mode.
func newServeLogger(stderr io.Writer) *slog.Logger {
	logWriter := stderr
	if logWriter == nil {
		logWriter = io.Discard
	}
	if os.Getenv("HYDRO_DEV_MODE") == "1" {
		return slog.New(observability.DevelopmentHandler(logWriter))
	}
	return slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// startWorkers starts the long-running background workers of a standalone or
// FastCGI process; CGI requests run their jobs through the lazy cron instead.
func (a *serveApp) startWorkers() {
	ctx, cfg, db, modules, stderr := a.ctx, a.cfg, a.db, a.modules, a.stderr
	requireTenantScope, tenantScopes, logger := a.requireTenantScope, a.tenantScopes, a.logger
	cgiMode, metrics := a.opts.cgi, a.metrics
	if !cgiMode {
		go runSessionCleanupObserved(ctx, a.sessionStore, stderr, metrics)
	}
	if modules["media"] && !cgiMode {
		go runStaleMediaTempSweeper(ctx, cfg.Media.Dir, stderr)
	}
	if modules["forms"] && a.smtpSender != nil && !cgiMode && !cfg.SafeMode {
		if modules["pages"] {
			a.mailWorker.OnSchedule = func(jobCtx context.Context, now time.Time) error {
				_, err := runAdminWeeklyDigestOnce(jobCtx, now, db, a.transactioner, adminstudiosql.Store{Dialect: cfg.Database.Dialect}, a.userStore, a.spoolStore, tenantScopes, requireTenantScope, modules["analytics"], modules["members"])
				return err
			}
		}
		go a.mailWorker.Run(ctx, 500*time.Millisecond)
	}
	if modules["news"] && !cgiMode && !cfg.SafeMode {
		go runNewsWorker(ctx, db, a.newsWorker, metrics, stderr, a.budget.WorkerConcurrency)
	}
	if modules["commerce"] && !cgiMode && !cfg.SafeMode {
		go runCommerceReservationSweeper(ctx, commerce.Worker{Service: a.commerceService()}, db, stderr)
	}
	if !cfg.SafeMode && !cgiMode && (modules["pages"] || modules["news"] || modules["commerce"] || (modules["collections"] && subscribesToContent(cfg.Webhooks))) {
		go runOutboxWorker(ctx, webhooks.Worker{DB: db, Store: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, TenantScopes: tenantScopes, Dispatcher: webhooks.Dispatcher{Endpoints: a.webhookEndpoints}, WorkerID: "outbox-worker-0", Lease: time.Minute, RetryAfter: time.Minute}, stderr)
	}
	if !cfg.SafeMode && !cgiMode && modules["site"] && a.linkScanManager != nil && a.linkScanOrigin != "" {
		tenantKeys := []string{"default"}
		if requireTenantScope {
			tenantKeys = make([]string, 0, len(tenantScopes))
			for _, scope := range tenantScopes {
				tenantKeys = append(tenantKeys, scope.TenantID)
			}
		}
		if len(tenantKeys) > 0 {
			go linkcheck.RunMonthly(ctx, a.linkScanManager, tenantKeys, a.linkScanOrigin, func(err error) { logger.WarnContext(ctx, "scheduled link audit could not start", "error", err) })
		}
	}
	if !cfg.SafeMode && !cgiMode {
		go runHookSchedules(ctx, cfg, a.hookCaller(), time.Now, logger)
	}
	if !cgiMode && (cfg.Database.Dialect == database.SQLite || cfg.Database.Dialect == database.Postgres) {
		go database.RunWeeklyMaintenance(ctx, a.runDatabaseMaintenance)
	}
}

// runDatabaseMaintenance runs one weekly maintenance pass on its own
// connection and audits the result when the pages module is enabled.
func (a *serveApp) runDatabaseMaintenance(maintenanceCtx context.Context) {
	cfg, modules, logger := a.cfg, a.modules, a.logger
	maintenanceCtx, cancel := context.WithTimeout(maintenanceCtx, 30*time.Minute)
	defer cancel()
	maintenanceDB, openErr := database.OpenWithOptions(maintenanceCtx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(a.budget, cfg.Profile))
	if openErr != nil {
		logger.ErrorContext(maintenanceCtx, "database maintenance connection failed", "error", openErr)
		return
	}
	defer maintenanceDB.Close()
	path := ""
	if cfg.Database.Dialect == database.SQLite {
		path, openErr = sqliteDatabasePath(cfg.Database.DSN)
		if openErr != nil {
			logger.ErrorContext(maintenanceCtx, "database maintenance path unavailable", "error", openErr)
			return
		}
	}
	report, maintenanceErr := database.Maintain(maintenanceCtx, maintenanceDB, cfg.Database.Dialect, path)
	if maintenanceErr != nil {
		logger.ErrorContext(maintenanceCtx, "database maintenance failed", "dialect", cfg.Database.Dialect, "error", maintenanceErr)
		return
	}
	logger.InfoContext(maintenanceCtx, "database maintenance completed", "dialect", report.Dialect, "reclaimed_bytes", report.ReclaimedBytes, "duration", report.Duration, "wal_checkpoint_deferred", report.WALCheckpointDeferred)
	if modules["pages"] {
		target := fmt.Sprintf("dialect=%s reclaimed_bytes=%d duration=%s wal_checkpoint_deferred=%t", report.Dialect, report.ReclaimedBytes, report.Duration, report.WALCheckpointDeferred)
		if a.requireTenantScope {
			for _, scope := range a.tenantScopes {
				if auditErr := a.auditRecorder.Record(tenant.WithScope(maintenanceCtx, scope), "database.maintenance", target); auditErr != nil {
					logger.ErrorContext(maintenanceCtx, "database maintenance audit failed", "tenant", scope.TenantID, "error", auditErr)
				}
			}
		} else if auditErr := a.auditRecorder.Record(maintenanceCtx, "database.maintenance", target); auditErr != nil {
			logger.ErrorContext(maintenanceCtx, "database maintenance audit failed", "error", auditErr)
		}
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
)

// serve runs the CMS: it parses flags and configuration, wires the runtime
// (serveApp), assembles the route table, starts background workers and then
// serves HTTP, FastCGI or a single CGI request. Resources are released in
// reverse order of acquisition when it returns.
func serve(args []string, stderr io.Writer) (serveErr error) {
	if stderr == nil {
		stderr = io.Discard
	}
	opts, cfg, err := parseServeArgs(args, stderr)
	if err != nil {
		return err
	}
	tenantResolver, tenantScopes, err := configureTenancy(cfg)
	if err != nil {
		return err
	}
	devMode, restoreDevMode, err := enableDevMode(cfg.HTTP.Address, opts.dev, stderr)
	if err != nil {
		return err
	}
	defer restoreDevMode()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopPprof, err := startDebugPprof()
	if err != nil {
		return err
	}
	defer stopPprof()
	tracingProfile, err := observability.LoadTracingProfile()
	if err != nil {
		return fmt.Errorf("tracing profile: %w", err)
	}
	shutdownTracing, err := observability.InstallOTLPProvider(ctx, tracingProfile)
	if err != nil {
		return fmt.Errorf("install otlp tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

	app := &serveApp{
		ctx:                ctx,
		stop:               stop,
		stderr:             stderr,
		opts:               opts,
		cfg:                cfg,
		devMode:            devMode,
		tenantResolver:     tenantResolver,
		tenantScopes:       tenantScopes,
		requireTenantScope: cfg.Tenancy.Enabled,
		modules:            enabledModuleSet(cfg.Modules),
	}
	defer func() {
		if closeErr := app.close(); closeErr != nil {
			serveErr = errors.Join(serveErr, closeErr)
		}
	}()
	for _, step := range []func() error{
		app.openDatabase,
		app.initAuth,
		app.initOptionalModules,
		app.initMail,
		app.initPublicCaches,
		app.startScheduledBackup,
		app.initMedia,
		app.initSite,
		app.initNews,
		app.initAdminStudio,
	} {
		if err := step(); err != nil {
			return err
		}
	}
	auditstore.SetChain(cfg.Audit.Chain)
	routes, err := app.routes()
	if err != nil {
		return err
	}
	app.initMetrics()
	app.logger = newServeLogger(stderr)
	if !cfg.Audit.Chain {
		if heads, err := (auditstore.Store{Dialect: cfg.Database.Dialect}).ChainHeads(context.Background(), app.db); err == nil && len(heads) > 0 {
			app.logger.Warn("audit.chain is off but a chain exists: new events are not linked; turn it on or run 'hydro audit relink' after the gap")
		}
	}
	app.startWorkers()
	return app.listen(routes)
}

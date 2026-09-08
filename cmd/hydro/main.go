package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/migrations"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	pagesstore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	pageservice "github.com/ardaozyildirim/hydrogo/modules/pages/service"
	pagetransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
)

const version = "0.0.0-dev"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hydro:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("a command is required")
	}

	switch args[0] {
	case "version":
		if len(args) != 1 {
			return errors.New("version does not accept arguments")
		}
		_, err := fmt.Fprintln(stdout, version)
		return err
	case "serve":
		return serve(args[1:], stderr)
	case "migrate":
		return migrate(args[1:], stderr)
	case "create-admin":
		return createAdmin(args[1:], stdin, stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.DefaultArgon2Params)
	mfaService := &auth.MFAService{Store: userStore, MasterKey: []byte(cfg.Security.MFAMasterKey)}
	recoveryService := &auth.PasswordRecovery{Users: userStore, Tokens: userStore, Sessions: sessionStore, ArgonParams: auth.DefaultArgon2Params}
	invitationService := &auth.InvitationService{Users: userStore, Invites: userStore, ArgonParams: auth.DefaultArgon2Params}
	authHTTP := auth.HTTPHandler{Service: authSvc, Audit: newAuditRecorder(db, cfg.Database.Dialect), MFA: mfaService, MFADB: db, Transactions: database.NewTransactor(db), LoginLimiter: security.NewAttemptLimiter(10000, 5, time.Minute), MFALimiter: security.NewAttemptLimiter(10000, 5, time.Minute), Recovery: recoveryService, RecoveryDB: db, Invitation: invitationService, InviteDB: db}
	contentService := pageservice.ContentService{
		Transactions: database.NewTransactor(db),
		Pages:        pagesstore.Store{Dialect: cfg.Database.Dialect},
		Audit:        auditstore.Store{Dialect: cfg.Database.Dialect},
		Outbox:       outboxstore.Store{Dialect: cfg.Database.Dialect},
		PublicCache:  cache.New[pages.Page](256, 30*time.Second),
		Sanitize:     ui.NewHTMLSanitizer().Sanitize,
	}
	pageHTTP := pagetransport.Handler{DB: db, Content: contentService}
	uploader, err := media.NewUploader(cfg.Media.Dir, 10<<20)
	if err != nil {
		return err
	}
	mediaService := media.Service{Transactions: database.NewTransactor(db), Uploader: uploader, Assets: mediastore.Store{Dialect: cfg.Database.Dialect}}
	mediaHTTP := media.HTTPHandler{Service: mediaService, DB: db}
	pageStore := pagesstore.Store{Dialect: cfg.Database.Dialect}
	siteStore := sitestore.Store{Dialect: cfg.Database.Dialect}
	siteService := site.Service{Transactions: database.NewTransactor(db), Store: siteStore, Audit: auditstore.Store{Dialect: cfg.Database.Dialect}}
	siteHTTP := sitetransport.Handler{DB: db, Store: siteStore, Service: &siteService, SubmissionLimiter: security.NewAttemptLimiter(10000, 5, time.Minute), AllowedForms: map[string]struct{}{"contact": {}}, SitemapOrigin: cfg.HTTP.PublicOrigin, SitemapProvider: func(providerCtx context.Context) ([]site.SitemapEntry, error) {
		published, err := pageStore.ListPublishedRoutes(providerCtx, db)
		if err != nil {
			return nil, err
		}
		entries := make([]site.SitemapEntry, 0, len(published))
		for _, page := range published {
			seo, err := site.BuildSEO(site.SEOInput{Origin: cfg.HTTP.PublicOrigin, Locale: page.Locale, Path: page.Slug, Title: page.Title})
			if err != nil {
				return nil, err
			}
			entries = append(entries, site.SitemapEntry{URL: seo.CanonicalURL, LastModified: page.UpdatedAt})
		}
		return entries, nil
	}}
	newsStore := newsstore.Store{Dialect: cfg.Database.Dialect}
	newsService := newsservice.Service{Transactions: database.NewTransactor(db), Repository: newsStore, Audit: auditstore.Store{Dialect: cfg.Database.Dialect}, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect}}
	newsHTTP := newstransport.Handler{Service: newsService, DB: db}
	newsWorker := newsservice.Worker{
		Durable: jobstore.Queue{Store: jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: 10000}, DB: db},
		Service: newsService,
		Lease:   time.Minute,
		Retry:   time.Minute,
	}
	routes := append(authHTTP.RouterRoutes(), pageRouterRoutes(pageHTTP.Routes())...)
	routes = append(routes, mediaRouterRoutes(mediaHTTP.Routes())...)
	routes = append(routes, siteRouterRoutes(siteHTTP.Routes())...)
	routes = append(routes, newsRouterRoutes(newsHTTP.Routes())...)
	go runNewsWorker(ctx, db, newsWorker, stderr)

	return server.Serve(ctx, server.Config{
		Address: cfg.HTTP.Address,
		AuthSvc: authSvc,
		Routes:  routes,
		Ready: func(checkCtx context.Context) error {
			if err := db.PingContext(checkCtx); err != nil {
				return err
			}
			return database.CheckCompatibility(checkCtx, db, cfg.Database.Dialect, migrations.FS)
		},
	})
}

func runNewsWorker(ctx context.Context, db database.Executor, worker newsservice.Worker, stderr io.Writer) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		now := time.Now().UTC()
		if err := worker.EnqueueDue(ctx, db, now); err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "hydro news worker enqueue: %v\n", err)
		}
		for i := 0; i < 100; i++ {
			processed, err := worker.RunOnce(ctx, "news-worker", now)
			if err != nil {
				if ctx.Err() == nil {
					_, _ = fmt.Fprintf(stderr, "hydro news worker run: %v\n", err)
				}
				break
			}
			if !processed {
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func pageRouterRoutes(routes []pagetransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func mediaRouterRoutes(routes []media.HTTPRoute) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func siteRouterRoutes(routes []sitetransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if route.Public {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func newsRouterRoutes(routes []newstransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Protected
		if !route.Protected {
			access = router.Public
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}

func newAuditRecorder(db *sql.DB, dialect database.Dialect) auth.AuditRecorder {
	store := auditstore.Store{Dialect: dialect}
	return auth.AuditRecorderFunc(func(ctx context.Context, action, targetID string) error {
		return database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
			return store.Record(ctx, tx, newAuditID(), action, targetID)
		})
	})
}

func newAuditID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("audit-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func migrate(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	return database.ApplyMigrations(ctx, db, cfg.Database.Dialect, migrations.FS)
}

func createAdmin(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("create-admin", flag.ContinueOnError)
	flags.SetOutput(stderr)
	email := flags.String("email", "", "Admin email address")
	passwordStdin := flags.Bool("password-stdin", false, "Read admin password from stdin")
	values := addRuntimeFlags(flags)

	if err := flags.Parse(args); err != nil {
		return err
	}
	if *email == "" || !*passwordStdin {
		return errors.New("--email and --password-stdin are required")
	}
	password, err := readPassword(stdin)
	if err != nil {
		return fmt.Errorf("failed to read admin password: %w", err)
	}
	if password == "" {
		return errors.New("admin password cannot be empty")
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.DefaultArgon2Params)

	u, err := authSvc.CreateUser(ctx, *email, password, auth.RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	fmt.Fprintf(stdout, "Created admin user: %s (id: %s)\n", u.Email, u.ID)
	return nil
}

func readPassword(stdin io.Reader) (string, error) {
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

type runtimeFlags struct {
	configPath      *string
	address         *string
	databaseDialect *string
	databaseDSN     *string
}

func addRuntimeFlags(flags *flag.FlagSet) runtimeFlags {
	return runtimeFlags{
		configPath:      flags.String("config", "", "YAML config file"),
		address:         flags.String("address", "", "HTTP listen address"),
		databaseDialect: flags.String("database-dialect", "", "sqlite or postgres"),
		databaseDSN:     flags.String("database-dsn", "", "database connection string"),
	}
}

func loadRuntimeConfig(flags *flag.FlagSet, values runtimeFlags) (config.Config, error) {
	set := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) { set[item.Name] = true })
	overrides := config.Overrides{}
	if set["address"] {
		overrides.HTTPAddress = values.address
	}
	if set["database-dialect"] {
		overrides.DatabaseDialect = values.databaseDialect
	}
	if set["database-dsn"] {
		overrides.DatabaseDSN = values.databaseDSN
	}
	return config.Load(*values.configPath, os.LookupEnv, overrides)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  hydro version
  hydro migrate [runtime flags]
  hydro create-admin --email EMAIL --password-stdin [runtime flags]
  hydro serve [runtime flags]
  hydro help

Runtime flags:
  --config PATH
  --address HOST:PORT
  --database-dialect sqlite|postgres
  --database-dsn DSN

`)
}

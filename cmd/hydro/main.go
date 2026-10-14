package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	aicore "github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	analyticsstore "github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	pushservice "github.com/ardaozyildirim/hydrogo/internal/push"
	pushstore "github.com/ardaozyildirim/hydrogo/internal/push/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	securitystore "github.com/ardaozyildirim/hydrogo/internal/security/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/storage"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/trash"
	trashstore "github.com/ardaozyildirim/hydrogo/internal/trash/sqlstore"
	trashtransport "github.com/ardaozyildirim/hydrogo/internal/trash/transport"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	membertransport "github.com/ardaozyildirim/hydrogo/modules/members/transport"
	newsservice "github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
	newstransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	pagesstore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	pageservice "github.com/ardaozyildirim/hydrogo/modules/pages/service"
	pagetransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
	"github.com/ardaozyildirim/hydrogo/modules/seo"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

type pageCache interface {
	GetOrLoad(string, func() (pages.Page, error)) (pages.Page, error)
	GetOrLoadAtGeneration(string, uint64, func() (pages.Page, error)) (pages.Page, error)
	Clear()
	Stats() cache.Stats
}

func clearPublicCaches(page pageCache, html *server.PublicHTMLCache) {
	if page != nil {
		page.Clear()
	}
	if html != nil {
		html.Clear()
	}
}

const version = "0.0.0-dev"

var (
	buildCommit = "unknown"
	buildTime   = "unknown"
)

type versionInfo struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	BuildTime       string `json:"build_time"`
	GoVersion       string `json:"go_version"`
	SchemaMigration string `json:"schema_migration"`
}

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
	case "icons":
		return runIconPack(args[1:], stdout, stderr)
	case "healthcheck":
		if len(args) != 1 {
			return errors.New("healthcheck takes no arguments")
		}
		address := os.Getenv("HYDRO_HEALTHCHECK_URL")
		if address == "" {
			address = "http://127.0.0.1:8080/healthz"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("healthcheck returned %s", resp.Status)
		}
		return nil
	case "version":
		if len(args) == 1 {
			_, err := fmt.Fprintln(stdout, version)
			return err
		}
		if len(args) == 2 && args[1] == "--json" {
			info, err := currentVersionInfo()
			if err != nil {
				return err
			}
			data, err := json.Marshal(info)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, string(data))
			return err
		}
		return errors.New("version accepts only --json")
	case "serve":
		return serve(args[1:], stderr)
	case "run", "dev":
		return devCommand(args[1:], stdout, stderr)
	case "migrate":
		return migrate(args[1:], stderr)
	case "rollback":
		return rollbackCommand(args[1:], stdout, stderr)
	case "ban":
		return ipBanCommand(args[1:], false, stdout, stderr)
	case "unban":
		return ipBanCommand(args[1:], true, stdout, stderr)
	case "migration":
		return migrationCommand(args[1:], stdout, stderr)
	case "upgrade":
		return upgrade(args[1:], stdout, stderr)
	case "create-admin":
		return createAdmin(args[1:], stdin, stdout, stderr)
	case "seed-demo":
		return seedDemo(args[1:], stdout, stderr)
	case "seed":
		return seedDemo(args[1:], stdout, stderr)
	case "clean":
		return cleanDemoCommand(args[1:], stdout, stderr)
	case "sdk":
		return sdkCommand(args[1:], stdout, stderr)
	case "schema":
		return schemaCommand(args[1:], stdout, stderr)
	case "db":
		return dbCommand(args[1:], stdout, stderr)
	case "doctor":
		return doctor(args[1:], stdout, stderr)
	case "validate":
		return validateCommand(args[1:], stdout, stderr)
	case "test":
		return testCommand(args[1:], stdout, stderr)
	case "backup":
		return backupCommand(args[1:], stdout, stderr)
	case "restore":
		return restoreCommand(args[1:], stdout, stderr)
	case "portable":
		return portableCommand(args[1:], stdout, stderr)
	case "release":
		return releaseCommand(args[1:], stdout, stderr)
	case "runtime":
		return runtimeCommand(args[1:], stdout, stderr)
	case "jobs":
		return jobsCommand(args[1:], stdout, stderr)
	case "tenant":
		return tenantCommand(args[1:], stdin, stdout, stderr)
	case "profile":
		return profileCommand(args[1:], stdout, stderr)
	case "events":
		return eventsCommand(args[1:], stdout)
	case "fonts":
		return fontsCommand(args[1:], stdout)
	case "scan-links":
		return scanLinksCommand(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type repeatedFlag []string

func (f *repeatedFlag) String() string { return strings.Join(*f, ",") }
func (f *repeatedFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("repeated value cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted || strings.HasPrefix(value, wanted+"=") {
			return true
		}
	}
	return false
}

var migrationIDPattern = regexp.MustCompile(`^[0-9]{12}_[a-z][a-z0-9_]{1,62}$`)

func currentVersionInfo() (versionInfo, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return versionInfo{}, fmt.Errorf("read embedded migrations: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return versionInfo{}, errors.New("no embedded migrations found")
	}
	return versionInfo{Version: version, Commit: buildCommit, BuildTime: buildTime, GoVersion: runtime.Version(), SchemaMigration: ids[len(ids)-1]}, nil
}

func contentViewFilterFromRequest(r *http.Request, limit int) analyticsstore.ContentViewFilter {
	filter := analyticsstore.ContentViewFilter{Limit: limit}
	if r == nil {
		return filter
	}
	q := r.URL.Query()
	filter.TenantID, filter.ContentType, filter.ContentID, filter.Locale = strings.TrimSpace(q.Get("tenant_id")), strings.TrimSpace(q.Get("content_type")), strings.TrimSpace(q.Get("content_id")), strings.TrimSpace(q.Get("locale"))
	switch q.Get("range") {
	case "7d", "30d":
		days := 7
		if q.Get("range") == "30d" {
			days = 30
		}
		filter.FromDay = time.Now().UTC().AddDate(0, 0, -days+1).Format("2006-01-02")
		filter.ToDay = time.Now().UTC().Format("2006-01-02")
	case "custom":
		if from, to, ok := customAnalyticsDateRange(q.Get("from"), q.Get("to")); ok {
			filter.FromDay, filter.ToDay = from, to
		}
	}
	return filter
}

func customAnalyticsDateRange(from, to string) (string, string, bool) {
	start, startErr := time.Parse("2006-01-02", from)
	end, endErr := time.Parse("2006-01-02", to)
	today, _ := time.Parse("2006-01-02", time.Now().UTC().Format("2006-01-02"))
	if startErr != nil || endErr != nil || end.Before(start) || end.After(today) || end.Sub(start) > 365*24*time.Hour {
		return "", "", false
	}
	return start.Format("2006-01-02"), end.Format("2006-01-02"), true
}

func dailyNoCookieAnalytics(ctx context.Context, store analyticsstore.Store, tenantID, day string) ([]ui.NoCookieAnalyticsRow, int64, error) {
	end, err := time.Parse("2006-01-02", day)
	if err != nil {
		return nil, 0, err
	}
	start := end.AddDate(0, 0, -29).Format("2006-01-02")
	rows, err := store.ListDailyStatsBetween(ctx, start, day, tenantID, 5000)
	if err != nil {
		return nil, 0, err
	}
	aggregates := make(map[string]int64)
	dailyViews := make(map[string]int64)
	var unique int64
	todayRows, err := store.ListDailyStats(ctx, day, tenantID, 500)
	if err != nil {
		return nil, 0, err
	}
	for _, row := range todayRows {
		if row.Metric == "unique_visitors" {
			unique = row.EventCount
			break
		}
	}
	for _, row := range rows {
		if row.Metric == "page" {
			aggregates[row.Metric+"\x00"+row.Dimension] += row.EventCount
			dailyViews[row.Day] += row.EventCount
		} else if row.Metric != "unique_visitors" {
			aggregates[row.Metric+"\x00"+row.Dimension] += row.EventCount
		}
	}
	result := make([]ui.NoCookieAnalyticsRow, 0, len(aggregates)+len(dailyViews))
	for date, count := range dailyViews {
		result = append(result, ui.NoCookieAnalyticsRow{Day: date, Metric: "daily_views", EventCount: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Day < result[j].Day })
	var dimensions []ui.NoCookieAnalyticsRow
	for key, count := range aggregates {
		parts := strings.SplitN(key, "\x00", 2)
		dimensions = append(dimensions, ui.NoCookieAnalyticsRow{Metric: parts[0], Dimension: parts[1], EventCount: count})
	}
	sort.Slice(dimensions, func(i, j int) bool {
		if dimensions[i].Metric != dimensions[j].Metric {
			return dimensions[i].Metric < dimensions[j].Metric
		}
		if dimensions[i].EventCount != dimensions[j].EventCount {
			return dimensions[i].EventCount > dimensions[j].EventCount
		}
		return dimensions[i].Dimension < dimensions[j].Dimension
	})
	result = append(result, dimensions...)
	return result, unique, nil
}

// enforceDevModeLocalBind requires --dev/HYDRO_DEV_MODE to bind only to a
// loopback address, so a development server is never reachable from outside
// the local machine.
func enforceDevModeLocalBind(address string) error {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("--dev/HYDRO_DEV_MODE requires a valid --address host:port, got %q: %w", address, err)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("--dev/HYDRO_DEV_MODE refused: server must bind to 127.0.0.1, localhost or ::1 (got %q)", host)
	}
	return nil
}

// printDevModeWarning prints a banner so a dev-mode server is never mistaken
// for a production one.
func printDevModeWarning(stderr io.Writer) {
	fmt.Fprint(stderr, `
################################################################
#   DEV MODE - NOT FOR PRODUCTION                              #
#   Enabled via --dev/HYDRO_DEV_MODE; bound to loopback only.  #
################################################################
`)
}

func serve(args []string, stderr io.Writer) (serveErr error) {
	if stderr == nil {
		stderr = io.Discard
	}
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	cgiMode := flags.Bool("cgi", false, "handle one request from a CGI web server")
	fastCGIMode := flags.Bool("fastcgi", false, "serve FastCGI requests on a listener")
	fastCGIAddress := flags.String("fastcgi-address", "127.0.0.1:9001", "FastCGI listener address (loopback by default)")
	sslMode := flags.String("ssl", "", "enable standalone HTTPS with --ssl=auto")
	sslDomain := flags.String("domain", "", "hostname for standalone HTTPS")
	devFlag := flags.Bool("dev", false, "development mode: relaxed headers, demo frontend and Redis fallback; requires a loopback --address")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *cgiMode && *fastCGIMode {
		return errors.New("--cgi and --fastcgi cannot be combined")
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	if *sslMode != "" {
		if *sslMode != "auto" || strings.TrimSpace(*sslDomain) == "" {
			return errors.New("--ssl=auto requires --domain")
		}
		if cfg.HTTP.BehindProxy {
			return errors.New("standalone automatic TLS cannot run behind a reverse proxy")
		}
		cfg.HTTP.AutoTLS, cfg.HTTP.Domain = true, strings.TrimSpace(*sslDomain)
		if cfg.HTTP.PublicOrigin == "" {
			cfg.HTTP.PublicOrigin = "https://" + cfg.HTTP.Domain
		}
		cfg.Security.EnableHSTS = true
		cfg.Security.SecureCookies = true
	}
	if cfg.HTTP.AutoTLS {
		if cfg.HTTP.PublicOrigin == "" {
			cfg.HTTP.PublicOrigin = "https://" + cfg.HTTP.Domain
		}
		cfg.Security.EnableHSTS, cfg.Security.SecureCookies = true, true
		if err := cfg.Validate(); err != nil {
			return err
		}
	}
	extensionRegistry, err := adminstudio.LoadSignedAdminExtensions(cfg.AdminExtensions.ManifestDir, cfg.AdminExtensions.TrustedKeysFile)
	if err != nil {
		return fmt.Errorf("admin extensions: %w", err)
	}
	var tenantResolver *tenant.HostResolver
	var tenantScopes []tenant.Scope
	requireTenantScope := cfg.Tenancy.Enabled
	if cfg.Tenancy.Enabled {
		hostScopes := make(map[string]tenant.Scope, len(cfg.Tenancy.Hosts))
		hostNames := make([]string, 0, len(cfg.Tenancy.Hosts))
		for host := range cfg.Tenancy.Hosts {
			hostNames = append(hostNames, host)
		}
		sort.Strings(hostNames)
		for _, host := range hostNames {
			tenantID := cfg.Tenancy.Hosts[host]
			scope, scopeErr := tenant.NewScope(tenantID, tenantID)
			if scopeErr != nil {
				return fmt.Errorf("tenancy host %q: %w", host, scopeErr)
			}
			hostScopes[host] = scope
			tenantScopes = append(tenantScopes, scope)
		}
		resolver := tenant.NewHostResolver(hostScopes, cfg.Tenancy.RequireHost)
		tenantResolver = &resolver
	}
	devModeEnv := strings.TrimSpace(os.Getenv("HYDRO_DEV_MODE")) == "1"
	devMode := *devFlag || devModeEnv
	if devMode {
		if !devModeEnv {
			// Other packages read HYDRO_DEV_MODE; scope it to this serve call.
			_ = os.Setenv("HYDRO_DEV_MODE", "1")
			defer func() { _ = os.Unsetenv("HYDRO_DEV_MODE") }()
		}
		if err := enforceDevModeLocalBind(cfg.HTTP.Address); err != nil {
			return err
		}
		printDevModeWarning(stderr)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if addr := strings.TrimSpace(os.Getenv("HYDRO_DEBUG_PPROF_ADDR")); addr != "" {
		if host, _, splitErr := net.SplitHostPort(addr); splitErr != nil || (host != "127.0.0.1" && host != "localhost") {
			return errors.New("HYDRO_DEBUG_PPROF_ADDR must bind to 127.0.0.1 or localhost")
		}
		pprofSrv := &http.Server{Addr: addr, ReadHeaderTimeout: 5 * time.Second}
		go func() { _ = pprofSrv.ListenAndServe() }()
		defer func() { _ = pprofSrv.Close() }()
	}
	tracingProfile, err := observability.LoadTracingProfile()
	if err != nil {
		return fmt.Errorf("tracing profile: %w", err)
	}
	shutdownTracing, err := observability.InstallOTLPProvider(ctx, tracingProfile)
	if err != nil {
		return fmt.Errorf("install otlp tracing: %w", err)
	}
	defer func() { _ = shutdownTracing(context.Background()) }()
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer func() {
		stop()
		if cfg.Database.Dialect == database.SQLite && !*cgiMode {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), budget.ShutdownTimeout)
			defer cancel()
			if err := database.CheckpointSQLite(shutdownCtx, db); err != nil {
				serveErr = errors.Join(serveErr, fmt.Errorf("SQLite shutdown checkpoint: %w", err))
			}
		}
		if err := db.Close(); err != nil {
			serveErr = errors.Join(serveErr, fmt.Errorf("close database: %w", err))
		}
	}()
	if cfg.Database.Dialect == database.SQLite {
		if err := database.RecoverSQLite(ctx, db); err != nil {
			return fmt.Errorf("SQLite startup recovery failed: %w", err)
		}
		if stderr != nil {
			_, _ = fmt.Fprintln(stderr, `{"event":"sqlite_recovery","integrity":"ok","wal_checkpoint":"restart"}`)
		}
	}
	transactioner := database.NewTenantTransactor(db, cfg.Database.Dialect, requireTenantScope)
	ipBlocklist := security.NewIPBlocklist(4096)
	banStore := securitystore.BanStore{Dialect: cfg.Database.Dialect}
	if err := refreshIPBlocklist(ctx, db, banStore, ipBlocklist); err != nil {
		return fmt.Errorf("load IP security rules: %w", err)
	}
	if !*cgiMode {
		go runIPBlocklistRefresh(ctx, db, banStore, ipBlocklist)
	}

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	userStore.RequireTenantScope = requireTenantScope
	sessionStore.RequireTenantScope = requireTenantScope
	mobileAuth := auth.NewMobileAuthService(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, mobileAuth, auth.DefaultArgon2Params)
	authSvc.SetPermissionStore(userStore)
	mfaService := &auth.MFAService{Store: userStore, MasterKey: []byte(cfg.Security.MFAMasterKey)}
	recoveryService := &auth.PasswordRecovery{Users: userStore, Tokens: userStore, Sessions: sessionStore, ArgonParams: auth.DefaultArgon2Params}
	invitationService := &auth.InvitationService{Users: userStore, Invites: userStore, ArgonParams: auth.DefaultArgon2Params}
	redisClient, err := newRedisRuntime(ctx, cfg.Redis.URL)
	if err != nil {
		if devMode || strings.EqualFold(string(cfg.Profile), "compact") {
			if stderr != nil {
				_, _ = fmt.Fprintf(stderr, "warning: Redis connection failed (%v); falling back to in-memory rate limiting and cache\n", err)
			}
			redisClient = redisRuntime{}
		} else {
			return fmt.Errorf("connect redis: %w", err)
		}
	}
	defer redisClient.Close()
	newRateLimiter := func(limit int, window time.Duration) security.RateLimiter {
		return newRuntimeRateLimiter(redisClient, cfg.Redis.KeyPrefix, limit, window, requireTenantScope)
	}
	auditRecorder := newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope)
	var socialOAuth *auth.SocialOAuthHandler
	if !cfg.SafeMode {
		socialOAuth, err = configuredSocialOAuth(db, cfg.Database.Dialect, authSvc, auditRecorder, cfg.Security.SecureCookies, requireTenantScope)
		if err != nil {
			return fmt.Errorf("configure social sign-in: %w", err)
		}
	}
	authHTTP := auth.HTTPHandler{Service: authSvc, Audit: auditRecorder, Secure: cfg.Security.SecureCookies, MFA: mfaService, MFARequiredRoles: configuredMFARoles(cfg.Security.MFARequiredRoles), MFADB: db, Transactions: transactioner, LoginLimiter: newRateLimiter(5, time.Minute), MFALimiter: newRateLimiter(5, time.Minute), RecoveryLimiter: newRateLimiter(3, time.Hour), RecoveryConfirmLimiter: newRateLimiter(10, time.Hour), InviteAcceptLimiter: newRateLimiter(10, time.Hour), LoginTarpit: security.NewTarpit(10000, 15*time.Minute, time.Second, 8*time.Second), Recovery: recoveryService, RecoveryDB: db, Invitation: invitationService, InviteDB: db, PublicOrigin: cfg.HTTP.PublicOrigin, SocialOAuth: socialOAuth, LoginSecurity: userStore, TokenLifetimeSettings: authstore.TokenLifetimeStore{DB: db, Dialect: cfg.Database.Dialect}}
	authHTTP.PasswordResetTTL = cfg.PasswordResetDuration()
	authHTTP.InvitationTTL = cfg.InvitationDuration()
	mobileAuthHTTP := auth.MobileHTTPHandler{Auth: authSvc, Mobile: mobileAuth, MFA: mfaService, MFADB: db, LoginLimiter: newRateLimiter(5, time.Minute), MFALimiter: newRateLimiter(5, time.Minute), Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope)}
	memberStore := memberstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	memberSessionStore := memberstore.SessionStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	memberAuditStore := memberstore.AuditStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	modules := enabledModuleSet(cfg.Modules)
	var contentAssistant aicore.Assistant
	var contentAIAudit auditstore.Store
	if modules["ai"] && strings.EqualFold(strings.TrimSpace(os.Getenv("HYDRO_AI_ENABLED")), "true") {
		endpoint, apiKey, model := strings.TrimSpace(os.Getenv("HYDRO_AI_ENDPOINT")), os.Getenv("HYDRO_AI_API_KEY"), strings.TrimSpace(os.Getenv("HYDRO_AI_MODEL"))
		if endpoint == "" || strings.TrimSpace(apiKey) == "" || model == "" {
			return errors.New("AI assistance requires HYDRO_AI_ENDPOINT, HYDRO_AI_API_KEY, and HYDRO_AI_MODEL")
		}
		provider := aicore.OpenAICompatible{Endpoint: endpoint, APIKey: apiKey, Model: model}
		if provider.Validate() != nil {
			return errors.New("AI assistance provider configuration is invalid")
		}
		contentAssistant = aicore.Assistant{Enabled: true, Provider: provider, Timeout: 15 * time.Second}
		contentAIAudit = auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	}
	var pushDispatcher *pushservice.Dispatcher
	if modules["push"] && !cfg.SafeMode {
		repository := pushstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		dispatcher := &pushservice.Dispatcher{Repository: repository}
		vapidPublic, vapidPrivate, vapidSubject := os.Getenv("HYDRO_PUSH_VAPID_PUBLIC_KEY"), os.Getenv("HYDRO_PUSH_VAPID_PRIVATE_KEY"), os.Getenv("HYDRO_PUSH_VAPID_SUBJECT")
		if vapidPublic != "" || vapidPrivate != "" || vapidSubject != "" {
			if vapidPublic == "" || vapidPrivate == "" || vapidSubject == "" {
				return fmt.Errorf("HYDRO_PUSH_VAPID_PUBLIC_KEY, HYDRO_PUSH_VAPID_PRIVATE_KEY, and HYDRO_PUSH_VAPID_SUBJECT must be set together")
			}
			dispatcher.Web = pushservice.WebPushSender{Public: vapidPublic, Private: vapidPrivate, Subject: vapidSubject}
		}
		if credentialsPath := os.Getenv("HYDRO_PUSH_FCM_CREDENTIALS"); credentialsPath != "" {
			credentials, readErr := os.ReadFile(credentialsPath)
			if readErr != nil {
				return fmt.Errorf("read FCM credentials: %w", readErr)
			}
			fcm, fcmErr := pushservice.NewFCMSender(credentials, nil)
			if fcmErr != nil {
				return fmt.Errorf("configure FCM sender: %w", fcmErr)
			}
			dispatcher.Mobile = fcm
		}
		if dispatcher.Web == nil && dispatcher.Mobile == nil {
			return fmt.Errorf("the push module requires VAPID settings or HYDRO_PUSH_FCM_CREDENTIALS")
		}
		pushDispatcher = dispatcher
	}
	var realtimeHub *realtime.Hub
	if modules["realtime"] {
		realtimeHub = &realtime.Hub{RequireTenantScope: requireTenantScope}
	}
	var editPresence *realtime.Presence
	if modules["pages"] || modules["news"] {
		editPresence = &realtime.Presence{TTL: realtime.DefaultPresenceTTL}
	}
	var analyticsStore analyticsstore.Store
	var contentViewQueue *analytics.ContentViewQueue
	var searchQueryQueue *analytics.SearchQueryQueue
	var noCookieQueue *analytics.NoCookieQueue
	var noCookieHasher *analytics.VisitorHasher
	if modules["analytics"] {
		analyticsStore = analyticsstore.Store{Executor: db, Dialect: cfg.Database.Dialect}
		contentViewQueue = analytics.NewContentViewQueue(analyticsStore, 256, 1)
		defer contentViewQueue.Close()
		if modules["search"] {
			searchQueryQueue = analytics.NewSearchQueryQueue(analyticsStore, 256, 1)
			defer searchQueryQueue.Close()
		}
		if cfg.Analytics.NoCookieEnabled {
			noCookieHasher, err = analytics.NewVisitorHasher([]byte(cfg.Analytics.HashKey))
			if err != nil {
				return fmt.Errorf("initialize local analytics: %w", err)
			}
			noCookieQueue = analytics.NewNoCookieQueue(analyticsStore, 256)
			defer noCookieQueue.Close()
		}
	}
	var memberService members.Service
	var memberHTTP membertransport.Handler
	if modules["members"] {
		memberService = members.Service{Members: memberStore, Sessions: memberSessionStore, Audit: memberAuditStore, ArgonParams: auth.DefaultArgon2Params}
	}
	if modules["members"] {
		memberHTTP = membertransport.Handler{Service: memberService, Mobile: &members.MobileAuthService{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, Secure: cfg.Security.SecureCookies, Limiter: newRateLimiter(10, time.Minute), Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope), DB: db, DefaultLocale: cfg.I18N.DefaultLocale, Push: pushDispatcher}
	}
	var smtpSender mailer.Sender
	var emailBrandingSettings mailer.EmailBrandingProvider
	if modules["auth"] {
		emailBrandingSettings = mailstore.EmailBrandingStore{DB: db, Dialect: cfg.Database.Dialect}
	}
	decorateEmailSender := func(sender mailer.Sender) mailer.Sender {
		if emailBrandingSettings == nil {
			return sender
		}
		return mailer.BrandedSender{Next: sender, Settings: emailBrandingSettings, PublicOrigin: cfg.HTTP.PublicOrigin}
	}
	if cfg.Mail.Host != "" && !cfg.SafeMode {
		sender, err := mailer.NewSMTP(mailer.SMTPConfig{Host: cfg.Mail.Host, Port: cfg.Mail.Port, From: cfg.Mail.From, Username: cfg.Mail.Username, Password: cfg.Mail.Password, StartTLS: cfg.Mail.StartTLS, MaxPerSecond: cfg.Mail.MaxPerSecond, MaxPerHour: cfg.Mail.MaxPerHour})
		if err != nil {
			return err
		}
		smtpSender = decorateEmailSender(sender)
		notifier := mailer.NotificationSender{Sender: smtpSender}
		authHTTP.Notifier = notifier
		authHTTP.InviteNotice = notifier
	} else if cfg.Mail.From != "" && !cfg.SafeMode {
		// Without a submission provider, the configured sender address opts into
		// direct STARTTLS delivery to recipient MX servers. Unavailable direct
		// delivery remains in the durable mail spool for retry.
		sender := mailer.MXSender{From: cfg.Mail.From}
		smtpSender = decorateEmailSender(sender)
		notifier := mailer.NotificationSender{Sender: smtpSender}
		authHTTP.Notifier = notifier
		authHTTP.InviteNotice = notifier
	}
	if modules["members"] && smtpSender != nil && strings.TrimSpace(cfg.HTTP.PublicOrigin) != "" {
		memberHTTP.Verification = members.EmailVerificationService{Tokens: memberStore}
		memberHTTP.VerificationTx = transactioner
		memberHTTP.VerificationNotifier = mailer.NotificationSender{Sender: smtpSender}
		memberHTTP.VerificationLimiter = newRateLimiter(3, time.Hour)
		memberHTTP.PublicOrigin = cfg.HTTP.PublicOrigin
		memberHTTP.VerificationEnabled = true
	}
	publicPageCache, err := newPageCache(modules["pages"], budget, redisClient, cfg.Redis.KeyPrefix)
	if err != nil {
		return fmt.Errorf("configure redis page cache: %w", err)
	}
	var publicHTMLCache *server.PublicHTMLCache
	if modules["pages"] || modules["news"] {
		publicHTMLCache = server.NewPublicHTMLCache(server.PublicHTMLCacheOptions{
			MaxEntries: budget.PublicHTMLCacheEntries,
			MaxBytes:   budget.PublicHTMLCacheMaxBytes,
			MaxBody:    budget.PublicHTMLCacheMaxBodyBytes,
			TTL:        budget.CacheTTL,
		})
	}
	staticCacheDir := ""
	if *cgiMode && (modules["pages"] || modules["news"]) {
		if documentRoot := strings.TrimSpace(os.Getenv("DOCUMENT_ROOT")); documentRoot != "" {
			staticCacheDir = filepath.Join(documentRoot, "static_cache")
		}
	}
	contentService := pageservice.ContentService{
		Transactions: transactioner,
		Pages:        pagesstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		Audit:        auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		Outbox:       outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		PublicCache:  publicPageCache,
		Realtime:     realtimeHub,
		Sanitize:     ui.NewHTMLSanitizer().Sanitize,
	}
	if staticCacheDir != "" || publicHTMLCache != nil {
		contentService.PublicRoutePurge = func(locale, slug string) {
			path := "/public/pages/" + url.PathEscape(locale) + "/" + url.PathEscape(slug) + "/html"
			if publicHTMLCache != nil {
				publicHTMLCache.PurgePath(path)
			}
			if staticCacheDir != "" {
				if err := server.PurgeStaticDiskCachePath(staticCacheDir, path); err != nil {
					slog.Warn("published page cache invalidation failed", "error", err)
				}
			}
		}
	}
	var sqliteCacheConn *sql.Conn
	if !*cgiMode && cfg.Database.SQLiteExternalWriteWatch && cfg.Database.Dialect == database.SQLite && (publicPageCache != nil || publicHTMLCache != nil) {
		sqliteCacheConn, err = db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("open sqlite cache monitor connection: %w", err)
		}
		defer sqliteCacheConn.Close()
		monitor := cache.NewSQLiteDataVersionMonitor(db)
		monitor.ReadVersion = func(readCtx context.Context) (int64, error) {
			var version int64
			err := sqliteCacheConn.QueryRowContext(readCtx, "PRAGMA data_version").Scan(&version)
			return version, err
		}
		monitor.OnChange = func() { clearPublicCaches(publicPageCache, publicHTMLCache) }
		monitor.OnError = func(monitorErr error) {
			if stderr != nil {
				_, _ = fmt.Fprintf(stderr, "hydro sqlite cache monitor: %v\n", monitorErr)
			}
		}
		go func() {
			if monitorErr := monitor.Run(ctx); monitorErr != nil && ctx.Err() == nil && stderr != nil {
				_, _ = fmt.Fprintf(stderr, "hydro sqlite cache monitor stopped: %v\n", monitorErr)
			}
		}()
	}
	if cfg.Backup.Enabled && !cfg.SafeMode && !*cgiMode && !*fastCGIMode {
		dispatcher, backupErr := backupstore.NewS3Dispatcher(backupstore.S3Config{Endpoint: cfg.Media.S3Endpoint, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Prefix: cfg.Backup.Prefix, Secure: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle})
		if backupErr != nil {
			return fmt.Errorf("configure scheduled cloud backup: %w", backupErr)
		}
		go backupstore.RunDaily(ctx, cfg.Backup.HourUTC, func(backupCtx context.Context) error {
			root, err := os.MkdirTemp("", "hydrogo-backup-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(root)
			snapshot := filepath.Join(root, "snapshot")
			if cfg.Database.Dialect == database.SQLite {
				_, err = backupstore.SnapshotSQLite(backupCtx, db, cfg.Media.Dir, snapshot)
			} else if cfg.Database.Dialect == database.Postgres {
				var tools backupstore.PostgresTools
				tools, err = backupstore.LocatePostgresTools()
				if err == nil {
					_, err = backupstore.SnapshotPostgres(backupCtx, tools, cfg.Database.DSN, cfg.Media.Dir, snapshot)
				}
			} else {
				err = errors.New("scheduled cloud backups support SQLite and PostgreSQL")
			}
			if err != nil {
				return err
			}
			name := fmt.Sprintf("hydrogo-%s.tar.gz", time.Now().UTC().Format("20060102T150405Z"))
			if err = dispatcher.UploadSnapshot(backupCtx, snapshot, name); err != nil {
				return err
			}
			_, err = dispatcher.Prune(backupCtx, time.Duration(cfg.Backup.RetentionDays)*24*time.Hour, time.Now().UTC())
			return err
		}, func(err error) { slog.Error("scheduled cloud backup failed", "error", err) })
	}
	var selectedTheme *themev1.ThemeBundle
	themeName := cfg.Theme
	if themeName == "" {
		themeName = "corporate"
	}
	if themeName != "" {
		themeDir := cfg.ThemeDir
		if !filepath.IsAbs(themeDir) && values.configPath != nil && *values.configPath != "" {
			candidate := filepath.Join(filepath.Dir(*values.configPath), themeDir)
			if info, statErr := os.Stat(filepath.Join(candidate, themeName)); statErr == nil && info.IsDir() {
				themeDir = candidate
			}
		}
		bundle, err := themev1.LoadTheme(themeDir, themeName)
		if err == nil {
			selectedTheme = &bundle
		} else if cfg.Theme != "" {
			return fmt.Errorf("load theme %q: %w", cfg.Theme, err)
		}
	}
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
		if !*cgiMode {
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
		mediaService = media.Service{Transactions: transactioner, Uploader: uploader, BlobStore: blobStore, Assets: mediaStore, Optimizer: mediaOptimizer, AdminSettings: mediaStore, DefaultWatermark: configuredWatermark}
		if mediaOptimizer != nil {
			go func() { <-ctx.Done(); mediaOptimizer.Close() }()
		}
	}
	mediaHTTP := media.HTTPHandler{Service: mediaService, DB: db, HotlinkProtection: cfg.Media.HotlinkProtection, CanonicalOrigin: cfg.HTTP.PublicOrigin, AllowedRefererHosts: cfg.Media.AllowedRefererHosts, SettingsRepository: mediaStore, SettingsCache: media.NewAdminSettingsCache(), SettingsDefaults: media.AdminSettings{HotlinkProtection: cfg.Media.HotlinkProtection, AllowedRefererHosts: cfg.Media.AllowedRefererHosts, WatermarkEnabled: configuredWatermark != nil, WatermarkText: cfg.Media.WatermarkText, WatermarkPosition: cfg.Media.WatermarkPosition, WatermarkOpacity: cfg.Media.WatermarkOpacity, WatermarkMargin: cfg.Media.WatermarkMargin}, RequireTenantScope: requireTenantScope}
	if modules["media"] {
		mediaHTTP.Garbage = &media.GarbageCollector{DB: db, Repository: mediaStore, References: mediastore.MediaReferences{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Pages: modules["pages"], News: modules["news"], Site: modules["site"], Forms: modules["forms"]}, BlobStore: mediaService.BlobStore, StorageDir: cfg.Media.Dir, Retention: 7 * 24 * time.Hour}
	}
	pageStore := pagesstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	siteStore := sitestore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	newsRouteStore := newsstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	var spoolStore mailstore.Spool
	var mailWorker mailer.SpoolWorker
	if modules["forms"] {
		spoolStore = mailstore.Spool{DB: db, Tx: transactioner, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, MaxPending: budget.QueueEntries}
		if smtpSender != nil {
			mailWorker = mailer.SpoolWorker{Store: spoolStore, Sender: smtpSender, WorkerID: "mail-spool-worker-0", Lease: time.Minute, TenantScopes: tenantScopes}
			if stderr != nil {
				mailWorker.OnError = func(err error) { _, _ = fmt.Fprintf(stderr, "hydro mail spool: %v\n", err) }
			}
		}
	}
	var siteService site.Service
	var siteHTTP sitetransport.Handler
	var formViewLimiter security.RateLimiter
	var linkScanManager *linkcheck.Manager
	var linkScanOrigin string
	var redirectRegistry *site.RedirectRegistry
	if modules["site"] {
		siteService = site.Service{Transactions: transactioner, Store: siteStore, Audit: auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, RequireTenantScope: requireTenantScope}
		if modules["forms"] && noCookieHasher != nil {
			siteService.FormMetrics = analyticsStore
			formViewLimiter = newRateLimiter(120, time.Minute)
		}
		redirectRegistry = &site.RedirectRegistry{RequireTenantScope: requireTenantScope}
		redirectScopes := tenantScopes
		if !requireTenantScope {
			redirectScopes = []tenant.Scope{{}}
		}
		for _, scope := range redirectScopes {
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
		}
		if modules["forms"] {
			siteService.Forms = formstore.Store{Executor: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
			siteService.MailQueue = spoolStore
		}
		sitemapOrigin := cfg.HTTP.PublicOrigin
		if sitemapOrigin == "" && cfg.HTTP.Address != "" {
			sitemapOrigin = "http://" + cfg.HTTP.Address
		}
		linkScanOrigin = cfg.HTTP.PublicOrigin
		linkScanManager = linkcheck.NewManager(linkcheck.Options{MaxPages: 500, MaxTargets: 5000, MaxFindings: 1000, Concurrency: 4, RequestDelay: 100 * time.Millisecond, Retention: 35 * 24 * time.Hour})
		siteHTTP = sitetransport.Handler{DB: db, Store: siteStore, Service: &siteService, Media: &mediaService, Redirects: redirectRegistry, FontsDir: cfg.SEO.FontsDir, FontsEnabled: modules["pages"], SubmissionLimiter: newRateLimiter(3, time.Hour), FormViewLimiter: formViewLimiter, FormMetrics: analyticsStore, VisitorHasher: noCookieHasher, RequireTenantScope: requireTenantScope, FormsEnabled: modules["forms"], Realtime: realtimeHub, SitemapOrigin: sitemapOrigin, LinkScanManager: linkScanManager, LinkScanLimiter: newRateLimiter(1, time.Minute), SitemapProvider: func(providerCtx context.Context) ([]site.SitemapEntry, error) {
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
		if contentAssistant.Enabled {
			siteHTTP.Assistant = contentAssistant
			siteHTTP.AIAudit = contentAIAudit
		}
		siteHTTP.NavigationLocales = cfg.I18N.Locales
		if modules["forms"] {
			proxyRanges, geoErr := geolocation.ParseTrustedProxyRanges(cfg.GeoIP.TrustedProxyCIDRs)
			if geoErr != nil {
				return fmt.Errorf("configure form geolocation: %w", geoErr)
			}
			resolver := geolocation.Resolver{TrustedProxyRanges: proxyRanges}
			if cfg.GeoIP.DatabasePath != "" {
				db, openErr := geolocation.OpenMaxMind(cfg.GeoIP.DatabasePath)
				if openErr != nil {
					return fmt.Errorf("open form geolocation database: %w", openErr)
				}
				defer db.Close()
				resolver.Database = db
			}
			siteHTTP.GeoResolver = resolver
		}
		if moduleEnabled(cfg.Modules, "pages") {
			siteHTTP.NavigationSuggestions = func(suggestionCtx context.Context, locale string) ([]site.NavigationSuggestion, error) {
				suggestions := make([]site.NavigationSuggestion, 0, 200)
				limitReached := errors.New("navigation suggestion limit reached")
				err := pageStore.StreamPublishedRoutes(suggestionCtx, db, func(page pages.Page) error {
					if page.Locale != locale {
						return nil
					}
					suggestions = append(suggestions, site.NavigationSuggestion{Label: page.Title, Href: "/public/pages/" + url.PathEscape(page.Locale) + "/" + url.PathEscape(page.Slug) + "/html"})
					if len(suggestions) == cap(suggestions) {
						return limitReached
					}
					return nil
				})
				if errors.Is(err, limitReached) {
					return suggestions, nil
				}
				return suggestions, err
			}
		}
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
		if modules["forms"] {
			siteHTTP.AllowedForms = map[string]struct{}{"contact": {}}
			siteHTTP.MailSpool = spoolStore
			if cfg.Forms.TurnstileSecretKey != "" {
				siteHTTP.Turnstile = formsruntime.TurnstileVerifier{Secret: cfg.Forms.TurnstileSecretKey}
			}
		}
	}
	var newsService newsservice.Service
	var newsWorker newsservice.Worker
	if moduleEnabled(cfg.Modules, "news") {
		newsService = newsservice.Service{Transactions: transactioner, Repository: newsRouteStore, Realtime: realtimeHub, Audit: auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, WorkflowRole: func(ctx context.Context) string {
			if user := auth.UserFromContext(ctx); user != nil {
				return string(user.Role)
			}
			return ""
		}}
		newsWorker = newsservice.Worker{
			Durable:      jobstore.Queue{Store: jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: budget.QueueEntries}, DB: db},
			Service:      newsService,
			Lease:        time.Minute,
			Retry:        time.Minute,
			TenantScopes: tenantScopes,
		}
	}

	studioRegistry, err := adminstudio.NewRegistry([]adminstudio.Collection{
		{ID: "pages", Label: "Sayfalar", Description: "Taslak, bloklar ve yayınlanmış içerik.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}, {ID: "blocks", Label: "Bloklar", Kind: adminstudio.FieldRelation}}, Actions: []adminstudio.Action{{Label: "Tüm sayfalar", Href: "/admin/pages"}, {Label: "Ana sayfayı düzenle", Href: "/admin/pages/demo-home"}, {Label: "Showcase'i düzenle", Href: "/admin/pages/demo-showcase"}, {Label: "Hakkında sayfasını düzenle", Href: "/admin/pages/demo-about"}}},
		{ID: "news", Label: "Haberler", Description: "Taslak, zamanlama ve yayın akışı.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "cover_media_id", Label: "Kapak görseli (herkese açık asset)", Kind: adminstudio.FieldText, Widget: "media"}, {ID: "excerpt", Label: "Kısa özet", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "key_takeaways", Label: "Önemli Noktalar (Key Takeaways)", Kind: adminstudio.FieldText, Widget: "list"}, {ID: "category", Label: "Kategori", Kind: adminstudio.FieldText}, {ID: "author_name", Label: "Yazar adı", Kind: adminstudio.FieldText}, {ID: "featured", Label: "Öne çıkan", Kind: adminstudio.FieldText, Widget: "checkbox"}, {ID: "tags", Label: "Etiketler", Kind: adminstudio.FieldText}, {ID: "seo_title", Label: "SEO başlığı", Kind: adminstudio.FieldText}, {ID: "seo_description", Label: "SEO açıklaması", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}}, Actions: []adminstudio.Action{{Label: "Tüm haberler", Href: "/admin/news"}, {Label: "Release notes", Href: "/admin/news/demo-news"}, {Label: "Başlangıç rehberi", Href: "/admin/news/demo-news-guide"}}},
	})
	if err != nil {
		return fmt.Errorf("admin studio registry: %w", err)
	}
	activeStudioCollections := func() []adminstudio.Collection {
		return adminstudio.FilterActiveCollections(studioRegistry.Collections(), modules)
	}
	var pageHTTP pagetransport.Handler
	var previewSigner *preview.Signer
	if len(cfg.Security.PreviewSigningKey) >= 32 {
		previewSigner = &preview.Signer{Key: []byte(cfg.Security.PreviewSigningKey)}
	}
	var pageLLMSCache *seo.DiscoveryCache
	if moduleEnabled(cfg.Modules, "pages") {
		pageCollection, ok := studioRegistry.Collection("pages")
		if !ok {
			return errors.New("admin studio pages collection missing")
		}
		for i := range pageCollection.Fields {
			if pageCollection.Fields[i].ID == "body" {
				pageCollection.Fields[i].Widget = "richtext"
			}
			if pageCollection.Fields[i].ID == "locale" {
				if !cfg.I18N.Enabled {
					pageCollection.Fields[i].Widget = "hidden"
					pageCollection.Fields[i].VisibleInList = false
				} else if len(cfg.I18N.Locales) > 0 {
					pageCollection.Fields[i].Options = cfg.I18N.Locales
				}
			}
		}
		pageCollection.BulkActions = []adminstudio.Action{{Label: "CSV dışa aktar", Href: "/admin/pages/bulk"}}
		pageLLMSCache = &seo.DiscoveryCache{}
		notificationStore := adminstudiosql.Store{Dialect: cfg.Database.Dialect}
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
		pageHTTP = pagetransport.Handler{DB: db, Content: contentService, Theme: selectedTheme, Media: mediaService, AdminCollection: pageCollection, DefaultLocale: cfg.I18N.DefaultLocale, SupportedLocales: cfg.I18N.Locales, Assistant: contentAssistant, AIAudit: contentAIAudit, AllowDemoFrontend: demoFrontendEnabled() || devMode, SEOOrigin: cfg.HTTP.PublicOrigin, PreviewSigner: previewSigner, PreviewOrigin: cfg.HTTP.PublicOrigin, RequireTenantScope: requireTenantScope, SearchEnabled: modules["search"], ContentViews: analyticsStore, Feedback: analyticsStore, AnalyticsEnabled: modules["analytics"], Presence: editPresence, RecordSlugRedirect: recordSlugRedirect, RecordNotFound: recordNotFound, PublicSupport: adminstudio.SupportConfig{Enabled: cfg.AdminSupport.Enabled, Phone: cfg.AdminSupport.Phone, WhatsAppURL: cfg.AdminSupport.WhatsAppURL}, OGRenderer: &seo.OGRenderer{CacheDir: cfg.SEO.OGCacheDir}, DiscoveryCache: pageLLMSCache, Notifications: &notificationStore, WeeklyDigestEnabled: modules["forms"] && smtpSender != nil, NavigationProvider: func(navCtx context.Context, locale string) ([]site.NavigationItem, error) {
			if !modules["site"] {
				return nil, nil
			}
			return siteStore.ListNavigation(navCtx, db, locale)
		}, PublicSEOSettings: func(seoCtx context.Context) pagetransport.PublicSEOSettings {
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
				case "design.font_css":
					result.FontCSS = setting.Value
				}
			}
			appearance := site.AppearanceFromSettings(settings)
			if css, err := appearance.CSS(); err == nil {
				result.AppearanceCSS = css
			}
			result.FaviconURL = appearance.FaviconURL
			result.SocialImageURL = appearance.SocialImageURL
			if profile, ok := site.BusinessProfileFromSettings(settings); ok {
				result.BusinessProfile = &profile
			}
			return result
		}}
		if modules["site"] {
			pageHTTP.PublicContactSettings = func(contactCtx context.Context) ([]site.Setting, error) {
				return siteStore.ListPublicSettings(contactCtx, db)
			}
		}
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
				discovery.Pages = append(discovery.Pages, seo.DiscoveryPage{Title: page.Title, Path: "/" + page.Locale + "/" + page.Slug, Text: string(ui.RichTextHTML(page.Body))})
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
					discovery.Pages = append(discovery.Pages, seo.DiscoveryPage{Title: title, Path: "/public/news/" + item.Locale + "/" + item.Slug, Description: item.SEODescription, Text: string(ui.RichTextHTML(text))})
					return nil
				}); err != nil {
					return seo.Discovery{}, err
				}
			}
			return discovery, nil
		}
		if modules["site"] {
			siteHTTP.InvalidateDiscovery = pageLLMSCache.Invalidate
		}
	}
	var newsHTTP newstransport.Handler
	if moduleEnabled(cfg.Modules, "news") {
		newsCollection, ok := studioRegistry.Collection("news")
		if !ok {
			return errors.New("admin studio news collection missing")
		}
		for i := range newsCollection.Fields {
			if newsCollection.Fields[i].ID == "body" {
				newsCollection.Fields[i].Widget = "richtext"
			}
			if newsCollection.Fields[i].ID == "locale" {
				if !cfg.I18N.Enabled {
					newsCollection.Fields[i].Widget = "hidden"
					newsCollection.Fields[i].VisibleInList = false
				} else if len(cfg.I18N.Locales) > 0 {
					newsCollection.Fields[i].Options = cfg.I18N.Locales
				}
			}
		}
		newsHTTP = newstransport.Handler{Service: newsService, DB: db, Media: mediaService, MediaLookup: mediaService, AdminCollection: newsCollection, DefaultLocale: cfg.I18N.DefaultLocale, PublicOrigin: cfg.HTTP.PublicOrigin, ContentViews: analyticsStore, AnalyticsEnabled: modules["analytics"], Presence: editPresence, RequireTenantScope: requireTenantScope, PreviewSigner: previewSigner, PublicSupport: adminstudio.SupportConfig{Enabled: cfg.AdminSupport.Enabled, Phone: cfg.AdminSupport.Phone, WhatsAppURL: cfg.AdminSupport.WhatsAppURL}, NavigationProvider: func(navCtx context.Context, locale string) ([]themev1.ThemeNavigationItem, error) {
			if !modules["site"] {
				return nil, nil
			}
			items, err := siteStore.ListNavigation(navCtx, db, locale)
			if err != nil {
				return nil, err
			}
			navigation := make([]themev1.ThemeNavigationItem, 0, len(items))
			for _, item := range items {
				navigation = append(navigation, themev1.ThemeNavigationItem{ID: item.ID, ParentID: item.ParentID, Label: item.Label, Href: item.Href, Target: item.Target, Position: item.Position})
			}
			return navigation, nil
		}}
		if modules["analytics"] {
			newsHTTP.Feedback = analyticsStore
		}
		if modules["site"] {
			newsHTTP.PublicContactSettings = func(contactCtx context.Context) ([]site.Setting, error) {
				return siteStore.ListPublicSettings(contactCtx, db)
			}
		}
	}
	routes := append([]router.Route{}, authHTTP.RouterRoutes()...)
	if modules["auth"] {
		routes = append(routes, router.Route{Pattern: "GET /auth/invites/accept", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			validToken := len(token) == 64
			if validToken {
				_, err := hex.DecodeString(token)
				validToken = err == nil
			}
			locale := "en"
			if r.URL.Query().Get("lang") == "tr" {
				locale = "tr"
			}
			message := r.URL.Query().Get("error")
			switch message {
			case "mismatch", "weak", "rate", "unavailable":
			default:
				if !validToken {
					message = "invalid"
				} else {
					message = ""
				}
			}
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Language", locale)
			var page bytes.Buffer
			props := designsystem.InviteAcceptFormProps{Token: token, CSRF: security.CSRFTokenFromContext(r.Context()), Locale: locale, Error: message}
			if err := designsystem.InviteAcceptForm(props).Render(r.Context(), &page); err != nil {
				http.Error(w, "Invitation page is temporarily unavailable.", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Security-Policy", security.MergeCSP(w.Header().Get("Content-Security-Policy"), security.InlineCSPDirectives(page.Bytes())))
			_, _ = w.Write(page.Bytes())
		})})
		tokenSettingsHandler := adminTokenLifetimeSettingsHandler{
			Settings: authHTTP.TokenLifetimeSettings,
			Defaults: auth.TokenLifetimes{PasswordReset: cfg.PasswordResetDuration(), Invitation: cfg.InvitationDuration()},
			Audit:    auditRecorder,
		}
		routes = append(routes,
			router.Route{Pattern: "GET /admin/security/token-lifetimes", Access: router.Protected, Handler: tokenSettingsHandler},
			router.Route{Pattern: "POST /admin/security/token-lifetimes", Access: router.Protected, Handler: tokenSettingsHandler},
		)
	}
	if modules["collections"] {
		collectionDB := collectionstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		collectionService := collections.Service{Repository: collectionDB}
		contentRoutes, err := contentTypeRoutes(ctx, cfg.ContentTypesDir, collectionService, tenantScopes)
		if err != nil {
			return err
		}
		routes = append(routes, contentRoutes...)
	}
	if modules["push"] {
		routes = append(routes, (pushservice.Handler{Dispatcher: *pushDispatcher, Limiter: newRateLimiter(3, time.Minute), Audit: auditRecorder}).Routes()...)
	}
	trashKinds := map[trash.Kind]bool{trash.Pages: modules["pages"], trash.News: modules["news"], trash.Submissions: modules["site"], trash.Media: modules["media"]}
	if trashKinds[trash.Pages] || trashKinds[trash.News] || trashKinds[trash.Submissions] || trashKinds[trash.Media] {
		trashRepository := trashstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Enabled: trashKinds}
		trashHandler := trashtransport.Handler{Service: trash.Service{
			Repository:   trashRepository,
			Transactions: transactioner,
			Audit:        auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		}}
		trashHandler.AfterMutation = func(_ context.Context, kind trash.Kind, _ string) {
			if kind == trash.Pages || kind == trash.News {
				if publicPageCache != nil {
					publicPageCache.Clear()
				}
				if publicHTMLCache != nil {
					publicHTMLCache.Clear()
				}
				if staticCacheDir != "" {
					if err := server.PurgeStaticDiskCache(staticCacheDir); err != nil {
						slog.Warn("trash cache invalidation failed", "error", err)
					}
				}
			}
		}
		routes = append(routes, trashHandler.Routes()...)
	}
	sessionPage := server.SessionPageHandler{Service: authSvc}
	routes = append(routes, router.Route{Pattern: "GET /auth/me/sessions/manage", Access: router.Protected, Handler: sessionPage}, router.Route{Pattern: "POST /auth/me/sessions/manage/revoke-others", Access: router.Protected, Handler: sessionPage}, router.Route{Pattern: "POST /auth/me/sessions/manage/{id}/revoke", Access: router.Protected, Handler: sessionPage})
	if modules["search"] {
		routes = append(routes, router.Route{Pattern: "GET /api/v1/search/autocomplete", Access: router.Public, Handler: adminsearch.AutocompleteHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Modules: modules}})
		autocomplete := adminsearch.AutocompleteHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Modules: modules}
		routes = append(routes, router.Route{Pattern: "GET /search", Access: router.Public, Handler: adminsearch.PublicSearchHandler{Autocomplete: autocomplete, Recorder: searchQueryQueue}})
		if modules["analytics"] {
			routes = append(routes, router.Route{Pattern: "GET /admin/search-insights", Access: router.Protected, Handler: adminsearch.SearchInsightsHandler(analyticsStore, requireTenantScope)})
		}
	}
	routes = append(routes, mobileAuthHTTP.RouterRoutes()...)
	if modules["pages"] {
		pageRoutes := pageRouterRoutes(pageHTTP.Routes())
		for i := range pageRoutes {
			if strings.HasPrefix(pageRoutes[i].Pattern, "GET /public/pages/") {
				if modules["analytics"] {
					pageRoutes[i].Handler = analytics.NoCookieMiddleware(analytics.ContentViewMiddleware(pageRoutes[i].Handler, contentViewQueue, "page"), noCookieQueue, noCookieHasher)
				}
			} else if !strings.HasPrefix(pageRoutes[i].Pattern, "GET ") {
				original := pageRoutes[i].Handler
				pageRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if pageLLMSCache != nil {
						defer pageLLMSCache.Invalidate()
					}
					original.ServeHTTP(w, r)
				})
			}
		}
		routes = append(routes, pageRoutes...)
	}
	if enabledModuleSet(cfg.Modules)["members"] {
		routes = append(routes, memberHTTP.Routes()...)
	}
	localeEnabled := cfg.I18N.Enabled && len(cfg.I18N.Locales) > 0
	routes = append(routes, router.Route{Pattern: "GET /{locale}/", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := configuredLocale(r.PathValue("locale"), cfg.I18N.Locales)
		if !localeEnabled || locale == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, technicalHomeWithLocales(locale, cfg.I18N.Locales, publicHomePath(locale), demoFrontendEnabled()))
	})}, router.Route{Pattern: "GET /{$}", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if localeEnabled {
				selected, manual := requestedLocale(r, cfg.I18N.Locales, cfg.I18N.DefaultLocale)
				if manual {
					http.SetCookie(w, &http.Cookie{Name: "hydro_locale", Value: selected, Path: "/", MaxAge: 31536000, HttpOnly: true, Secure: cfg.Security.SecureCookies, SameSite: http.SameSiteLaxMode})
					localeRedirect(w, r, selected)
					return
				}
				if selected != "" && !strings.EqualFold(selected, cfg.I18N.DefaultLocale) {
					localeRedirect(w, r, selected)
					return
				}
			}
			homePath := ""
			if modules["site"] {
				if settings, err := siteStore.ListPublicSettings(r.Context(), db); err == nil {
					homePath = site.ResolvePublicHomePath(settings, "")
				}
			}
			if homePath == "" && modules["news"] {
				homePath = "/public/news/" + url.PathEscape(cfg.I18N.DefaultLocale) + "/html"
			}
			if homePath != "" && homePath != "/" {
				http.Redirect(w, r, homePath, http.StatusFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, technicalHomeWithLocales(cfg.I18N.DefaultLocale, cfg.I18N.Locales, publicHomePath(cfg.I18N.DefaultLocale), demoFrontendEnabled()))
			return
		}
		http.NotFound(w, r)
	})})
	routes = append(routes, router.Route{Pattern: "GET /docs", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		if lang != "tr" {
			lang = "en"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Language", lang)
		_, _ = io.WriteString(w, docsPortalHTML(lang))
	})})
	if modules["analytics"] {
		routes = append(routes, router.Route{Pattern: "POST /analytics/consent", Access: router.Public, Handler: analytics.ConsentHandler(analyticsStore, cfg.Security.SecureCookies)})
		routes = append(routes, router.Route{Pattern: "POST /analytics/contact", Access: router.Public, Handler: analytics.ContactClickHandler(analyticsStore)})
		routes = append(routes, router.Route{Pattern: "POST /api/v1/feedback", Access: router.Public, Handler: analytics.FeedbackHandler{Store: analyticsStore, Limiter: newRateLimiter(8, time.Hour), RequireTenantScope: requireTenantScope}})
	}
	if modules["pages"] || modules["news"] || modules["members"] || modules["media"] || modules["site"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/api/search", Access: router.Protected, Handler: adminsearch.AdminSearchHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope, Modules: modules}})
	}
	routes = append(routes, router.Route{Pattern: "GET /admin/studio/manifest", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(adminstudio.E2EManifestForCollections(activeStudioCollections()))
	})})
	routes = append(routes, router.Route{Pattern: "GET /admin/extensions/manifest", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(extensionRegistry.Manifests())
	})})
	routes = append(routes, router.Route{Pattern: "GET /admin/extensions/{extension_id}/{surface_id}", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		session := auth.SessionFromContext(r.Context())
		if user == nil || session == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		principal := adminstudio.ChannelPrincipal{SessionID: session.ID, UserID: user.ID}
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			principal.TenantID = scope.TenantID
		}
		extensionRegistry.ServeExtensionSurface(w, r, r.PathValue("extension_id"), r.PathValue("surface_id"), cfg.I18N.DefaultLocale, principal, func(p string) bool { return auth.Can(user, p) })
	})})
	routes = append(routes,
		router.Route{Pattern: "POST /admin/extensions/channels/bind", Access: router.Protected, Handler: adminExtensionChannelBindHandler(extensionRegistry)},
		router.Route{Pattern: "POST /admin/extensions/channels/revoke", Access: router.Protected, Handler: adminExtensionChannelRevokeHandler(extensionRegistry)},
	)
	routes = append(routes, router.Route{Pattern: "GET /password-reset", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		validToken := len(token) == 64
		if validToken {
			_, err := hex.DecodeString(token)
			validToken = err == nil
		}
		locale := "en"
		if r.URL.Query().Get("lang") == "tr" {
			locale = "tr"
		}
		message := r.URL.Query().Get("error")
		switch message {
		case "mismatch", "weak", "rate", "unavailable":
		default:
			if !validToken {
				message = "invalid"
			} else {
				message = ""
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Language", locale)
		var page bytes.Buffer
		props := designsystem.PasswordResetFormProps{Token: token, CSRF: security.CSRFTokenFromContext(r.Context()), Locale: locale, Error: message}
		if err := designsystem.PasswordResetForm(props).Render(r.Context(), &page); err != nil {
			http.Error(w, "Password reset is temporarily unavailable.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Security-Policy", security.MergeCSP(w.Header().Get("Content-Security-Policy"), security.InlineCSPDirectives(page.Bytes())))
		_, _ = w.Write(page.Bytes())
	})})
	routes = append(routes, router.Route{Pattern: "GET /login", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		returnTo := security.SafeReturnPath(r.URL.Query().Get("return_to"), "/admin")
		loginLocale := "en"
		if r.URL.Query().Get("lang") == "tr" {
			loginLocale = "tr"
		}
		if user := auth.UserFromContext(r.Context()); user != nil && user.Active && user.Role == auth.RoleAdmin {
			http.Redirect(w, r, returnTo, http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Language", loginLocale)
		appearance := site.Appearance{LoginOverlayOpacity: .4}
		if modules["site"] {
			if settings, settingsErr := siteStore.ListPublicSettings(r.Context(), db); settingsErr == nil {
				appearance = site.AppearanceFromSettings(settings)
			}
		}
		var page bytes.Buffer
		if err := ui.RenderAdminLoginWithLocaleAndAppearance(&page, security.CSRFTokenFromContext(r.Context()), returnTo, loginLocale, appearance); err != nil {
			http.Error(w, "admin login unavailable", http.StatusInternalServerError)
			return
		}
		var successMessage string
		if r.URL.Query().Get("invite_accepted") == "1" {
			successMessage = `Your account is ready. Sign in with the password you just created.`
			if loginLocale == "tr" {
				successMessage = `Hesabın hazır. Az önce belirlediğin parolayla giriş yap.`
			}
		} else if r.URL.Query().Get("password_reset") == "1" {
			successMessage = `Your password has been updated. Sign in with your new password.`
			if loginLocale == "tr" {
				successMessage = `Parolan güncellendi. Yeni parolanla giriş yap.`
			}
		}
		if successMessage != "" {
			message := `<div id="login-status" class="form-alert alert-success" role="status">` + successMessage + `</div>`
			pageBytes := bytes.Replace(page.Bytes(), []byte(`<div id="login-status" class="form-alert" role="alert"></div>`), []byte(message), 1)
			page.Reset()
			_, _ = page.Write(pageBytes)
		}
		w.Header().Set("Content-Security-Policy", security.MergeCSP(w.Header().Get("Content-Security-Policy"), ui.AdminLoginCSP(page.Bytes())))
		_, _ = w.Write(page.Bytes())
	})})
	var runtimeMetrics *observability.Metrics
	adminDashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active {
			http.Error(w, "Bu alana erişmek için giriş yapın.", http.StatusUnauthorized)
			return
		}
		if linkScanManager != nil {
			tenantKey := "default"
			if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
				tenantKey = scope.TenantID
			}
			if job, ok := linkScanManager.Latest(tenantKey); ok && len(job.Findings) > 0 {
				support := adminstudio.SupportConfigFromContext(r.Context())
				support.BrokenLinksActive, support.BrokenLinksJobID = true, job.ID
				r = r.WithContext(adminstudio.WithSupportConfig(r.Context(), support))
			}
		}
		if user.Role == auth.RoleEditor && r.URL.Path == "/admin" {
			securityNotice, noticeErr := userStore.LoginSecuritySummary(r.Context(), user.ID)
			if noticeErr != nil {
				http.Error(w, "Yönetim özeti şu anda açılamıyor.", http.StatusInternalServerError)
				return
			}
			previousLogin := ""
			if securityNotice.PreviousLoginAt != nil {
				previousLogin = securityNotice.PreviousLoginAt.Local().Format("02 Jan 2006 15:04 MST")
			}
			if err := ui.RenderBusinessOwnerDashboardWithSecurityNotice(w, r, modules, previousLogin, securityNotice.PreviousLoginIP, securityNotice.FailedAttempts); err != nil {
				http.Error(w, "Bu sayfa şu anda açılamıyor.", http.StatusInternalServerError)
			}
			return
		}
		if user.Role != auth.RoleAdmin {
			http.Error(w, "Bu alana erişim yetkiniz yok.", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/admin/reports" {
			var stats members.MemberStats
			if modules["members"] {
				stats, _ = memberService.Stats(r.Context())
			}
			dataTime := time.Now().Format("02.01.2006 15:04:05")
			dateRange := r.URL.Query().Get("range")
			dateFrom, dateTo := "", ""
			if dateRange == "custom" {
				var valid bool
				dateFrom, dateTo, valid = customAnalyticsDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
				if !valid {
					http.Error(w, "Choose a valid date range within the past year.", http.StatusBadRequest)
					return
				}
			}
			contentViews := []analyticsstore.ContentViewAggregate(nil)
			var noCookieStats []ui.NoCookieAnalyticsRow
			var uniqueVisitors int64
			if modules["analytics"] {
				contentViews, _ = analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 100))
				tenantID := ""
				if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
					tenantID = scope.TenantID
				}
				noCookieStats, uniqueVisitors, _ = dailyNoCookieAnalytics(r.Context(), analyticsStore, tenantID, time.Now().UTC().Format("2006-01-02"))
			}
			contentViewRows := make([]ui.ContentViewReportRow, 0, len(contentViews))
			for _, row := range contentViews {
				contentViewRows = append(contentViewRows, ui.ContentViewReportRow{Day: row.Day, TenantID: row.TenantID, ContentType: row.ContentType, ContentID: row.ContentID, Locale: row.Locale, ViewCount: row.ViewCount})
			}
			if err := ui.RenderAdminReports(w, ui.AdminDashboardData{
				User:        user,
				Email:       user.Email,
				Collections: activeStudioCollections(),

				Active:                   "reports",
				Metrics:                  runtimeMetrics,
				MemberStats:              stats,
				DataTime:                 dataTime,
				DataSource:               "Proses İçi Bellek Telemetrisi",
				DateFilter:               dateRange,
				DateFrom:                 dateFrom,
				DateTo:                   dateTo,
				Locale:                   adminLocaleFromRequest(r),
				ContentViews:             contentViewRows,
				AnalyticsEnabled:         modules["analytics"] && cfg.Analytics.NoCookieEnabled,
				SearchInsightsEnabled:    modules["search"] && modules["analytics"],
				CollectionsModuleEnabled: modules["collections"],
				NoCookieStats:            noCookieStats,
				UniqueVisitors:           uniqueVisitors,
			}); err != nil {
				http.Error(w, "admin reports unavailable", http.StatusInternalServerError)
			}
			return
		}
		securityNotice, noticeErr := userStore.LoginSecuritySummary(r.Context(), user.ID)
		if noticeErr != nil {
			http.Error(w, "Yönetim özeti şu anda açılamıyor.", http.StatusInternalServerError)
			return
		}
		previousLogin := ""
		if securityNotice.PreviousLoginAt != nil {
			previousLogin = securityNotice.PreviousLoginAt.Local().Format("02 Jan 2006 15:04 MST")
		}
		if err := ui.RenderAdminDashboard(w, ui.AdminDashboardData{User: user, Email: user.Email, Collections: activeStudioCollections(), Active: "dashboard", Locale: adminLocaleFromRequest(r), PushEnabled: modules["push"], EmailPreviewEnabled: smtpSender != nil, CommerceModuleEnabled: modules["commerce"], CollectionsModuleEnabled: modules["collections"], PreviousLogin: previousLogin, PreviousLoginIP: securityNotice.PreviousLoginIP, FailedLoginAttempts: securityNotice.FailedAttempts}); err != nil {
			http.Error(w, "admin dashboard unavailable", http.StatusInternalServerError)
		}
	})
	routes = append(routes, router.Route{Pattern: "GET /admin", Access: router.Protected, Handler: adminDashboardHandler})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports", Access: router.Protected, Handler: adminDashboardHandler})
	if modules["pages"] {
		auditExplorer := auditstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		routes = append(routes, router.Route{Pattern: "GET /admin/audit", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if !activePlatformAdmin(user) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			query := r.URL.Query()
			filter := auditstore.ExplorerFilter{Actor: query.Get("actor"), Action: query.Get("action"), Resource: query.Get("resource"), Target: query.Get("target"), Limit: 50}
			if raw := query.Get("offset"); raw != "" {
				var err error
				filter.Offset, err = strconv.Atoi(raw)
				if err != nil || filter.Offset < 0 || filter.Offset > 1_000_000 {
					http.Error(w, "invalid audit page", http.StatusBadRequest)
					return
				}
			}
			for _, value := range []string{filter.Actor, filter.Action, filter.Resource, filter.Target} {
				if len(value) > 256 {
					http.Error(w, "invalid audit filter", http.StatusBadRequest)
					return
				}
			}
			for _, pair := range []struct {
				input     string
				exclusive bool
			}{{"from", false}, {"until", true}} {
				if raw := query.Get(pair.input); raw != "" {
					day, err := time.Parse("2006-01-02", raw)
					if err != nil {
						http.Error(w, "invalid date filter", http.StatusBadRequest)
						return
					}
					if pair.exclusive {
						filter.Until = day.AddDate(0, 0, 1).UTC().Format("2006-01-02T15:04:05.000000000Z")
					} else {
						filter.From = day.UTC().Format("2006-01-02T15:04:05.000000000Z")
					}
				}
			}
			result, err := auditExplorer.Explore(r.Context(), filter)
			if err != nil {
				slog.Error("admin audit explorer unavailable", "error", err)
				http.Error(w, "Audit history is temporarily unavailable.", http.StatusInternalServerError)
				return
			}
			page := designsystem.AdminAuditPageProps{Actor: filter.Actor, Action: filter.Action, Resource: filter.Resource, Target: filter.Target, From: query.Get("from"), Until: query.Get("until"), Total: result.Total, Offset: filter.Offset, Limit: filter.Limit, Entries: make([]designsystem.AdminAuditEntry, 0, len(result.Events))}
			for _, event := range result.Events {
				at := event.CreatedAt
				legacy := at == "1970-01-01T00:00:00.000000000Z"
				if timestamp, parseErr := time.Parse("2006-01-02T15:04:05.000000000Z", at); parseErr == nil && !legacy {
					at = timestamp.Local().Format("02 Jan 2006 15:04:05 MST")
				} else if legacy {
					at = "Older event"
				}
				actor := event.ActorEmail
				if actor == "" {
					actor = "System"
					if event.ActorID != "" && !legacy {
						actor = "Unavailable user"
					}
				}
				targetLabel := ""
				if event.TargetID != "" {
					targetLabel = "Record"
				}
				page.Entries = append(page.Entries, designsystem.AdminAuditEntry{At: at, Actor: actor, Action: event.Action, Resource: event.ResourceType, Target: targetLabel, RequestID: event.RequestID, BeforeJSON: event.BeforeJSON, AfterJSON: event.AfterJSON, Legacy: legacy})
			}
			if err := ui.RenderAdminAudit(w, adminLocaleFromRequest(r), page); err != nil {
				slog.Error("admin audit page render failed", "error", err)
				http.Error(w, "Audit history is temporarily unavailable.", http.StatusInternalServerError)
			}
		})})
	}
	backupConfirmationLimiter := newRateLimiter(3, time.Minute)
	adminBackupHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodGet {
			recent := make([]designsystem.AdminBackupItem, 0, 12)
			publicKey := ""
			if signingKey, err := backupstore.ParseSigningSeed(cfg.Backup.SigningSeed); err == nil {
				publicKey = hex.EncodeToString(signingKey[32:])
			}
			if cfg.Backup.Enabled {
				var snapshots []backupstore.SnapshotSummary
				dispatcher, err := backupstore.NewS3Dispatcher(backupstore.S3Config{Endpoint: cfg.Media.S3Endpoint, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Prefix: cfg.Backup.Prefix, Secure: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle})
				if err == nil {
					listCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
					snapshots, err = dispatcher.ListRecent(listCtx, 12)
					cancel()
				}
				if err != nil {
					slog.Error("admin backup history unavailable", "error", err)
				} else {
					for _, snapshot := range snapshots {
						recent = append(recent, designsystem.AdminBackupItem{Name: snapshot.Name, Created: snapshot.CreatedAt.Local().Format("02 Jan 2006 15:04 MST"), Size: fmt.Sprintf("%.1f MB", float64(snapshot.Bytes)/(1024*1024))})
					}
				}
			}
			if err := ui.RenderAdminBackup(w, security.CSRFTokenFromContext(r.Context()), adminLocaleFromRequest(r), recent, cfg.Backup.Enabled, publicKey); err != nil {
				http.Error(w, "backup page unavailable", http.StatusInternalServerError)
			}
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		if !security.AllowForRequest(backupConfirmationLimiter, r.Context(), "admin-backup:"+user.ID) {
			http.Error(w, "Please wait before trying again.", http.StatusTooManyRequests)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid backup request", http.StatusBadRequest)
			return
		}
		signingKey, err := backupstore.ParseSigningSeed(cfg.Backup.SigningSeed)
		if err != nil {
			http.Error(w, "Backup signing is not configured.", http.StatusServiceUnavailable)
			return
		}
		confirmed, err := authSvc.Authenticate(r.Context(), user.Email, r.FormValue("password"))
		if err != nil || confirmed.ID != user.ID || confirmed.Role != auth.RoleAdmin {
			http.Error(w, "Password confirmation failed.", http.StatusForbidden)
			return
		}
		workDir, err := os.MkdirTemp("", "hydrogo-admin-backup-")
		if err != nil {
			http.Error(w, "Backup is temporarily unavailable.", http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(workDir)
		snapshotDir := filepath.Join(workDir, "snapshot")
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
		defer cancel()
		switch cfg.Database.Dialect {
		case database.SQLite:
			_, err = backupstore.SnapshotSQLite(ctx, db, cfg.Media.Dir, snapshotDir)
		case database.Postgres:
			var tools backupstore.PostgresTools
			tools, err = backupstore.LocatePostgresTools()
			if err == nil {
				_, err = backupstore.SnapshotPostgres(ctx, tools, cfg.Database.DSN, cfg.Media.Dir, snapshotDir)
			}
		case database.MySQL:
			var tools backupstore.MySQLTools
			tools, err = backupstore.LocateMySQLTools()
			if err == nil {
				_, err = backupstore.SnapshotMySQL(ctx, tools, cfg.Database.DSN, cfg.Media.Dir, snapshotDir)
			}
		default:
			err = errors.New("unsupported backup database")
		}
		if err != nil {
			slog.Error("admin backup snapshot failed", "error", err)
			http.Error(w, "Backup could not be created.", http.StatusInternalServerError)
			return
		}
		if auditRecorder != nil {
			if err := auditRecorder.Record(r.Context(), "backup.downloaded", user.ID); err != nil {
				slog.Error("admin backup audit failed", "error", err)
				http.Error(w, "Backup could not be created.", http.StatusInternalServerError)
				return
			}
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="hydrogo-backup-`+time.Now().UTC().Format("20060102T150405Z")+`.tar.gz"`)
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := backupstore.WriteSignedSnapshotArchive(ctx, w, snapshotDir, signingKey); err != nil {
			slog.Error("admin backup archive stream failed", "error", err)
		}
	})
	routes = append(routes, adminBackupRoutes(adminBackupHandler)...)
	routes = append(routes, router.Route{Pattern: "POST /admin/account/locale", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		locale := strings.TrimSpace(r.FormValue("locale"))
		if !i18n.IsSupportedCatalogLocale(locale) {
			http.Error(w, "unsupported locale", http.StatusBadRequest)
			return
		}
		setAdminLocaleCookie(w, locale, cfg.Security.SecureCookies)
		http.Redirect(w, r, "/admin/account?locale_saved=1", http.StatusSeeOther)
	})})

	adminReportsExportHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Query().Get("range") == "custom" {
			if _, _, valid := customAnalyticsDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to")); !valid {
				http.Error(w, "Choose a valid date range within the past year.", http.StatusBadRequest)
				return
			}
		}
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "csv"
		}
		var paths []observability.PathStatsSnapshot
		if runtimeMetrics != nil {
			paths = runtimeMetrics.PathStats()
		}
		if format == "json" {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"hydro-report-%s.json\"", time.Now().Format("20060102-150405")))
			var stats members.MemberStats
			if modules["members"] {
				stats, _ = memberService.Stats(r.Context())
			}
			totalReq := uint64(0)
			totalViews := uint64(0)
			totalErrors := uint64(0)
			avgLatency := uint64(0)
			uniqueVisitors := int64(0)
			var noCookieStats []ui.NoCookieAnalyticsRow
			var startedAt time.Time
			if runtimeMetrics != nil {
				totalReq = runtimeMetrics.TotalRequests()
				totalViews = runtimeMetrics.TotalViews()
				totalErrors = runtimeMetrics.TotalErrors()
				avgLatency = runtimeMetrics.AvgLatencyMS()
				startedAt = runtimeMetrics.StartedAt()
			}
			if modules["analytics"] && cfg.Analytics.NoCookieEnabled {
				tenantID := ""
				if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
					tenantID = scope.TenantID
				}
				noCookieStats, uniqueVisitors, _ = dailyNoCookieAnalytics(r.Context(), analyticsStore, tenantID, time.Now().UTC().Format("2006-01-02"))
			}
			out := map[string]any{
				"timestamp":       time.Now().UTC().Format(time.RFC3339),
				"source":          "HydroGo in-memory telemetry",
				"process_started": startedAt.Format(time.RFC3339),
				"summary": map[string]any{
					"total_requests":   totalReq,
					"total_views":      totalViews,
					"total_errors":     totalErrors,
					"avg_latency_ms":   avgLatency,
					"unique_visitors":  uniqueVisitors,
					"total_members":    stats.TotalMembers,
					"active_members":   stats.ActiveMembers,
					"verified_members": stats.VerifiedMembers,
				},
				"paths": paths,
				"dimensions": func() []observability.DimensionStatsSnapshot {
					if runtimeMetrics == nil {
						return nil
					}
					return runtimeMetrics.DimensionStats()
				}(),
				"content_views": func() []analyticsstore.ContentViewAggregate {
					if !modules["analytics"] {
						return nil
					}
					rows, _ := analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 500))
					return rows
				}(),
				"analytics_daily": noCookieStats,
			}
			_ = json.NewEncoder(w).Encode(out)
			return
		}

		// CSV format
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"hydro-report-%s.csv\"", time.Now().Format("20060102-150405")))
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"Path", "Views", "Requests", "Errors", "AvgLatencyMS", "P50MS", "P95MS", "P99MS", "BytesIn", "BytesOut", "Status2xx", "Status3xx", "Status4xx", "Status5xx", "Timeouts", "RateLimited"})
		for _, p := range paths {
			_ = writer.Write([]string{p.Path, strconv.FormatUint(p.Views, 10), strconv.FormatUint(p.Requests, 10), strconv.FormatUint(p.Errors, 10), strconv.FormatUint(p.AvgMS, 10), strconv.FormatUint(p.P50MS, 10), strconv.FormatUint(p.P95MS, 10), strconv.FormatUint(p.P99MS, 10), strconv.FormatUint(p.BytesIn, 10), strconv.FormatUint(p.BytesOut, 10), strconv.FormatUint(p.Status2xx, 10), strconv.FormatUint(p.Status3xx, 10), strconv.FormatUint(p.Status4xx, 10), strconv.FormatUint(p.Status5xx, 10), strconv.FormatUint(p.Timeouts, 10), strconv.FormatUint(p.RateLimited, 10)})
		}
		if runtimeMetrics != nil {
			_ = writer.Write([]string{})
			_ = writer.Write([]string{"Dimensions", "TenantID", "Module", "Requests", "Errors", "AvgLatencyMS"})
			for _, d := range runtimeMetrics.DimensionStats() {
				_ = writer.Write([]string{"", d.TenantID, d.Module, strconv.FormatUint(d.Requests, 10), strconv.FormatUint(d.Errors, 10), strconv.FormatUint(d.AvgMS, 10)})
			}
		}
		if modules["analytics"] {
			if rows, err := analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 500)); err == nil && len(rows) > 0 {
				_ = writer.Write([]string{})
				_ = writer.Write([]string{"ContentViews", "Day", "TenantID", "ContentType", "ContentID", "Locale", "ViewCount"})
				for _, row := range rows {
					_ = writer.Write([]string{"", row.Day, row.TenantID, row.ContentType, row.ContentID, row.Locale, strconv.FormatInt(row.ViewCount, 10)})
				}
			}
		}
		if modules["analytics"] && cfg.Analytics.NoCookieEnabled {
			tenantID := ""
			if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
				tenantID = scope.TenantID
			}
			if rows, err := analyticsStore.ListDailyStats(r.Context(), time.Now().UTC().Format("2006-01-02"), tenantID, 500); err == nil && len(rows) > 0 {
				_ = writer.Write([]string{})
				_ = writer.Write([]string{"DailyAnalytics", "Metric", "Dimension", "EventCount"})
				for _, row := range rows {
					_ = writer.Write([]string{"", row.Metric, row.Dimension, strconv.FormatInt(row.EventCount, 10)})
				}
			}
		}
		writer.Flush()
	})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports/export", Access: router.Protected, Handler: adminReportsExportHandler})

	adminControlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !adminControlAllowed(user, r.URL.Path) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		active := "control"
		if r.URL.Path == "/admin/site" {
			active = "site"
		}
		if r.URL.Path == "/admin/media" {
			active = "media"
		}
		if r.URL.Path == "/admin/security" {
			active = "security"
		}
		if r.URL.Path == "/admin/operations" {
			active = "operations"
		}
		if r.URL.Path == "/admin/developer" {
			if err := ui.RenderAdminArea(w, ui.AdminAreaData{Request: r, Email: user.Email, CSRF: security.CSRFTokenFromContext(r.Context()), Area: "developer", Locale: adminLocaleFromRequest(r)}); err != nil {
				http.Error(w, "developer tools unavailable", http.StatusInternalServerError)
			}
			return
		}
		if r.URL.Path == "/admin/developer/resources" {
			if err := ui.RenderAdminResourceUsage(w, ui.AdminResourceUsageData{Email: user.Email, Snapshot: runtimeMetrics.Snapshot(), Locale: adminLocaleFromRequest(r)}); err != nil {
				http.Error(w, "resource usage unavailable", http.StatusInternalServerError)
			}
			return
		}
		if r.URL.Path == "/admin/developer/components" {
			http.Redirect(w, r, "/admin/developer", http.StatusMovedPermanently)
			return
		}
		if r.URL.Path == "/admin/account" {
			active = "account"
		}
		if r.URL.Path == "/admin/users" {
			active = "users"
		}
		if r.URL.Path == "/admin/permissions" {
			active = "permissions"
		}
		var err error
		if active == "control" {
			err = ui.RenderAdminControl(w, ui.AdminControlData{Email: user.Email, CSRF: security.CSRFTokenFromContext(r.Context()), Active: active, Locale: adminLocaleFromRequest(r)})
		} else {
			scope, _ := tenant.FromContext(r.Context())
			areaData := ui.AdminAreaData{Request: r, Email: user.Email, CSRF: security.CSRFTokenFromContext(r.Context()), Area: active, Locale: adminLocaleFromRequest(r), TenantID: scope.TenantID, LocaleSaved: r.URL.Query().Get("locale_saved") == "1", SoundAlertsEnabled: modules["realtime"] && modules["forms"], FontToolsEnabled: modules["pages"], EmailVerificationEnabled: smtpSender != nil, PasswordResetLinkEnabled: recoveryService != nil && cfg.HTTP.PublicOrigin != "", Permissions: authSvc.AvailablePermissions()}
			if active == "account" {
				areaData.MFAEnabled, err = mfaService.Enabled(r.Context(), db, user.ID)
				areaData.MFAStatusLoaded = err == nil
				keySize := len([]byte(cfg.Security.MFAMasterKey))
				areaData.MFAAvailable = areaData.MFAStatusLoaded && (keySize == 16 || keySize == 24 || keySize == 32)
				err = nil // Keep account settings available if the MFA status store is temporarily unavailable.
			}
			if active == "users" {
				if repo, ok := any(userStore).(auth.UserAdminRepository); ok {
					areaData.Users, err = repo.List(r.Context())
				} else {
					err = fmt.Errorf("admin user repository unavailable")
				}
			}
			if active == "security" {
				entries, listErr := banStore.ListActive(r.Context(), db, time.Now().UTC(), 4096)
				if listErr != nil {
					err = listErr
				} else {
					areaData.IPBans = make([]ui.SecurityBanRow, 0, len(entries))
					for _, entry := range entries {
						expires := "Süresiz"
						if !entry.Expires.IsZero() {
							expires = entry.Expires.UTC().Format(time.RFC3339)
						}
						areaData.IPBans = append(areaData.IPBans, ui.SecurityBanRow{Address: entry.Prefix.String(), Expires: expires})
					}
				}
			}
			if err == nil {
				if active == "account" {
					var page bytes.Buffer
					err = ui.RenderAdminArea(&page, areaData)
					if err == nil {
						w.Header().Set("Cache-Control", "private, no-store")
						w.Header().Set("X-Hydro-Custom-CSP", security.InlineCSPDirectives(page.Bytes()))
						_, err = w.Write(page.Bytes())
					}
				} else {
					err = ui.RenderAdminArea(w, areaData)
				}
			}
		}
		if err != nil {
			http.Error(w, "admin control center unavailable", http.StatusInternalServerError)
		}
	})
	for _, pattern := range adminControlRoutePatterns(modules) {
		routes = append(routes, router.Route{Pattern: pattern, Access: router.Protected, Handler: adminControlHandler})
	}
	adminBanMutation := func(remove bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if !activePlatformAdmin(user) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid IP rule", http.StatusBadRequest)
				return
			}
			prefix, err := parseIPBanPrefix(r.FormValue("address"))
			if err != nil {
				http.Error(w, "invalid IP address or CIDR", http.StatusBadRequest)
				return
			}
			if remove {
				err = banStore.Remove(r.Context(), db, prefix)
				if err == nil {
					err = ipBlocklist.Unban(prefix.String())
				}
			} else {
				var expires time.Time
				rawDuration := strings.TrimSpace(r.FormValue("duration"))
				if rawDuration != "" {
					duration, parseErr := time.ParseDuration(rawDuration)
					if parseErr != nil || duration < 0 || duration > 365*24*time.Hour {
						http.Error(w, "invalid ban duration", http.StatusBadRequest)
						return
					}
					if duration > 0 {
						expires = time.Now().UTC().Add(duration)
					}
				}
				err = banStore.Put(r.Context(), db, prefix, expires)
				if err == nil {
					err = ipBlocklist.Ban(prefix.String(), expires)
				}
			}
			if err != nil {
				http.Error(w, "IP rule could not be updated", http.StatusServiceUnavailable)
				return
			}
			action := "security.ip_ban.created"
			if remove {
				action = "security.ip_ban.removed"
			}
			if authHTTP.Audit != nil {
				_ = authHTTP.Audit.Record(r.Context(), action, prefix.String())
			}
			http.Redirect(w, r, "/admin/security", http.StatusSeeOther)
		}
	}
	routes = append(routes, adminSecurityBanRoutes(adminBanMutation(false), adminBanMutation(true))...)
	if modules["media"] {
		routes = append(routes, mediaRouterRoutes(mediaHTTP.AdminRoutes())...)
		routes = append(routes, mediaRouterRoutes(mediaHTTP.Routes())...)
	}
	if modules["site"] {
		siteRoutes := siteRouterRoutes(siteHTTP.Routes())
		for i := range siteRoutes {
			if !strings.HasPrefix(siteRoutes[i].Pattern, "GET ") {
				original := siteRoutes[i].Handler
				siteRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					original.ServeHTTP(w, r)
					if publicHTMLCache != nil {
						publicHTMLCache.PurgeTag("pages")
						publicHTMLCache.PurgeTag("news")
					}
					if staticCacheDir != "" {
						_ = server.PurgeStaticDiskCache(staticCacheDir)
					}
				})
			}
		}
		routes = append(routes, siteRoutes...)
	}
	if modules["news"] {
		newsRoutes := newsRouterRoutes(newsHTTP.Routes())
		for i := range newsRoutes {
			if strings.HasPrefix(newsRoutes[i].Pattern, "GET /public/news/") {
				if modules["analytics"] {
					original := newsRoutes[i].Handler
					tracked := analytics.NoCookieMiddleware(analytics.ContentViewMiddleware(original, contentViewQueue, "news"), noCookieQueue, noCookieHasher)
					newsRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get("preview_token") != "" {
							original.ServeHTTP(w, r)
							return
						}
						tracked.ServeHTTP(w, r)
					})
				}
			} else if !strings.HasPrefix(newsRoutes[i].Pattern, "GET ") {
				original := newsRoutes[i].Handler
				newsRoutes[i].Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if pageLLMSCache != nil {
						defer pageLLMSCache.Invalidate()
					}
					if staticCacheDir != "" {
						defer server.PurgeStaticDiskCache(staticCacheDir)
					}
					if publicHTMLCache != nil {
						defer publicHTMLCache.PurgeTag("news")
					}
					original.ServeHTTP(w, r)
				})
			}
		}
		routes = append(routes, newsRoutes...)
	}
	if modules["pages"] || modules["forms"] || modules["news"] {
		routes = append(routes, router.Route{Pattern: "GET /designsystem/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(designsystem.ServeAsset)})
	}
	if modules["pages"] || modules["news"] {
		routes = append(routes, router.Route{Pattern: "GET /csrf-token", Access: router.Public, Handler: http.HandlerFunc(security.TokenHandler)})
	}
	if modules["pages"] {
		fontsDir := cfg.SEO.FontsDir
		if fontsDir == "" {
			fontsDir = "public/fonts"
		}
		routes = append(routes, router.Route{Pattern: "GET /fonts/{path...}", Access: router.Public, Handler: designsystem.FontAssets(fontsDir)})
	}
	if modules["forms"] {
		routes = append(routes, router.Route{Pattern: "GET /forms/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(formsruntime.ServeAsset)})
	}
	if selectedTheme != nil {
		routes = append(routes, router.Route{Pattern: "GET /theme/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(selectedTheme.ServeAsset)})
	}
	// Local editor assets are embedded in the release binary. Uploaded media and
	// themes remain external persistent resources by design.
	vendorFS, err := fs.Sub(staticassets.FS, "vendor")
	if err != nil {
		return fmt.Errorf("prepare embedded vendor assets: %w", err)
	}
	routes = append(routes, router.Route{Pattern: "GET /vendor/{path...}", Access: router.Public, Handler: http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS)))}, adminAssetsRoute())

	metrics := observability.NewMetrics()
	runtimeMetrics = metrics
	metrics.SetWorkerConcurrency(budget.WorkerConcurrency)
	newsWorker.OnRetry = metrics.RecordWorkerRetry
	newsWorker.OnDeadLetter = metrics.RecordWorkerDeadLetter
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
	if !*cgiMode {
		go runSessionCleanupObserved(ctx, sessionStore, stderr, metrics)
	}
	if modules["media"] && !*cgiMode {
		go runStaleMediaTempSweeper(ctx, cfg.Media.Dir, stderr)
	}
	if modules["forms"] && smtpSender != nil && !*cgiMode && !cfg.SafeMode {
		if modules["pages"] {
			mailWorker.OnSchedule = func(jobCtx context.Context, now time.Time) error {
				_, err := runAdminWeeklyDigestOnce(jobCtx, now, db, transactioner, adminstudiosql.Store{Dialect: cfg.Database.Dialect}, userStore, spoolStore, tenantScopes, requireTenantScope, modules["analytics"], modules["members"])
				return err
			}
		}
		go mailWorker.Run(ctx, 500*time.Millisecond)
	}
	if modules["news"] && !*cgiMode && !cfg.SafeMode {
		go runNewsWorker(ctx, db, newsWorker, metrics, stderr, budget.WorkerConcurrency)
	}
	webhookEndpoints := make([]webhooks.Endpoint, 0, len(cfg.Webhooks))
	if !cfg.SafeMode {
		for _, configured := range cfg.Webhooks {
			topics := make(map[string]struct{}, len(configured.Topics))
			for _, topic := range configured.Topics {
				topics[strings.TrimSpace(topic)] = struct{}{}
			}
			webhookEndpoints = append(webhookEndpoints, webhooks.Endpoint{ID: configured.ID, URL: configured.URL, Secret: configured.Secret, Topics: topics})
		}
	}
	if len(webhookEndpoints) > 0 {
		routes = append(routes, router.Route{Pattern: "GET /admin/webhooks/test", Access: router.Protected, Handler: webhooks.TestHandler{Dispatcher: webhooks.Dispatcher{}, Endpoints: webhookEndpoints}}, router.Route{Pattern: "POST /admin/api/webhooks/test", Access: router.Protected, Handler: webhooks.TestHandler{Dispatcher: webhooks.Dispatcher{}, Endpoints: webhookEndpoints}})
	}
	if !cfg.SafeMode && !*cgiMode && (modules["pages"] || modules["news"] || modules["commerce"]) {
		go runOutboxWorker(ctx, webhooks.Worker{DB: db, Store: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, TenantScopes: tenantScopes, Dispatcher: webhooks.Dispatcher{Endpoints: webhookEndpoints}, WorkerID: "outbox-worker-0", Lease: time.Minute, RetryAfter: time.Minute}, stderr)
	}
	logWriter := stderr
	if logWriter == nil {
		logWriter = io.Discard
	}
	var logger *slog.Logger
	if os.Getenv("HYDRO_DEV_MODE") == "1" {
		logger = slog.New(observability.DevelopmentHandler(logWriter))
	} else {
		logger = slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	if !cfg.SafeMode && !*cgiMode && modules["site"] && linkScanManager != nil && linkScanOrigin != "" {
		tenantKeys := []string{"default"}
		if requireTenantScope {
			tenantKeys = make([]string, 0, len(tenantScopes))
			for _, scope := range tenantScopes {
				tenantKeys = append(tenantKeys, scope.TenantID)
			}
		}
		if len(tenantKeys) > 0 {
			go linkcheck.RunMonthly(ctx, linkScanManager, tenantKeys, linkScanOrigin, func(err error) { logger.WarnContext(ctx, "scheduled link audit could not start", "error", err) })
		}
	}
	if !*cgiMode && (cfg.Database.Dialect == database.SQLite || cfg.Database.Dialect == database.Postgres) {
		go database.RunWeeklyMaintenance(ctx, func(maintenanceCtx context.Context) {
			maintenanceCtx, cancel := context.WithTimeout(maintenanceCtx, 30*time.Minute)
			defer cancel()
			maintenanceDB, openErr := database.OpenWithOptions(maintenanceCtx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
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
				if requireTenantScope {
					for _, scope := range tenantScopes {
						if auditErr := auditRecorder.Record(tenant.WithScope(maintenanceCtx, scope), "database.maintenance", target); auditErr != nil {
							logger.ErrorContext(maintenanceCtx, "database maintenance audit failed", "tenant", scope.TenantID, "error", auditErr)
						}
					}
				} else if auditErr := auditRecorder.Record(maintenanceCtx, "database.maintenance", target); auditErr != nil {
					logger.ErrorContext(maintenanceCtx, "database maintenance audit failed", "error", auditErr)
				}
			}
		})
	}
	var panicReporter *observability.SentryReporter
	if cfg.Observability.SentryDSN != "" {
		panicReporter, err = observability.NewSentryReporter(cfg.Observability.SentryDSN, nil, 64)
		if err != nil {
			return fmt.Errorf("Sentry configuration: %w", err)
		}
		defer panicReporter.Close()
	}
	adminScriptHashes, err := ui.AdminScriptHashes()
	if err != nil {
		return fmt.Errorf("Admin script CSP catalog: %w", err)
	}
	adminScriptHashes = append(adminScriptHashes, media.AdminScriptHashes()...)
	if modules["news"] {
		adminScriptHashes = append(adminScriptHashes, newstransport.AdminScriptHashes()...)
	}
	serverConfig := server.Config{
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
		RealtimeHub:     realtimeHub,
		PublicHTMLCache: publicHTMLCache,
		ShutdownTimeout: budget.ShutdownTimeout,
		AuthSvc:         authSvc,
		Metrics:         metrics,
		Logger:          logger,
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
		IPBlocklist:          ipBlocklist,
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
		TenantResolver: tenantResolver,
		Routes:         routes,
		Ready: func(checkCtx context.Context) error {
			if err := db.PingContext(checkCtx); err != nil {
				return err
			}
			return database.CheckCompatibilityForModules(checkCtx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
		},
	}
	if *cgiMode {
		handler := server.NewHandler(serverConfig)
		handler = server.StaticDiskCacheMiddleware(server.StaticCacheOptions{
			BaseDir:       staticCacheDir,
			SessionCookie: auth.SessionCookieName,
		})(handler)

		// 2. Comprehensive Lazy Cron runner for CGI request lifecycles
		handler = server.LazyCron(handler, func(cronCtx context.Context) {
			// Bounded cron context
			cronCtx, cancel := context.WithTimeout(cronCtx, 5*time.Second)
			defer cancel()

			// A. Scheduled news publications
			if modules["news"] && !cfg.SafeMode {
				published := false
				if err := newsWorker.EnqueueDue(cronCtx, db, time.Now().UTC()); err == nil {
					for i := 0; i < 4; i++ {
						worked, err := newsWorker.RunOnce(cronCtx, "cgi-lazy-cron", time.Now().UTC())
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
					Store:        outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
					TenantScopes: tenantScopes,
					Dispatcher:   webhooks.Dispatcher{Endpoints: webhookEndpoints},
					WorkerID:     "cgi-lazy-cron",
					Lease:        10 * time.Second,
					RetryAfter:   30 * time.Second,
				}
				_, _ = outboxWorker.RunOnce(cronCtx, time.Now().UTC())
			}
			if !cfg.SafeMode && modules["forms"] && modules["pages"] && smtpSender != nil {
				_, _ = runAdminWeeklyDigestOnce(cronCtx, time.Now(), db, transactioner, adminstudiosql.Store{Dialect: cfg.Database.Dialect}, userStore, spoolStore, tenantScopes, requireTenantScope, modules["analytics"], modules["members"])
				_, _ = mailWorker.RunOnce(cronCtx)
			}
		})
		return server.ServeCGI(handler)
	}
	if smtpSender != nil && modules["auth"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/mailer/test-preview", Access: router.Protected, Handler: mailer.TestPreviewHandler{Sender: smtpSender}}, router.Route{Pattern: "POST /admin/api/mailer/test-preview", Access: router.Protected, Handler: mailer.TestPreviewHandler{Sender: smtpSender}})
		brandingHandler := mailer.EmailBrandingHandler{Settings: mailstore.EmailBrandingStore{DB: db, Dialect: cfg.Database.Dialect}, Audit: auditRecorder}
		routes = append(routes, router.Route{Pattern: "GET /admin/mailer/branding", Access: router.Protected, Handler: brandingHandler}, router.Route{Pattern: "POST /admin/mailer/branding", Access: router.Protected, Handler: brandingHandler})
		verifyNotifier := auth.EmailVerificationNotifierFunc(func(notifyCtx context.Context, email, link string) error {
			return (mailer.NotificationSender{Sender: smtpSender}).SendEmailVerification(notifyCtx, email, link)
		})
		verificationHandler := auth.EmailVerificationHandler{Service: auth.EmailVerificationService{Users: userStore, Tokens: userStore}, Transactions: transactioner, Notifier: verifyNotifier, PublicOrigin: cfg.HTTP.PublicOrigin, Limiter: newRateLimiter(3, time.Hour), AdminUsers: userStore, Audit: auditRecorder}
		routes = append(routes, verificationHandler.Routes()...)
	}
	if modules["auth"] && strings.TrimSpace(cfg.Mail.From) != "" {
		routes = append(routes, router.Route{Pattern: "GET /admin/mailer/dns-diagnostics", Access: router.Protected, Handler: mailer.DNSDiagnosticsHandler{From: cfg.Mail.From, Selector: cfg.Mail.DKIMSelector}})
	}
	if *fastCGIMode {
		return server.ServeFastCGI(ctx, "tcp", *fastCGIAddress, serverConfig)
	}
	return server.Serve(ctx, serverConfig)
}

/*

 */

type expiredSessionCleaner interface {
	DeleteExpired(context.Context, time.Time) (int64, error)
}

type sessionCleanupObserver interface {
	RecordSessionCleanup(int64)
	RecordSessionCleanupError()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func profileDatabaseOptions(budget capacity.Budget, profile capacity.Profile) database.OpenOptions {
	sqliteProfile := database.SQLiteCompactProfile
	if profile != capacity.Compact {
		sqliteProfile = database.SQLiteVPSProfile
	}
	return database.OpenOptions{
		MaxOpenConns:  budget.DBMaxOpenConns,
		MaxIdleConns:  budget.DBMaxIdleConns,
		SQLiteProfile: sqliteProfile,
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

/*

 */

func moduleEnabled(modules []string, wanted string) bool {
	wanted = strings.ToLower(strings.TrimSpace(wanted))
	for _, module := range modules {
		if strings.ToLower(strings.TrimSpace(module)) == wanted {
			return true
		}
	}
	return false
}

func demoFrontendEnabled() bool {
	if strings.TrimSpace(os.Getenv("HYDRO_DEV_MODE")) == "1" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("HYDRO_ENABLE_DEMO_FRONTEND"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func enabledModuleSet(modules []string) map[string]bool {
	result := make(map[string]bool, len(modules))
	for _, module := range modules {
		name := strings.ToLower(strings.TrimSpace(module))
		if name != "" {
			result[name] = true
		}
	}
	return result
}

func adminControlRoutePatterns(enabled map[string]bool) []string {
	patterns := []string{
		"GET /admin/control",
		"GET /admin/security",
		"GET /admin/operations",
		"GET /admin/account",
		"GET /admin/users",
		"GET /admin/permissions",
		"GET /admin/developer",
		"GET /admin/developer/resources",
	}
	if enabled["site"] {
		patterns = append(patterns, "GET /admin/site")
	}
	return patterns
}

func adminControlAllowed(user *auth.User, path string) bool {
	if path == "/admin/account" {
		return user != nil && user.Active
	}
	if path == "/admin/site" {
		return auth.Can(user, auth.PermissionSiteManage)
	}
	if !activePlatformAdmin(user) {
		return false
	}
	switch path {
	case "/admin/users", "/admin/permissions":
		return auth.Can(user, auth.PermissionUsersManage)
	case "/admin/operations":
		return auth.Can(user, auth.PermissionOperations)
	}
	return true
}

func newAuditRecorder(db *sql.DB, dialect database.Dialect, requireTenantScope bool) auth.AuditRecorder {
	store := auditstore.Store{Dialect: dialect, RequireTenantScope: requireTenantScope}
	return auth.AuditRecorderFunc(func(ctx context.Context, action, targetID string) error {
		return database.NewTenantTransactor(db, dialect, requireTenantScope).Within(ctx, func(tx database.Executor) error {
			return store.Record(ctx, tx, newAuditID(), action, targetID)
		})
	})
}

func configuredMFARoles(values []string) []auth.Role {
	roles := make([]auth.Role, 0, len(values))
	for _, value := range values {
		roles = append(roles, auth.Role(value))
	}
	return roles
}

func newAuditID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("audit-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

type demoNewsSeed struct {
	id, slug, title, excerpt, takeaways, category, author, coverMediaID, tags, body string
	featured                                                                        bool
}

const (
	maxReleaseManifestBytes = 1 << 20
	maxTrustedKeyFileBytes  = 4 << 10
)

type runtimeFlags struct {
	capacityProfile *string
	configPath      *string
	address         *string
	databaseDialect *string
	databaseDSN     *string
	safeMode        *bool
}

func addRuntimeFlags(flags *flag.FlagSet) runtimeFlags {
	return runtimeFlags{
		capacityProfile: flags.String("capacity-profile", "", "compact, standard or distributed"),
		configPath:      flags.String("config", "", "YAML config file"),
		address:         flags.String("address", "", "HTTP listen address"),
		databaseDialect: flags.String("database-dialect", "", "sqlite or postgres"),
		databaseDSN:     flags.String("database-dsn", "", "database connection string"),
		safeMode:        flags.Bool("safe-mode", false, "suppress external integrations and non-essential workers"),
	}
}

func loadRuntimeConfig(flags *flag.FlagSet, values runtimeFlags) (config.Config, error) {
	set := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) { set[item.Name] = true })
	overrides := config.Overrides{}
	if set["capacity-profile"] {
		overrides.CapacityProfile = values.capacityProfile
	}
	if set["address"] {
		overrides.HTTPAddress = values.address
	}
	if set["database-dialect"] {
		overrides.DatabaseDialect = values.databaseDialect
	}
	if set["database-dsn"] {
		overrides.DatabaseDSN = values.databaseDSN
	}
	if set["safe-mode"] {
		overrides.SafeMode = values.safeMode
	}
	return config.Load(*values.configPath, os.LookupEnv, overrides)
}

func configuredSocialOAuth(db *sql.DB, dialect database.Dialect, service *auth.AuthService, audit auth.AuditRecorder, secure, requireTenantScope bool) (*auth.SocialOAuthHandler, error) {
	providers := make(map[string]auth.OAuthProviderConfig)
	for _, name := range []string{"google", "github", "apple"} {
		prefix := "HYDRO_OAUTH_" + strings.ToUpper(name) + "_"
		clientID, secret, redirect := os.Getenv(prefix+"CLIENT_ID"), os.Getenv(prefix+"CLIENT_SECRET"), os.Getenv(prefix+"REDIRECT_URL")
		if clientID == "" && secret == "" && redirect == "" {
			continue
		}
		if clientID == "" || secret == "" || redirect == "" {
			return nil, fmt.Errorf("%s requires CLIENT_ID, CLIENT_SECRET and REDIRECT_URL", prefix)
		}
		provider, err := auth.DefaultOAuthProvider(name, clientID, secret, redirect)
		if err != nil {
			return nil, err
		}
		providers[name] = provider
	}
	if len(providers) == 0 {
		return nil, nil
	}
	key := []byte(os.Getenv("HYDRO_OAUTH_STATE_KEY"))
	if len(key) < 32 {
		return nil, errors.New("HYDRO_OAUTH_STATE_KEY must contain at least 32 bytes when a provider is enabled")
	}
	resolver := authstore.OAuthIdentityStore{DB: db, Dialect: dialect, RequireTenantScope: requireTenantScope}
	return &auth.SocialOAuthHandler{Providers: providers, Resolver: resolver, Service: service, Audit: audit, StateKey: key, Secure: secure}, nil
}

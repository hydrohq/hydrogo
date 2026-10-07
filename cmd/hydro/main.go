package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	collectiontransport "github.com/ardaozyildirim/hydrogo/internal/collections/transport"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	formsruntime "github.com/ardaozyildirim/hydrogo/internal/forms"
	formstore "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/generator"
	"github.com/ardaozyildirim/hydrogo/internal/geolocation"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/licensing"
	"github.com/ardaozyildirim/hydrogo/internal/linkcheck"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
	"github.com/ardaozyildirim/hydrogo/internal/portable"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/projectgen"
	pushservice "github.com/ardaozyildirim/hydrogo/internal/push"
	pushstore "github.com/ardaozyildirim/hydrogo/internal/push/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/release"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/sdkdist"
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
	case "generate":
		return generateCommand(args[1:], stdout, stderr)
	case "module":
		return moduleCommand(args[1:], stdout, stderr)
	case "resource":
		return resourceCommand(args[1:], stdout, stderr)
	case "collections":
		return collectionsCommand(args[1:], stdout, stderr)
	case "api":
		return apiCommand(args[1:], stdout, stderr)
	case "frontend":
		return frontendCommand(args[1:], stdout, stderr)
	case "schema":
		return schemaCommand(args[1:], stdout, stderr)
	case "db":
		return dbCommand(args[1:], stdout, stderr)
	case "permissions":
		return permissionsCommand(args[1:], stdout, stderr)
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
	case "license":
		return licenseCommand(args[1:], stdout, stderr)
	case "extension":
		return extensionCommand(args[1:], stdout, stderr)
	case "new":
		if len(args) > 1 && args[1] == "resource" {
			return interactiveResourceCommand(args[2:], stdin, stdout, stderr)
		}
		return newProjectCommand(args[1:], stdin, stdout, stderr)
	case "project":
		return projectCommand(args[1:], stdin, stdout, stderr)
	case "check":
		return projectCheckCommand(args[1:], stdout, stderr)
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

func runIconPack(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("icons", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourcePath := flags.String("source", "", "master PNG, JPEG, GIF or WebP logo")
	outputDir := flags.String("output", "", "directory for favicon and web app icons")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourcePath == "" || *outputDir == "" || flags.NArg() != 0 {
		return errors.New("icons requires --source and --output")
	}
	file, err := os.Open(*sourcePath)
	if err != nil {
		return errors.New("icon source could not be opened")
	}
	defer file.Close()
	if err := media.GenerateIconPack(file, *outputDir); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "icon-pack-generated output=%s\n", *outputDir)
	return err
}

func fontsCommand(args []string, stdout io.Writer) error {
	if len(args) < 2 || args[0] != "add" {
		return errors.New("fonts requires: hydro fonts add FAMILY [--out DIR]")
	}
	family := args[1]
	destination := "public/fonts"
	for i := 2; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--out=") {
			destination = strings.TrimPrefix(args[i], "--out=")
			continue
		}
		if args[i] == "--out" && i+1 < len(args) {
			i++
			destination = args[i]
			continue
		}
		return fmt.Errorf("unknown fonts option %q", args[i])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cssPath, err := designsystem.LocalizeGoogleFont(ctx, family, destination, nil)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "Font files saved. Include /fonts/%s in the edition stylesheet.\n", filepath.Base(cssPath))
	return err
}

// devCommand dispatches to the right "hydro dev" implementation depending on
// where it is run from. HydroGo Core checkouts keep the existing
// tools/devserver-based flow untouched; scaffolded Client Edition projects
// (which have no go.mod/tools/devserver) use runEditionDev instead.
func devCommand(args []string, stdout, stderr io.Writer) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	switch detectDevMode(root) {
	case devModeCore:
		return coreDevCommand(args, stdout, stderr)
	case devModeClientEdition:
		return runEditionDev(args, stdout, stderr)
	default:
		return fmt.Errorf("hydro dev: %q is neither a HydroGo Core checkout (go.mod + tools/devserver) nor a scaffolded Client Edition project (hydrogo.lock + config.yaml)", root)
	}
}

type devMode int

const (
	devModeUnknown devMode = iota
	devModeCore
	devModeClientEdition
)

// detectDevMode decides which `hydro dev` implementation to use for root.
// A HydroGo Core checkout can itself carry a root-level config.yaml and
// hydrogo.lock (used for the Core repo's own local dev/testing), so the two
// signals are not mutually exclusive in practice — the Core checkout check
// intentionally takes priority so existing Core-repo workflows never change.
func detectDevMode(root string) devMode {
	switch {
	case isCoreCheckout(root):
		return devModeCore
	case isClientEdition(root):
		return devModeClientEdition
	default:
		return devModeUnknown
	}
}

// isCoreCheckout reports whether root is a HydroGo Core repository checkout,
// i.e. the source tree that owns tools/devserver.
func isCoreCheckout(root string) bool {
	if info, err := os.Stat(filepath.Join(root, "go.mod")); err != nil || info.IsDir() {
		return false
	}
	info, err := os.Stat(filepath.Join(root, "tools", "devserver"))
	return err == nil && info.IsDir()
}

// isClientEdition reports whether root is a project scaffolded by
// `hydro new`/`hydro generate scaffold`: it ships its own config and lock
// file but no Go source tree for tools/devserver to run against.
func isClientEdition(root string) bool {
	if info, err := os.Stat(filepath.Join(root, "hydrogo.lock")); err != nil || info.IsDir() {
		return false
	}
	info, err := os.Stat(filepath.Join(root, "config.yaml"))
	return err == nil && !info.IsDir()
}

func coreDevCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("dev", flag.ContinueOnError)
	flags.SetOutput(stderr)
	_ = flags.String("address", "0.0.0.0:8090", "development proxy listen address")
	_ = flags.String("app-address", "127.0.0.1:8091", "internal HydroGo application address")
	_ = flags.String("database-dsn", "", "SQLite/PostgreSQL DSN passed to the child server")
	_ = flags.String("redis-url", "", "Optional Redis URL passed to the child server")
	_ = flags.String("dependencies", "auto", "Local dependencies: auto, none, or required")
	_ = flags.String("compose-file", "deploy/local/docker-compose.storage.yml", "Docker Compose file for local dependencies")
	_ = flags.Duration("poll", 600*time.Millisecond, "file watch polling interval")
	_ = flags.Bool("seed-demo", true, "seed the disposable local demo data before starting")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	commandArgs := append([]string{"run", "./tools/devserver"}, args...)
	command := exec.Command("go", commandArgs...)
	command.Stdout, command.Stderr = stdout, stderr
	command.Dir, _ = os.Getwd()
	return command.Run()
}

func generateCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("generate requires a subcommand")
	}
	switch args[0] {
	case "module", "resource", "crud", "page", "form", "api", "admin":
		return generateModuleCommand(args[1:], stdout, stderr)
	case "scaffold":
		return generateScaffoldCommand(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unsupported generate target %q", args[0])
	}
}

// moduleCommand exposes the task-oriented spelling while retaining the
// historical `generate module` command for backwards compatibility.
func moduleCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || (args[0] != "create" && args[0] != "new") {
		return errors.New("module requires the create or new subcommand")
	}
	return generateModuleCommand(args[1:], stdout, stderr)
}

// resourceCommand is the collection-oriented spelling for CRUD scaffolding.
// It intentionally shares the same reviewable generator as module create.
func resourceCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("resource requires create, new or check subcommand")
	}
	if args[0] == "check" {
		return validateResourceCommand(args[1:], stdout, stderr)
	}
	if args[0] != "create" && args[0] != "new" {
		return errors.New("resource requires create, new or check subcommand")
	}
	if !containsString(args[1:], "--spec") {
		return resourceCreateFromFlags(args[1:], stdout, stderr)
	}
	return generateModuleCommand(args[1:], stdout, stderr)
}

func validateResourceCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("resource check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json); omit to discover generated spec.json files")
	root := flags.String("root", "", "Generated project root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*root) == "" {
		return errors.New("resource check requires --root and optionally --spec")
	}
	if strings.TrimSpace(*specPath) != "" {
		spec, err := generator.LoadSpec(*specPath)
		if err != nil {
			return err
		}
		report, err := generator.CheckCompatibility(*root, spec)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "resource-compatible collection=%s contract=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.ContractVersion, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked)
		return err
	}
	reports, err := generator.CheckGeneratedCompatibility(*root)
	if err != nil {
		return err
	}
	for _, report := range reports {
		if _, err := fmt.Fprintf(stdout, "resource-compatible collection=%s contract=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.ContractVersion, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked); err != nil {
			return err
		}
	}
	return nil
}

// apiCommand exposes compatibility checks as a first-class SDK workflow while
// retaining `validate api` as the lower-level spelling.
func frontendCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("frontend requires the check subcommand")
	}
	flags := flag.NewFlagSet("frontend check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	manifest := flags.String("manifest", "", "JSON frontend build manifest")
	jsonOutput := flags.Bool("json", false, "emit normalized routes as JSON")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*manifest) == "" {
		return errors.New("frontend check requires --manifest PATH")
	}
	routes, err := projectgen.ValidateFrontendManifest(*manifest)
	if err != nil {
		return err
	}
	if *jsonOutput {
		data, err := json.MarshalIndent(map[string]any{"version": 1, "routes": routes}, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}
	_, err = fmt.Fprintf(stdout, "frontend-manifest-valid routes=%d path=%s\n", len(routes), *manifest)
	return err
}

func apiCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "diff" {
		return errors.New("api requires the diff subcommand")
	}
	return validateAPICommand(args[1:], stdout, stderr)
}

func permissionsCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("permissions requires the check subcommand")
	}
	flags := flag.NewFlagSet("permissions check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json)")
	jsonOutput := flags.Bool("json", false, "Print machine-readable result")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*specPath) == "" {
		return errors.New("permissions check requires --spec")
	}
	spec, err := generator.LoadSpec(*specPath)
	if err != nil {
		return err
	}
	if err := generator.Validate(spec); err != nil {
		return fmt.Errorf("permission contract: %w", err)
	}
	if *jsonOutput {
		payload, err := json.Marshal(struct {
			Valid       bool   `json:"valid"`
			Collection  string `json:"collection"`
			Permissions int    `json:"permissions"`
		}{true, spec.Name, len(spec.Permissions)})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(payload))
		return err
	}
	_, err = fmt.Fprintf(stdout, "permissions-valid collection=%s permissions=%d\n", spec.Name, len(spec.Permissions))
	return err
}

// schemaCommand compares signed release migration ledgers. It deliberately
// reports IDs/counts only; SQL dialect and migration bodies stay internal.
func schemaCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "diff" {
		return errors.New("schema requires the diff subcommand")
	}
	flags := flag.NewFlagSet("schema diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseline := flags.String("baseline", "", "Baseline signed release manifest")
	current := flags.String("current", "", "Current signed release manifest")
	publicKey := flags.String("public-key", "", "Trusted Ed25519 public key (base64)")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*baseline) == "" || strings.TrimSpace(*current) == "" || strings.TrimSpace(*publicKey) == "" {
		return errors.New("schema diff requires --baseline, --current and --public-key")
	}
	base, err := readVerifiedReleaseManifest(*baseline, *publicKey)
	if err != nil {
		return fmt.Errorf("verify baseline manifest: %w", err)
	}
	next, err := readVerifiedReleaseManifest(*current, *publicKey)
	if err != nil {
		return fmt.Errorf("verify current manifest: %w", err)
	}
	diff, err := release.CompareMigrationChecksums(base.MigrationChecksums, next.MigrationChecksums)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "schema-compatible added=%d removed=%d changed=%d\n", len(diff.Added), len(diff.Removed), len(diff.Changed))
	return err
}

// dbCommand dispatches read-only schema comparison and explicit maintenance.
func dbCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("db requires the diff or reindex subcommand")
	}
	if args[0] == "reindex" {
		return dbReindexCommand(args[1:], stdout, stderr)
	}
	if args[0] != "diff" {
		return errors.New("db requires the diff or reindex subcommand")
	}
	flags := flag.NewFlagSet("db diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	jsonOutput := flags.Bool("json", false, "Emit the migration comparison as JSON")
	if err := flags.Parse(args[1:]); err != nil {
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	pending, err := database.PendingMigrations(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
	if err != nil {
		return err
	}
	var schemaChanges []database.SchemaChange
	physicalInspection := cfg.Database.Dialect == database.SQLite || cfg.Database.Dialect == database.Postgres || cfg.Database.Dialect == database.MySQL
	switch cfg.Database.Dialect {
	case database.SQLite:
		schemaChanges, err = database.SQLiteSchemaDiff(ctx, db, migrations.FS, enabledModuleSet(cfg.Modules))
	case database.Postgres:
		schemaChanges, err = database.PostgresSchemaDiff(ctx, db, migrations.FS, enabledModuleSet(cfg.Modules))
	case database.MySQL:
		schemaChanges, err = database.MySQLSchemaDiff(ctx, db, migrations.FS, enabledModuleSet(cfg.Modules))
	}
	if err != nil {
		return err
	}
	sort.Strings(pending)
	if *jsonOutput {
		return json.NewEncoder(stdout).Encode(struct {
			Clean              bool                    `json:"clean"`
			Pending            []string                `json:"pending"`
			PhysicalInspection bool                    `json:"physical_inspection"`
			SchemaChanges      []database.SchemaChange `json:"schema_changes,omitempty"`
		}{Clean: len(pending) == 0 && len(schemaChanges) == 0, Pending: pending, PhysicalInspection: physicalInspection, SchemaChanges: schemaChanges})
	}
	if len(pending) == 0 && len(schemaChanges) == 0 {
		_, err = fmt.Fprintln(stdout, "schema-diff clean")
		return err
	}
	_, err = fmt.Fprintf(stdout, "schema-diff pending=%s drift=%d physical-inspection=%t\n", strings.Join(pending, ","), len(schemaChanges), physicalInspection)
	return err
}

func dbReindexCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("db reindex", flag.ContinueOnError)
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	started := time.Now()
	reindexErr := database.Reindex(ctx, db, cfg.Database.Dialect)
	closeErr := db.Close()
	if err := errors.Join(reindexErr, closeErr); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "database reindex completed in %s\n", time.Since(started).Round(time.Millisecond))
	return err
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

func resourceCreateFromFlags(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("resource create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "Resource name (snake_case)")
	table := flags.String("table", "", "Database table (defaults to resource name)")
	out := flags.String("out", "generated", "Output root for generated artifacts")
	adminLabel := flags.String("admin-label", "", "Admin collection label")
	var fieldArgs, permissionArgs, dialectArgs repeatedFlag
	flags.Var(&fieldArgs, "field", "Field name:type[:nullable]")
	flags.Var(&permissionArgs, "permission", "Permission name (repeatable)")
	flags.Var(&dialectArgs, "dialect", "Database dialect sqlite|postgres (repeatable)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*name) == "" || len(fieldArgs) == 0 {
		return errors.New("resource create requires --name and at least one --field")
	}
	if strings.TrimSpace(*table) == "" {
		*table = *name
	}
	fields := make([]generator.Field, 0, len(fieldArgs))
	for _, raw := range fieldArgs {
		parts := strings.Split(raw, ":")
		if len(parts) < 2 || len(parts) > 3 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("invalid --field %q; expected name:type[:nullable]", raw)
		}
		nullable := len(parts) == 3 && strings.EqualFold(strings.TrimSpace(parts[2]), "nullable")
		fields = append(fields, generator.Field{Name: strings.TrimSpace(parts[0]), Type: strings.TrimSpace(parts[1]), Nullable: nullable, Admin: generator.FieldAdmin{Label: strings.TrimSpace(parts[0]), VisibleInList: true}})
	}
	permissions := append([]string(nil), permissionArgs...)
	if len(permissions) == 0 {
		permissions = []string{strings.TrimSpace(*name) + ".read", strings.TrimSpace(*name) + ".write"}
	}
	dialects := append([]string(nil), dialectArgs...)
	if len(dialects) == 0 {
		dialects = []string{"sqlite", "postgres"}
	}
	spec := generator.Spec{Name: strings.TrimSpace(*name), Table: strings.TrimSpace(*table), Fields: fields, Permissions: permissions, Dialects: dialects, Admin: generator.Admin{Label: strings.TrimSpace(*adminLabel)}}
	lock, err := generator.Generate(*out, spec)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "resource-created collection=%s out=%s spec_sha256=%s generator=%s\n", spec.Name, *out, lock.SpecSHA256, lock.GeneratorVersion)
	return err
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

func migrationCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("migration requires the create subcommand")
	}
	flags := flag.NewFlagSet("migration create", flag.ContinueOnError)
	flags.SetOutput(stderr)
	id := flags.String("id", "", "Migration identifier (YYYYMMDDHHMM_name)")
	module := flags.String("module", "core", "Owning module name")
	root := flags.String("out", "migrations", "Migration root directory")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || !migrationIDPattern.MatchString(strings.TrimSpace(*id)) {
		return errors.New("migration create requires a valid --id (YYYYMMDDHHMM_name)")
	}
	if strings.TrimSpace(*module) == "" || strings.ContainsAny(*module, `/\\`) {
		return errors.New("--module must be a non-empty name")
	}
	dir := filepath.Join(*root, *id)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("migration already exists: %s", dir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	metadata := fmt.Sprintf("id: %q\nmodule: %q\nclass:\n  sqlite: \"expand\"\n  postgres: \"expand\"\ntransactional:\n  sqlite: true\n  postgres: true\ndependencies: []\nchecksum_sha256:\n  sqlite: \"REPLACE_AFTER_REVIEW\"\n  postgres: \"REPLACE_AFTER_REVIEW\"\nschema:\n  readable_epoch: 1\n  writable_epoch: 1\ncompatibility: \"Describe additive/compatibility behavior before merge.\"\nrollback: \"No production down migration; restore or forward migration only.\"\n", *id, strings.TrimSpace(*module))
	files := map[string]string{"migration.yaml": metadata, "sqlite.sql": "-- Write the SQLite migration after review.\n", "postgres.sql": "-- Write the PostgreSQL migration after review.\n"}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(stdout, "migration-created id=%s path=%s\n", *id, dir)
	return err
}

func generateModuleCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("generate module", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json)")
	outputRoot := flags.String("out", "", "Output root for generated review artifacts")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*specPath) == "" || strings.TrimSpace(*outputRoot) == "" {
		return errors.New("generate module requires --spec and --out")
	}
	spec, err := generator.LoadSpec(*specPath)
	if err != nil {
		return err
	}
	lock, err := generator.Generate(*outputRoot, spec)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "generated collection=%s spec_sha256=%s generator=%s\n", spec.Name, lock.SpecSHA256, lock.GeneratorVersion)
	return err
}

func generateScaffoldCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("generate scaffold", flag.ContinueOnError)
	flags.SetOutput(stderr)
	blueprintPath := flags.String("blueprint", "", "Blueprint YAML/JSON path")
	outputRoot := flags.String("out", ".", "Client Edition output root")
	verify := flags.Bool("verify", true, "Run bounded post-generation gofmt and tests when a Go module is present")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*blueprintPath) == "" {
		return errors.New("generate scaffold requires --blueprint")
	}
	spec, err := generator.LoadBlueprint(*blueprintPath)
	if err != nil {
		return err
	}
	lock, err := generator.Generate(*outputRoot, spec)
	if err != nil {
		return err
	}
	customRoot := filepath.Join(*outputRoot, "custom")
	if err := os.MkdirAll(customRoot, 0o755); err != nil {
		return err
	}
	customReadme := filepath.Join(customRoot, spec.Name+"_policy.go")
	if _, err := os.Stat(customReadme); errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf("// Client Edition policy for %s. Generated artifacts stay under generated/%s.\npackage custom\n\n// Add authorization, workflow, and tenant policy here.\n", spec.Name, spec.Name)
		if err := os.WriteFile(customReadme, []byte(content), 0o644); err != nil {
			return err
		}
	}
	hooksFile := filepath.Join(customRoot, spec.Name+"_hooks.go")
	if _, err := os.Stat(hooksFile); errors.Is(err, os.ErrNotExist) {
		content := fmt.Sprintf(`// Client Edition lifecycle hooks for %s.
// This file is human-owned; generated artifacts stay under generated/%s.
package custom

import "context"

// %sHooks is an explicit extension point. Keep database, authorization and
// tenant decisions in the owning service; dispatch side effects through the
// durable outbox rather than performing network I/O inline.
type %sHooks struct{}

func (%sHooks) BeforeCreate(context.Context, any) error { return nil }
func (%sHooks) AfterCreate(context.Context, any) error  { return nil }
func (%sHooks) BeforeUpdate(context.Context, any, any) error { return nil }
func (%sHooks) AfterUpdate(context.Context, any, any) error  { return nil }
func (%sHooks) BeforeDelete(context.Context, any) error { return nil }
func (%sHooks) AfterDelete(context.Context, any) error  { return nil }
`, spec.Name, spec.Name, generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name), generatorExportName(spec.Name))
		if err := os.WriteFile(hooksFile, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if *verify {
		if err := verifyGeneratedScaffold(*outputRoot); err != nil {
			return fmt.Errorf("generated scaffold verification: %w", err)
		}
	}
	_, err = fmt.Fprintf(stdout, "scaffold-created collection=%s generated=%s custom=%s spec_sha256=%s\n", spec.Name, filepath.Join(*outputRoot, "generated", spec.Name), customRoot, lock.SpecSHA256)
	return err
}

// generatorExportName mirrors the generator's stable Go identifier convention
// for scaffold-owned extension points without exposing generator internals to
// the CLI package.
func generatorExportName(value string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func verifyGeneratedScaffold(root string) error {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	files := make([]string, 0)
	for _, dir := range []string{filepath.Join(root, "generated"), filepath.Join(root, "custom")} {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr == nil && info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
				files = append(files, path)
			}
			return nil
		})
	}
	if len(files) == 0 {
		return nil
	}
	formatArgs := append([]string{"-w"}, files...)
	format := exec.CommandContext(ctx, "gofmt", formatArgs...)
	format.Dir = root
	if output, err := format.CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	test := exec.CommandContext(ctx, "go", "test", "./...")
	test.Dir = root
	if output, err := test.CombinedOutput(); err != nil {
		return fmt.Errorf("go test: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

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

func portableCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("portable command requires a subcommand (export or import)")
	}
	flags := flag.NewFlagSet("portable "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	path := flags.String("path", "", "Portable document path")
	maxBytes := flags.Int64("max-bytes", portable.DefaultMaxDocumentBytes, "Maximum portable document size in bytes")
	confirm := flags.Bool("confirm", false, "Confirm replacing allowlisted content tables")
	dryRun := flags.Bool("dry-run", false, "Validate the document without changing the database")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*path) == "" {
		return errors.New("--path is required")
	}
	switch args[0] {
	case "export":
		if *confirm || *dryRun {
			return errors.New("--confirm and --dry-run are only valid for portable import")
		}
		if err := validatePortableOutputPath(*path); err != nil {
			return err
		}
		if _, err := os.Stat(*path); err == nil {
			return fmt.Errorf("portable export destination already exists: %s", *path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg, err := loadRuntimeConfig(flags, values)
		if err != nil {
			return err
		}
		budget, err := capacity.Defaults(cfg.Profile)
		if err != nil {
			return err
		}
		db, err := database.OpenWithOptions(context.Background(), cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
		if err != nil {
			return err
		}
		defer db.Close()
		document, err := portable.ExportWithLimit(context.Background(), db, *maxBytes)
		if err != nil {
			return fmt.Errorf("portable export: %w", err)
		}
		data, err := portable.Marshal(document)
		if err != nil {
			return fmt.Errorf("marshal portable export: %w", err)
		}
		if *maxBytes <= 0 || int64(len(data)) > *maxBytes {
			return portable.ErrDocumentTooLarge
		}
		temporary, err := os.CreateTemp(filepath.Dir(*path), ".hydro-portable-*.tmp")
		if err != nil {
			return fmt.Errorf("create portable export temp file: %w", err)
		}
		temporaryPath := temporary.Name()
		defer os.Remove(temporaryPath)
		if err := temporary.Chmod(0o640); err != nil {
			temporary.Close()
			return fmt.Errorf("protect portable export temp file: %w", err)
		}
		if _, err := temporary.Write(append(data, '\n')); err != nil {
			temporary.Close()
			return fmt.Errorf("write portable export: %w", err)
		}
		if err := temporary.Close(); err != nil {
			return fmt.Errorf("close portable export: %w", err)
		}
		if err := os.Rename(temporaryPath, *path); err != nil {
			return fmt.Errorf("publish portable export: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "portable-exported path=%s tables=%d\n", *path, len(document.Tables))
		return err
	case "import":
		if err := validatePortableInputPath(*path); err != nil {
			return err
		}
		if *confirm && *dryRun {
			return errors.New("--confirm and --dry-run cannot be used together")
		}
		if !*confirm && !*dryRun {
			return errors.New("portable import requires --confirm, or use --dry-run")
		}
		file, err := os.Open(*path)
		if err != nil {
			return fmt.Errorf("read portable document: %w", err)
		}
		defer file.Close()
		document, err := portable.UnmarshalReader(file, *maxBytes)
		if err != nil {
			return fmt.Errorf("validate portable document: %w", err)
		}
		if *dryRun {
			_, err = fmt.Fprintf(stdout, "portable-import-valid path=%s tables=%d\n", *path, len(document.Tables))
			return err
		}
		cfg, err := loadRuntimeConfig(flags, values)
		if err != nil {
			return err
		}
		budget, err := capacity.Defaults(cfg.Profile)
		if err != nil {
			return err
		}
		db, err := database.OpenWithOptions(context.Background(), cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
		if err != nil {
			return err
		}
		defer db.Close()
		if err := database.NewTransactor(db).Within(context.Background(), func(tx database.Executor) error {
			return portable.Import(context.Background(), tx, document, cfg.Database.Dialect)
		}); err != nil {
			return fmt.Errorf("portable import: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "portable-imported path=%s tables=%d\n", *path, len(document.Tables))
		return err
	default:
		return fmt.Errorf("unknown portable subcommand %q", args[0])
	}
}

func validatePortableOutputPath(path string) error {
	if err := pathguard.RejectSymlinkComponents(filepath.Dir(path)); err != nil {
		return fmt.Errorf("portable export destination: %w", err)
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("portable export destination cannot be a symlink")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func validatePortableInputPath(path string) error {
	if err := pathguard.RejectSymlinkComponents(filepath.Dir(path)); err != nil {
		return fmt.Errorf("portable import source: %w", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("portable import source cannot be a symlink")
	}
	return nil
}

func backupCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("backup command requires a subcommand (sqlite or postgres)")
	}
	if args[0] == "prune" {
		return pruneBackups(args[1:], stdout, stderr)
	}
	if args[0] != "sqlite" && args[0] != "postgres" {
		return fmt.Errorf("unknown backup subcommand %q", args[0])
	}
	flags := flag.NewFlagSet("backup sqlite", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	destination := flags.String("destination", "", "New snapshot directory")
	mediaDir := flags.String("media-dir", "", "Media directory override")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*destination) == "" {
		return errors.New("--destination is required")
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	if cfg.License.Required {
		licenseResult, licenseErr := licensing.VerifyFile(context.Background(), cfg.License.File, cfg.License.ProjectID, cfg.License.Edition, cfg.License.Features)
		if licenseErr != nil {
			return fmt.Errorf("runtime license verification failed: %w", licenseErr)
		}
		if licenseResult.State == licensing.StateGracePeriod {
			_, _ = fmt.Fprintf(stderr, "hydro: warning: %s\n", licenseResult.Message)
		}
	}
	if *mediaDir != "" {
		cfg.Media.Dir = *mediaDir
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if args[0] == "postgres" {
		if cfg.Database.Dialect != database.Postgres {
			return errors.New("backup postgres requires --database-dialect postgres")
		}
		tools, err := backupstore.LocatePostgresTools()
		if err != nil {
			return err
		}
		manifest, err := backupstore.SnapshotPostgres(ctx, tools, cfg.Database.DSN, cfg.Media.Dir, *destination)
		if err != nil {
			return fmt.Errorf("create PostgreSQL backup: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "backup-created dialect=postgres destination=%s database-bytes=%d media-files=%d\n", *destination, manifest.Database.Bytes, len(manifest.Media))
		return err
	}
	if cfg.Database.Dialect != database.SQLite {
		return errors.New("backup sqlite requires --database-dialect sqlite")
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	manifest, err := backupstore.SnapshotSQLite(ctx, db, cfg.Media.Dir, *destination)
	if err != nil {
		return fmt.Errorf("create SQLite backup: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "backup-created destination=%s database-bytes=%d media-files=%d\n", *destination, manifest.Database.Bytes, len(manifest.Media))
	return err
}

func pruneBackups(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("backup prune", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Snapshot root directory")
	keep := flags.Int("keep", 1, "Number of newest unpinned snapshots to retain")
	pinned := flags.String("pinned", "", "Comma-separated snapshot directory names to retain")
	dryRun := flags.Bool("dry-run", false, "Plan removals without changing the filesystem")
	confirm := flags.Bool("confirm", false, "Confirm deletion of planned snapshots")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*root) == "" {
		return errors.New("--root is required")
	}
	if *dryRun && *confirm {
		return errors.New("--dry-run and --confirm cannot be used together")
	}
	if !*dryRun && !*confirm {
		return errors.New("backup prune requires --confirm, or use --dry-run")
	}
	policy := backupstore.SnapshotRetention{Keep: *keep, Pinned: parsePinnedNames(*pinned)}
	planned, err := backupstore.PlanSnapshotPrune(*root, policy)
	if err != nil {
		return fmt.Errorf("plan backup retention: %w", err)
	}
	if *dryRun {
		_, err = fmt.Fprintf(stdout, "backup-prune-plan root=%s snapshots=%s\n", *root, strings.Join(planned, ","))
		return err
	}
	removed, err := backupstore.PruneSnapshots(*root, policy)
	if err != nil {
		return fmt.Errorf("prune backups: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "backup-pruned root=%s snapshots=%s\n", *root, strings.Join(removed, ","))
	return err
}

func parsePinnedNames(value string) map[string]struct{} {
	pinned := make(map[string]struct{})
	for _, name := range strings.Split(value, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			pinned[name] = struct{}{}
		}
	}
	return pinned
}

func restoreCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("restore command requires a subcommand (unpack, verify, stage, postgres or activate-media)")
	}
	switch args[0] {
	case "unpack":
		flags := flag.NewFlagSet("restore unpack", flag.ContinueOnError)
		flags.SetOutput(stderr)
		archivePath := flags.String("archive", "", "Downloaded .tar.gz snapshot")
		destination := flags.String("destination", "", "New empty snapshot directory")
		maxBytes := flags.Int64("max-bytes", backupstore.DefaultArchiveExtractLimit, "Maximum uncompressed archive size")
		publicKey := flags.String("public-key", "", "Trusted Ed25519 public key in hexadecimal")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", flags.Args())
		}
		if *maxBytes < 1 || *maxBytes > 1<<40 {
			return errors.New("--max-bytes must be between 1 byte and 1 TiB")
		}
		trusted, err := backupstore.ParseSigningPublicKey(*publicKey)
		if err != nil {
			return err
		}
		if err := backupstore.ExtractSignedSnapshotArchive(context.Background(), *archivePath, *destination, *maxBytes, trusted); err != nil {
			return fmt.Errorf("unpack backup archive: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "restore-unpacked destination=%s\n", *destination)
		return err
	case "verify":
		flags := flag.NewFlagSet("restore verify", flag.ContinueOnError)
		flags.SetOutput(stderr)
		snapshot := flags.String("snapshot", "", "Snapshot directory")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", flags.Args())
		}
		if strings.TrimSpace(*snapshot) == "" {
			return errors.New("--snapshot is required")
		}
		manifest, err := backupstore.ReadManifest(filepath.Join(*snapshot, "manifest.json"))
		if err != nil {
			return fmt.Errorf("read backup manifest: %w", err)
		}
		if err := backupstore.Verify(*snapshot, manifest); err != nil {
			return fmt.Errorf("verify backup: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "restore-verified snapshot=%s database-bytes=%d media-files=%d\n", *snapshot, manifest.Database.Bytes, len(manifest.Media))
		return err
	case "stage":
		flags := flag.NewFlagSet("restore stage", flag.ContinueOnError)
		flags.SetOutput(stderr)
		snapshot := flags.String("snapshot", "", "Verified snapshot directory")
		destination := flags.String("destination", "", "New inactive staging directory")
		sessionsRevoked := flags.Bool("sessions-revoked", false, "Confirm all restored sessions are revoked")
		egressQuarantined := flags.Bool("egress-quarantined", false, "Confirm outbound integrations are quarantined")
		oldWritesBlocked := flags.Bool("old-writes-blocked", false, "Confirm old-version writes are blocked")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", flags.Args())
		}
		if strings.TrimSpace(*snapshot) == "" || strings.TrimSpace(*destination) == "" {
			return errors.New("--snapshot and --destination are required")
		}
		controls := backupstore.RestoreControls{SessionsRevoked: *sessionsRevoked, EgressQuarantined: *egressQuarantined, OldWritesBlocked: *oldWritesBlocked}
		manifest, err := backupstore.ReadManifest(filepath.Join(*snapshot, "manifest.json"))
		if err != nil {
			return fmt.Errorf("read backup manifest: %w", err)
		}
		if err := backupstore.RestoreToStaging(*snapshot, *destination, manifest, controls); err != nil {
			return fmt.Errorf("stage restore: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "restore-staged destination=%s controls=sessions-revoked,egress-quarantined,old-writes-blocked\n", *destination)
		return err
	case "postgres":
		flags := flag.NewFlagSet("restore postgres", flag.ContinueOnError)
		flags.SetOutput(stderr)
		values := addRuntimeFlags(flags)
		snapshot := flags.String("snapshot", "", "Verified snapshot directory")
		destination := flags.String("destination", "", "New inactive staging directory")
		sessionsRevoked := flags.Bool("sessions-revoked", false, "Confirm all restored sessions are revoked")
		egressQuarantined := flags.Bool("egress-quarantined", false, "Confirm outbound integrations are quarantined")
		oldWritesBlocked := flags.Bool("old-writes-blocked", false, "Confirm old-version writes are blocked")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", flags.Args())
		}
		if strings.TrimSpace(*snapshot) == "" || strings.TrimSpace(*destination) == "" {
			return errors.New("--snapshot and --destination are required")
		}
		cfg, err := loadRuntimeConfig(flags, values)
		if err != nil {
			return err
		}
		if cfg.Database.Dialect != database.Postgres {
			return errors.New("restore postgres requires --database-dialect postgres")
		}
		tools, err := backupstore.LocatePostgresTools()
		if err != nil {
			return err
		}
		manifest, err := backupstore.ReadManifest(filepath.Join(*snapshot, "manifest.json"))
		if err != nil {
			return fmt.Errorf("read backup manifest: %w", err)
		}
		controls := backupstore.RestoreControls{SessionsRevoked: *sessionsRevoked, EgressQuarantined: *egressQuarantined, OldWritesBlocked: *oldWritesBlocked}
		if err := backupstore.RestorePostgresToStaging(context.Background(), tools, cfg.Database.DSN, *snapshot, *destination, manifest, controls); err != nil {
			return fmt.Errorf("stage PostgreSQL restore: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "restore-staged dialect=postgres destination=%s controls=sessions-revoked,egress-quarantined,old-writes-blocked\n", *destination)
		return err
	case "activate-media":
		flags := flag.NewFlagSet("restore activate-media", flag.ContinueOnError)
		flags.SetOutput(stderr)
		staging := flags.String("staging", "", "Quarantined staging directory")
		active := flags.String("active", "", "Active media directory")
		rollback := flags.String("rollback", "", "Rollback media directory")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", flags.Args())
		}
		if strings.TrimSpace(*staging) == "" || strings.TrimSpace(*active) == "" || strings.TrimSpace(*rollback) == "" {
			return errors.New("--staging, --active and --rollback are required")
		}
		if err := backupstore.ActivateStagingMedia(*staging, *active, *rollback); err != nil {
			return fmt.Errorf("activate restored media: %w", err)
		}
		_, err := fmt.Fprintf(stdout, "restore-media-activated active=%s rollback=%s\n", *active, *rollback)
		return err
	default:
		return fmt.Errorf("unknown restore subcommand %q", args[0])
	}
}

// enforceDevModeOfficialBuild refuses --dev/HYDRO_DEV_MODE when this binary
// was built as an official release with a vendor public key embedded
// (internal/licensing.TrustedPublicKeyBase64 injected via -ldflags). Dev mode
// must only be reachable from ordinary `go build` binaries that were never
// part of the official distribution pipeline, regardless of bind address or
// any reverse proxy placed in front of the process.
func enforceDevModeOfficialBuild() error {
	if licensing.TrustedPublicKeyBase64 != "" {
		return errors.New("dev mode is not available in official builds: --dev/HYDRO_DEV_MODE is disabled because this binary was built with a vendor public key embedded")
	}
	return nil
}

// enforceDevModeLocalBind requires --dev/HYDRO_DEV_MODE to bind only to a
// loopback address, so the unlicensed server can never be reachable from
// outside the local machine.
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

// printDevModeWarning prints a large, impossible-to-miss banner so an
// unlicensed dev-mode server is never mistaken for a licensed one.
func printDevModeWarning(stderr io.Writer) {
	fmt.Fprint(stderr, `
################################################################
#                                                              #
#   UNLICENSED DEV MODE - NOT FOR PRODUCTION                   #
#                                                              #
#   License verification was bypassed via --dev/HYDRO_DEV_MODE #
#   This server only works on 127.0.0.1/localhost/::1 and is   #
#   disabled in official builds. Never expose it publicly.     #
#                                                              #
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
	devLicenseBypass := flags.Bool("dev", false, "bypass license verification; only works in non-official builds bound to 127.0.0.1/localhost, never for production")
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
	devMode := *devLicenseBypass || devModeEnv
	if devMode {
		if err := enforceDevModeOfficialBuild(); err != nil {
			return err
		}
		if err := enforceDevModeLocalBind(cfg.HTTP.Address); err != nil {
			return err
		}
		printDevModeWarning(stderr)
	} else if _, err := validateRuntimeLicense(cfg, stderr); err != nil {
		return err
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
		return fmt.Errorf("connect redis: %w", err)
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
		bundle, err := themev1.LoadTheme(cfg.ThemeDir, themeName)
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

	graphqlHTTP, graphqlEnabled := newGraphQLHandler(db, contentService, modules["pages"])
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
		pageHTTP = pagetransport.Handler{DB: db, Content: contentService, Theme: selectedTheme, Media: mediaService, AdminCollection: pageCollection, DefaultLocale: cfg.I18N.DefaultLocale, SupportedLocales: cfg.I18N.Locales, Assistant: contentAssistant, AIAudit: contentAIAudit, AllowDemoFrontend: demoFrontendEnabled(), SEOOrigin: cfg.HTTP.PublicOrigin, PreviewSigner: previewSigner, PreviewOrigin: cfg.HTTP.PublicOrigin, RequireTenantScope: requireTenantScope, SearchEnabled: modules["search"], ContentViews: analyticsStore, Feedback: analyticsStore, AnalyticsEnabled: modules["analytics"], Presence: editPresence, RecordSlugRedirect: recordSlugRedirect, RecordNotFound: recordNotFound, PublicSupport: adminstudio.SupportConfig{Enabled: cfg.AdminSupport.Enabled, Phone: cfg.AdminSupport.Phone, WhatsAppURL: cfg.AdminSupport.WhatsAppURL}, OGRenderer: &seo.OGRenderer{CacheDir: cfg.SEO.OGCacheDir}, DiscoveryCache: pageLLMSCache, Notifications: &notificationStore, WeeklyDigestEnabled: modules["forms"] && smtpSender != nil, NavigationProvider: func(navCtx context.Context, locale string) ([]site.NavigationItem, error) {
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
		routes = append(routes, (collectiontransport.Handler{Service: collectionService}).Routes()...)
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
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(adminstudio.E2EManifestForCollections(activeStudioCollections()))
	})})
	routes = append(routes, router.Route{Pattern: "GET /admin/extensions/manifest", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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
	routes = append(routes, router.Route{Pattern: "POST /admin/extensions/channels/bind", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, session := auth.UserFromContext(r.Context()), auth.SessionFromContext(r.Context())
		if user == nil || session == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var input struct {
			ChannelID string `json:"channel_id"`
			Challenge string `json:"challenge"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		principal := adminstudio.ChannelPrincipal{SessionID: session.ID, UserID: user.ID}
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			principal.TenantID = scope.TenantID
		}
		if err := extensionRegistry.BindChannel(input.ChannelID, input.Challenge, principal); err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})})
	routes = append(routes, router.Route{Pattern: "POST /admin/extensions/channels/revoke", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, session := auth.UserFromContext(r.Context()), auth.SessionFromContext(r.Context())
		if user == nil || session == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var input struct {
			ChannelID string `json:"channel_id"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 512)
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.ChannelID == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		principal := adminstudio.ChannelPrincipal{SessionID: session.ID, UserID: user.ID}
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			principal.TenantID = scope.TenantID
		}
		// Deliberately idempotent: callers do not learn whether a channel existed.
		extensionRegistry.RevokeChannel(input.ChannelID, principal)
		w.WriteHeader(http.StatusNoContent)
	})})
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
			if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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
	routes = append(routes, router.Route{Pattern: "GET /admin/backup", Access: router.Protected, Handler: adminBackupHandler}, router.Route{Pattern: "POST /admin/backup", Access: router.Protected, Handler: adminBackupHandler})
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
			if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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
	routes = append(routes,
		router.Route{Pattern: "POST /admin/security/bans", Access: router.Protected, Handler: adminBanMutation(false)},
		router.Route{Pattern: "POST /admin/security/bans/remove", Access: router.Protected, Handler: adminBanMutation(true)},
	)
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
	routes = append(routes, router.Route{Pattern: "GET /vendor/{path...}", Access: router.Public, Handler: http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS)))})
	/* Commerce is intentionally deferred; see ADR-0199. Retain this historical
	composition only until the removed module files disappear from all supported
	customer branches. It is excluded from the build below. */
	/*
		if moduleEnabled(cfg.Modules, "commerce") {
			commerceStore := commercestore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
			commerceService := commerce.Service{Transactions: transactioner, Store: commerceStore, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}}
			go runCommerceReservationReleaser(ctx, commerceService, 15*time.Minute, stderr)
			commerceCatalog := commerce.CatalogService{Transactions: transactioner, Store: commerceStore, Variants: commerceStore}
			graphqlHTTP.Catalog = commerceCatalog
			graphqlHTTP.Commerce = true
			providers := make(map[string]commerce.PaymentProvider)
			if cfg.Payments.StripeSecretKey != "" {
				providers["stripe"] = stripeprovider.Provider{SecretKey: cfg.Payments.StripeSecretKey, WebhookSecret: cfg.Payments.StripeWebhookSecret}
			}
			if cfg.Payments.IyzicoSecretKey != "" {
				providers["iyzico"] = iyzicoprovider.Provider{APIKey: cfg.Payments.IyzicoAPIKey, SecretKey: cfg.Payments.IyzicoSecretKey, BaseURL: cfg.Payments.IyzicoBaseURL, CallbackURL: cfg.Payments.IyzicoCallbackURL}
			}
			if cfg.Payments.PayTRMerchantKey != "" {
				providers["paytr"] = paytrprovider.Provider{MerchantID: cfg.Payments.PayTRMerchantID, MerchantKey: cfg.Payments.PayTRMerchantKey, MerchantSalt: cfg.Payments.PayTRMerchantSalt, Currency: cfg.Payments.PayTRCurrency, BaseURL: cfg.Payments.PayTRBaseURL, SuccessURL: cfg.Payments.PayTRSuccessURL, FailURL: cfg.Payments.PayTRFailURL}
			}
			paymentService := commerce.PaymentService{Transactions: transactioner, Store: commerceStore, Providers: providers}
			commerceCollection, ok := studioRegistry.Collection("commerce")
			if !ok {
				return errors.New("commerce admin studio metadata missing")
			}
			commerceCollection.Fields = append(commerceCollection.Fields, adminstudio.Field{ID: "description", Label: "Ürün açıklaması", Kind: adminstudio.FieldText, Widget: "richtext"})
			commerceHTTP := commercetransport.Handler{DB: db, Media: mediaService, Commerce: commerceService, Catalog: commerceCatalog, CatalogVariants: commerceStore, Orders: commerceStore, Payments: providers, PaymentService: paymentService, Limiter: newRateLimiter(10, time.Minute), Cart: commerce.CartService{Transactions: transactioner, Store: commerceStore}, AdminCollection: commerceCollection, SecureCookies: cfg.Security.SecureCookies, DefaultLocale: cfg.I18N.DefaultLocale,
				ResolveMemberID: func(r *http.Request) (string, bool) {
					cookie, err := r.Cookie(members.SessionCookieName)
					if err != nil || strings.TrimSpace(cookie.Value) == "" {
						return "", false
					}
					_, member, err := memberService.ValidateSession(r.Context(), cookie.Value)
					if err != nil || member == nil || !member.Active {
						return "", false
					}
					return member.ID, true
				},
				ResolveMobileMemberID: func(r *http.Request) (string, bool) {
					if memberHTTP.Mobile == nil {
						return "", false
					}
					parts := strings.Fields(r.Header.Get("Authorization"))
					if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
						return "", false
					}
					memberID, _, err := memberHTTP.Mobile.ValidateAccess(r.Context(), parts[1])
					if err != nil || strings.TrimSpace(memberID) == "" {
						return "", false
					}
					return memberID, true
				},
			}
			variantCollection, ok := studioRegistry.Collection("commerce_variants")
			if !ok {
				return errors.New("commerce variants admin studio metadata missing")
			}
			commerceHTTP.VariantCollection = variantCollection
			paymentHTTP := commercetransport.PaymentWebhookHandler{Service: paymentService}
			routes = append(routes, commerceRouterRoutes(commerceHTTP.Routes())...)
			routes = append(routes, commerceRouterRoutes(paymentHTTP.Routes())...)
		}
	*/
	graphqlEnabled = graphqlEnabled && modules["pages"]
	if graphqlEnabled {
		routes = append(routes, graphQLRoutes(graphqlHTTP, newRateLimiter(60, time.Minute))...)
	}

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
		SecurityStyleHashes:   ui.AdminStyleHashes(),
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

func runOutboxWorker(ctx context.Context, worker webhooks.Worker, stderr io.Writer) {
	if len(worker.Dispatcher.Endpoints) == 0 {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		processed, err := worker.RunOnce(ctx, time.Now().UTC())
		if err != nil && ctx.Err() == nil && stderr != nil {
			_, _ = fmt.Fprintf(stderr, "hydro outbox worker: %v\n", err)
		}
		if !processed {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}
}

func runAdminWeeklyDigestOnce(ctx context.Context, now time.Time, db database.QueryExecutor, tx database.Transactor, store adminstudiosql.Store, users *authstore.UserStore, spool mailstore.Spool, scopes []tenant.Scope, requireTenant, analyticsEnabled, membersEnabled bool) (int, error) {
	if err := database.RequireExecutor(db); err != nil {
		return 0, err
	}
	if users == nil {
		return 0, errors.New("digest user store is required")
	}
	if requireTenant && len(scopes) == 0 {
		return 0, tenant.ErrMissingScope
	}
	workScopes := scopes
	if !requireTenant {
		workScopes = []tenant.Scope{{}}
	}
	queued := 0
	for _, scope := range workScopes {
		jobCtx := ctx
		if requireTenant {
			jobCtx = tenant.WithScope(jobCtx, scope)
		}
		statsByWindow := map[string]adminstudio.DigestStats{}
		after := ""
		for {
			preferences, err := store.DigestPreferencesAfter(jobCtx, db, after, 500)
			if err != nil {
				return queued, err
			}
			if len(preferences) == 0 {
				break
			}
			after = preferences[len(preferences)-1].UserID
			duePreferences := make([]adminstudio.DigestPreference, 0, len(preferences))
			digestUsers := make([]adminstudio.DigestUser, 0, len(preferences))
			for _, preference := range preferences {
				frequency := adminstudio.DigestFrequency(preference.Frequency)
				if frequency == "" {
					frequency = adminstudio.DigestWeekly
				}
				_, _, key, due, err := adminstudio.DigestWindow(now, preference.Timezone, frequency)
				if err != nil {
					return queued, err
				}
				if !due || preference.LastSentPeriod == key {
					continue
				}
				account, err := users.GetByID(jobCtx, preference.UserID)
				if errors.Is(err, auth.ErrUserNotFound) {
					continue
				}
				if err != nil {
					return queued, err
				}
				if account.Role != auth.RoleAdmin && account.Role != auth.RoleEditor {
					continue
				}
				duePreferences = append(duePreferences, preference)
				digestUsers = append(digestUsers, adminstudio.DigestUser{ID: account.ID, Email: account.Email, Active: account.Active})
			}
			if len(duePreferences) > 0 {
				err = adminstudio.DeliverDueDigests(jobCtx, now, duePreferences, digestUsers, func(statsCtx context.Context, start, end time.Time) (adminstudio.DigestStats, error) {
					key := start.UTC().Format(time.RFC3339Nano) + "/" + end.UTC().Format(time.RFC3339Nano)
					if stats, ok := statsByWindow[key]; ok {
						return stats, nil
					}
					stats, err := store.WeeklyDigestStats(statsCtx, db, start, end, analyticsEnabled, membersEnabled)
					if err == nil {
						statsByWindow[key] = stats
					}
					return stats, err
				}, func(deliveryCtx context.Context, account adminstudio.DigestUser, period, body, htmlBody string) error {
					return tx.Within(deliveryCtx, func(exec database.Executor) error {
						claimed, err := store.ClaimDigestWeek(deliveryCtx, exec, account.ID, period)
						if err != nil || !claimed {
							return err
						}
						subject, eventKey := "Haftalık site özeti", "admin-weekly:"
						if strings.Contains(body, "Aylık yönetici özeti") {
							subject, eventKey = "Aylık yönetici özeti", "admin-monthly:"
						}
						if err := spool.Enqueue(deliveryCtx, exec, site.EmailNotification{EventKey: eventKey + account.ID + ":" + period, To: account.Email, Subject: subject, Body: body, HTMLBody: htmlBody}); err != nil {
							return err
						}
						queued++
						return nil
					})
				})
				if err != nil {
					return queued, err
				}
			}
			if len(preferences) < 500 {
				break
			}
		}
	}
	return queued, nil
}

/*
func runCommerceReservationReleaser(ctx context.Context, service commerce.Service, hold time.Duration, stderr io.Writer) {
	if hold <= 0 {
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		_, err := service.ExpirePendingOrders(ctx, time.Now().UTC().Add(-hold))
		if err != nil && ctx.Err() == nil && stderr != nil {
			_, _ = fmt.Fprintf(stderr, "hydro commerce reservation releaser: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
*/

func doctor(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	jsonOutput := flags.Bool("json", false, "Print machine-readable diagnostics")
	fix := flags.Bool("fix", false, "Repair safe local issues and report actions")
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	fixes := make([]string, 0, 2)
	sqliteJournalMode := ""
	sqliteForeignKeyViolations := 0
	if cfg.Database.Dialect == database.SQLite {
		if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&sqliteJournalMode); err != nil {
			return fmt.Errorf("database journal mode check: %w", err)
		}
		rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
		if err != nil {
			return fmt.Errorf("database foreign key check: %w", err)
		}
		for rows.Next() {
			sqliteForeignKeyViolations++
			if sqliteForeignKeyViolations > 1000 {
				break
			}
		}
		rowsErr := rows.Err()
		_ = rows.Close()
		if rowsErr != nil {
			return fmt.Errorf("database foreign key check: %w", rowsErr)
		}
		if sqliteForeignKeyViolations != 0 {
			return fmt.Errorf("database foreign key check found violations")
		}
		if *fix {
			var integrity string
			if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil {
				return fmt.Errorf("database integrity check: %w", err)
			}
			if integrity != "ok" {
				return fmt.Errorf("database integrity check found an issue")
			}
			var busy, logFrames, checkpointed int
			if err := db.QueryRowContext(ctx, "PRAGMA wal_checkpoint(PASSIVE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
				return fmt.Errorf("database WAL checkpoint: %w", err)
			}
			if busy != 0 {
				return fmt.Errorf("database WAL checkpoint is busy")
			}
			fixes = append(fixes, "sqlite_wal_checkpoint_passive")
		}
	}
	if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		return fmt.Errorf("schema compatibility: %w", err)
	}
	if err := os.MkdirAll(cfg.Media.Dir, 0o750); err != nil {
		return fmt.Errorf("media directory: %w", err)
	}
	mediaMode := ""
	info, err := os.Stat(cfg.Media.Dir)
	if err != nil {
		return fmt.Errorf("media directory: %w", err)
	}
	mediaMode = fmt.Sprintf("%04o", info.Mode().Perm())
	if *fix {
		if info.Mode().Perm() != 0o750 {
			if err := os.Chmod(cfg.Media.Dir, 0o750); err != nil {
				return fmt.Errorf("media directory permissions: %w", err)
			}
			fixes = append(fixes, "media_directory_mode_0750")
			mediaMode = "0750"
		}
	}
	if err := checkDirectoryWritable(cfg.Media.Dir); err != nil {
		return fmt.Errorf("media directory is not writable: %w", err)
	}
	backupTools := "sqlite"
	if cfg.Database.Dialect == database.Postgres {
		if _, err := backupstore.LocatePostgresTools(); err != nil {
			return fmt.Errorf("PostgreSQL backup tools: %w", err)
		}
		backupTools = "pg_dump,pg_restore"
	}
	deadLetteredLiteral := "0"
	if cfg.Database.Dialect == database.Postgres {
		deadLetteredLiteral = "TRUE"
	}
	var deadLetteredJobs int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE dead_lettered = "+deadLetteredLiteral).Scan(&deadLetteredJobs); err != nil {
		return fmt.Errorf("job diagnostics: %w", err)
	}
	if *jsonOutput {
		version, err := currentVersionInfo()
		if err != nil {
			return err
		}
		data, err := json.Marshal(struct {
			Version                    string   `json:"version"`
			Commit                     string   `json:"commit"`
			BuildTime                  string   `json:"build_time"`
			SchemaMigration            string   `json:"schema_migration"`
			Profile                    string   `json:"profile"`
			Database                   string   `json:"database"`
			BackupTools                string   `json:"backup_tools"`
			CacheEntries               int      `json:"cache_entries"`
			QueueEntries               int      `json:"queue_entries"`
			WorkerConcurrency          int      `json:"worker_concurrency"`
			DeadLetteredJobs           int64    `json:"dead_lettered_jobs"`
			MediaMaxBytes              int64    `json:"media_max_bytes"`
			MediaWritable              bool     `json:"media_writable"`
			SecureCookies              bool     `json:"secure_cookies"`
			HSTS                       bool     `json:"hsts"`
			Fixes                      []string `json:"fixes,omitempty"`
			SQLiteJournalMode          string   `json:"sqlite_journal_mode,omitempty"`
			SQLiteForeignKeyViolations int      `json:"sqlite_foreign_key_violations,omitempty"`
			MediaMode                  string   `json:"media_mode"`
		}{Version: version.Version, Commit: version.Commit, BuildTime: version.BuildTime, SchemaMigration: version.SchemaMigration, Profile: string(cfg.Profile), Database: string(cfg.Database.Dialect), BackupTools: backupTools, CacheEntries: budget.CacheEntries, QueueEntries: budget.QueueEntries, WorkerConcurrency: budget.WorkerConcurrency, DeadLetteredJobs: deadLetteredJobs, MediaMaxBytes: budget.MediaMaxBytes, MediaWritable: true, SecureCookies: cfg.Security.SecureCookies, HSTS: cfg.Security.EnableHSTS, Fixes: fixes, SQLiteJournalMode: sqliteJournalMode, SQLiteForeignKeyViolations: sqliteForeignKeyViolations, MediaMode: mediaMode})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}
	_, err = fmt.Fprintf(stdout, "profile=%s database=%s sqlite_journal_mode=%s sqlite_foreign_key_violations=%d media_mode=%s backup_tools=%s cache_entries=%d queue_entries=%d worker_concurrency=%d dead_lettered_jobs=%d media_max_bytes=%d fixes=%s\n", cfg.Profile, cfg.Database.Dialect, sqliteJournalMode, sqliteForeignKeyViolations, mediaMode, backupTools, budget.CacheEntries, budget.QueueEntries, budget.WorkerConcurrency, deadLetteredJobs, budget.MediaMaxBytes, strings.Join(fixes, ","))
	return err
}

// validateCommand checks configuration and profile/module invariants without
// opening the database or exposing secret values. It is safe for CI and local
// editor/pre-commit workflows where dependencies may not be running yet.
func validateCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "generator" {
		return validateGeneratorCommand(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "api" {
		return validateAPICommand(args[1:], stdout, stderr)
	}
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	jsonOutput := flags.Bool("json", false, "Print machine-readable validation result")
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	modules := enabledModuleSet(cfg.Modules)
	if *jsonOutput {
		payload, err := json.Marshal(struct {
			Valid        bool     `json:"valid"`
			Profile      string   `json:"profile"`
			Database     string   `json:"database"`
			Modules      []string `json:"modules"`
			CacheEntries int      `json:"cache_entries"`
			QueueEntries int      `json:"queue_entries"`
		}{true, string(cfg.Profile), string(cfg.Database.Dialect), sortedModuleNames(modules), budget.CacheEntries, budget.QueueEntries})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(payload))
		return err
	}
	_, err = fmt.Fprintf(stdout, "valid profile=%s database=%s modules=%s cache_entries=%d queue_entries=%d\n", cfg.Profile, cfg.Database.Dialect, strings.Join(sortedModuleNames(modules), ","), budget.CacheEntries, budget.QueueEntries)
	return err
}

func validateAPICommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("validate api", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseline := flags.String("baseline", "", "Baseline OpenAPI JSON document")
	current := flags.String("current", "", "Current OpenAPI JSON document")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*baseline) == "" || strings.TrimSpace(*current) == "" {
		return errors.New("validate api requires --baseline and --current")
	}
	report, err := generator.CheckAPICompatibility(*baseline, *current)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "api-compatible paths=%d methods=%d properties=%d\n", report.PathsChecked, report.MethodsChecked, report.PropertiesChecked)
	return err
}

func validateGeneratorCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("validate generator", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json)")
	root := flags.String("root", "", "Generated project root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || strings.TrimSpace(*specPath) == "" || strings.TrimSpace(*root) == "" {
		return errors.New("validate generator requires --spec and --root")
	}
	spec, err := generator.LoadSpec(*specPath)
	if err != nil {
		return err
	}
	report, err := generator.CheckCompatibility(*root, spec)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "compatible collection=%s contract=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.ContractVersion, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked)
	return err
}

// testCommand runs the repository's Go tests with a bounded context. Arguments
// are passed directly to exec.CommandContext; no shell interpolation is used.
func testCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(stderr)
	packagePattern := flags.String("package", "./...", "Go package pattern to test")
	timeout := flags.Duration("timeout", 5*time.Minute, "Maximum test duration")
	race := flags.Bool("race", false, "Run the race detector")
	count := flags.Int("count", 1, "Number of test runs")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*packagePattern) == "" || *timeout <= 0 || *count <= 0 {
		return errors.New("test requires a non-empty package, positive timeout, and positive count")
	}
	commandArgs := []string{"test", "-count=" + strconv.Itoa(*count), "-timeout=" + timeout.String()}
	if *race {
		commandArgs = append(commandArgs, "-race")
	}
	commandArgs = append(commandArgs, *packagePattern)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "go", commandArgs...)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("go tests exceeded timeout %s", timeout.String())
		}
		return fmt.Errorf("go test failed: %w", err)
	}
	return nil
}

func sortedModuleNames(modules map[string]bool) []string {
	result := make([]string, 0, len(modules))
	for name, enabled := range modules {
		if enabled {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func jobsCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "dead-letter" {
		return errors.New("jobs requires the dead-letter subcommand")
	}
	if len(args) < 2 {
		return errors.New("jobs dead-letter requires list or requeue")
	}
	flags := flag.NewFlagSet("jobs dead-letter "+args[1], flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	switch args[1] {
	case "list":
		limit := flags.Int("limit", 50, "Maximum number of jobs to display")
		jsonOutput := flags.Bool("json", false, "Print machine-readable output")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		if flags.NArg() != 0 || *limit < 1 || *limit > 1000 {
			return errors.New("list requires no arguments and --limit between 1 and 1000")
		}
		cfg, budget, db, err := openJobsDatabase(flags, values)
		if err != nil {
			return err
		}
		defer db.Close()
		listed, err := (jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: budget.QueueEntries}).ListDeadLetters(context.Background(), db, *limit)
		if err != nil {
			return fmt.Errorf("list dead-letter jobs: %w", err)
		}
		if *jsonOutput {
			type deadLetterSummary struct {
				ID          string    `json:"id"`
				Type        string    `json:"type"`
				AvailableAt time.Time `json:"available_at"`
				CreatedAt   time.Time `json:"created_at"`
				Attempts    int       `json:"attempts"`
				LastError   string    `json:"last_error"`
			}
			summaries := make([]deadLetterSummary, 0, len(listed))
			for _, job := range listed {
				summaries = append(summaries, deadLetterSummary{ID: job.ID, Type: job.Type, AvailableAt: job.AvailableAt, CreatedAt: job.CreatedAt, Attempts: job.Attempts, LastError: job.LastError})
			}
			data, err := json.Marshal(summaries)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, string(data))
			return err
		}
		for _, job := range listed {
			if _, err := fmt.Fprintf(stdout, "dead-letter id=%s type=%s attempts=%d created=%s last_error=%q\n", job.ID, job.Type, job.Attempts, job.CreatedAt.Format(time.RFC3339), job.LastError); err != nil {
				return err
			}
		}
		return nil
	case "requeue":
		id := flags.String("id", "", "Dead-letter job ID")
		confirm := flags.Bool("confirm", false, "Confirm requeue of the selected job")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		if flags.NArg() != 0 || strings.TrimSpace(*id) == "" {
			return errors.New("requeue requires --id and no positional arguments")
		}
		if !*confirm {
			return errors.New("requeue requires --confirm")
		}
		cfg, budget, db, err := openJobsDatabase(flags, values)
		if err != nil {
			return err
		}
		defer db.Close()
		if err := (jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: budget.QueueEntries}).RequeueDeadLetter(context.Background(), db, *id, time.Now().UTC()); err != nil {
			return fmt.Errorf("requeue dead-letter job: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "dead-letter-requeued id=%s\n", *id)
		return err
	case "purge":
		olderThan := flags.Duration("older-than", 0, "Delete dead-letter jobs older than this duration, for example 720h")
		limit := flags.Int("limit", 100, "Maximum number of jobs to delete")
		confirm := flags.Bool("confirm", false, "Confirm deletion of the selected jobs")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		if flags.NArg() != 0 || *olderThan <= 0 || *limit < 1 || *limit > 10000 {
			return errors.New("purge requires --older-than, no positional arguments, and --limit between 1 and 10000")
		}
		if !*confirm {
			return errors.New("purge requires --confirm")
		}
		cfg, budget, db, err := openJobsDatabase(flags, values)
		if err != nil {
			return err
		}
		defer db.Close()
		removed, err := (jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: budget.QueueEntries}).PurgeDeadLetters(context.Background(), db, time.Now().UTC().Add(-*olderThan), *limit)
		if err != nil {
			return fmt.Errorf("purge dead-letter jobs: %w", err)
		}
		_, err = fmt.Fprintf(stdout, "dead-letter-purged count=%d\n", removed)
		return err
	default:
		return fmt.Errorf("unknown jobs dead-letter subcommand %q", args[1])
	}
}

func openJobsDatabase(flags *flag.FlagSet, values runtimeFlags) (config.Config, capacity.Budget, *sql.DB, error) {
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return config.Config{}, capacity.Budget{}, nil, err
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return config.Config{}, capacity.Budget{}, nil, err
	}
	db, err := database.OpenWithOptions(context.Background(), cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return config.Config{}, capacity.Budget{}, nil, fmt.Errorf("database: %w", err)
	}
	return cfg, budget, db, nil
}

func checkDirectoryWritable(directory string) error {
	probe, err := os.CreateTemp(directory, ".hydro-write-check-*")
	if err != nil {
		return err
	}
	path := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return os.Remove(path)
}

type expiredSessionCleaner interface {
	DeleteExpired(context.Context, time.Time) (int64, error)
}

func runSessionCleanup(ctx context.Context, cleaner expiredSessionCleaner, stderr io.Writer) {
	runSessionCleanupObserved(ctx, cleaner, stderr, nil)
}

type sessionCleanupObserver interface {
	RecordSessionCleanup(int64)
	RecordSessionCleanupError()
}

func runSessionCleanupObserved(ctx context.Context, cleaner expiredSessionCleaner, stderr io.Writer, observer sessionCleanupObserver) {
	if cleaner == nil {
		return
	}
	if stderr == nil {
		stderr = io.Discard
	}
	const interval = 5 * time.Minute
	run := func() {
		deleted, err := cleaner.DeleteExpired(ctx, time.Now().UTC())
		if err != nil {
			if ctx.Err() == nil {
				if observer != nil {
					observer.RecordSessionCleanupError()
				}
				_, _ = fmt.Fprintf(stderr, "hydro session cleanup: %v\n", err)
			}
			return
		}
		if observer != nil {
			observer.RecordSessionCleanup(deleted)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func runStaleMediaTempSweeper(ctx context.Context, mediaDir string, stderr io.Writer) {
	for {
		if _, err := storage.SweepStaleFiles(ctx, mediaDir, 24*time.Hour, time.Now()); err != nil && ctx.Err() == nil {
			_, _ = fmt.Fprintf(stderr, "hydro stale media temp cleanup: %v\n", err)
		}
		timer := time.NewTimer(6 * time.Hour)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
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

func refreshIPBlocklist(ctx context.Context, db *sql.DB, store securitystore.BanStore, list *security.IPBlocklist) error {
	entries, err := store.ListActive(ctx, db, time.Now().UTC(), 4096)
	if err != nil {
		return err
	}
	return list.Replace(entries)
}

func runIPBlocklistRefresh(ctx context.Context, db *sql.DB, store securitystore.BanStore, list *security.IPBlocklist) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshCtx, cancel := context.WithTimeout(ctx, time.Second)
			err := refreshIPBlocklist(refreshCtx, db, store, list)
			cancel()
			if err != nil {
				slog.Warn("security blocklist refresh failed", "error", err)
			}
		}
	}
}

func runNewsWorker(ctx context.Context, db database.Executor, worker newsservice.Worker, metrics *observability.Metrics, stderr io.Writer, concurrency int) {
	if concurrency < 1 {
		concurrency = 1
	}
	var workers sync.WaitGroup
	for index := 0; index < concurrency; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			runNewsWorkerLoop(ctx, worker, metrics, stderr, newsWorkerID(index))
		}(index)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	defer workers.Wait()
	for {
		now := time.Now().UTC()
		if err := worker.EnqueueDue(ctx, db, now); err != nil && ctx.Err() == nil {
			metrics.RecordWorkerEnqueueError()
			_, _ = fmt.Fprintf(stderr, "hydro news worker enqueue: %v\n", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func newsWorkerID(index int) string {
	return fmt.Sprintf("news-worker-%d", index)
}

func runNewsWorkerLoop(ctx context.Context, worker newsservice.Worker, metrics *observability.Metrics, stderr io.Writer, workerID string) {
	const idlePoll = 100 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		processed, err := worker.RunOnce(ctx, workerID, time.Now().UTC())
		if err != nil {
			metrics.RecordWorkerFailure()
			if ctx.Err() == nil {
				_, _ = fmt.Fprintf(stderr, "hydro news worker %s run: %v\n", workerID, err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(idlePoll):
			}
			continue
		}
		if processed {
			metrics.RecordWorkerProcessed()
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(idlePoll):
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

/*
func commerceRouterRoutes(routes []commercetransport.Route) []router.Route {
	result := make([]router.Route, 0, len(routes))
	for _, route := range routes {
		access := router.Public
		if route.Protected {
			access = router.Protected
		}
		result = append(result, router.Route{Pattern: route.Pattern, Access: access, Handler: route.Handler})
	}
	return result
}
*/

func publicHomePath(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}
	return "/public/pages/" + url.PathEscape(locale) + "/demo/html"
}

func configuredLocale(locale string, supported []string) string {
	locale = strings.TrimSpace(locale)
	for _, candidate := range supported {
		if strings.EqualFold(locale, candidate) {
			return candidate
		}
	}
	return ""
}

func cookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func requestedLocale(r *http.Request, supported []string, defaultLocale string) (string, bool) {
	if selected := configuredLocale(r.URL.Query().Get("locale"), supported); selected != "" {
		return selected, true
	}
	if selected := configuredLocale(cookieValue(r, "hydro_locale"), supported); selected != "" {
		return selected, false
	}
	return i18n.ResolveAcceptLanguage(r.Header.Get("Accept-Language"), supported, defaultLocale), false
}

func localeRedirect(w http.ResponseWriter, r *http.Request, locale string) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Add("Vary", "Accept-Language")
	w.Header().Add("Vary", "Cookie")
	http.Redirect(w, r, "/"+url.PathEscape(locale)+"/", http.StatusFound)
}

func technicalHomeWithLocales(locale string, locales []string, demoPath string, demoEnabled bool) string {
	markup := technicalHomeHTML(locale, demoPath, demoEnabled)
	if len(locales) < 2 {
		return markup
	}
	var picker bytes.Buffer
	if err := designsystem.LocalePicker(designsystem.LocalePickerProps{Locales: locales, Selected: configuredLocale(locale, locales)}).Render(context.Background(), &picker); err != nil {
		return markup
	}
	return strings.Replace(markup, "</body>", picker.String()+"</body>", 1)
}

// technicalHomeHTML is intentionally small: the default root is a runtime
// status surface, not a marketing page. Product-specific frontends can still
// be configured through site.home_path or opened from the optional demo seed.
func technicalHomeHTML(locale, demoPath string, demoEnabled bool, nonce ...string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "en"
	}
	styleNonceAttr := ""
	if len(nonce) > 0 && strings.TrimSpace(nonce[0]) != "" {
		styleNonceAttr = fmt.Sprintf(` nonce="%s"`, template.HTMLEscapeString(nonce[0]))
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="%s">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Welcome to HydroGo</title>
  <style%s>
    :root {
      --bg: #090a0f;
      --card: #0d1017;
      --card-raised: #121620;
      --border: #1e2533;
      --border-focus: #0ea5e9;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --primary: #0ea5e9;
      --primary-hover: #38bdf8;
      --btn-secondary: #121620;
      --btn-secondary-hover: #181f2c;
      --font-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      min-height: 100vh;
      background: var(--bg);
      color: var(--text);
      font-family: var(--font-sans);
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 24px;
      line-height: 1.5;
      -webkit-font-smoothing: antialiased;
      text-align: center;
      position: relative;
      overflow: hidden;
    }
    .hero-glow-sphere {
      position: absolute;
      top: 20%%;
      left: 50%%;
      transform: translate(-50%%, -50%%);
      width: 500px;
      height: 500px;
      background: radial-gradient(circle, rgba(14, 165, 233, 0.12) 0%%, transparent 65%%);
      border-radius: 50%%;
      filter: blur(60px);
      pointer-events: none;
      z-index: 0;
    }
    .landing-wrapper {
      max-width: 640px;
      width: 100%%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      position: relative;
      z-index: 1;
      animation: fadeIn 0.4s ease;
    }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(8px); }
      to { opacity: 1; transform: translateY(0); }
    }
    .brand-icon-box {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 72px;
      height: 72px;
      border-radius: 20px;
      background: linear-gradient(135deg, #0284c7 0%%, #0ea5e9 100%%);
      color: #ffffff;
      box-shadow: 0 8px 32px rgba(14, 165, 233, 0.35);
      margin-bottom: 28px;
    }
    .hero-title {
      font-size: clamp(2rem, 4.5vw, 2.75rem);
      font-weight: 600;
      letter-spacing: -0.015em;
      color: #ffffff;
      margin-bottom: 14px;
      line-height: 1.2;
    }
    .hero-subtitle {
      font-size: clamp(1rem, 2.5vw, 1.1rem);
      color: var(--text-muted);
      margin-bottom: 36px;
      font-weight: 400;
      max-width: 500px;
      line-height: 1.6;
    }
    .actions-group {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 14px;
      flex-wrap: wrap;
    }
    .btn-hero {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      padding: 12px 28px;
      font-size: 0.95rem;
      font-weight: 600;
      border-radius: 10px;
      text-decoration: none;
      transition: all 0.2s ease;
      cursor: pointer;
      min-width: 140px;
    }
    .btn-start {
      background: var(--primary);
      color: #ffffff;
      border: 1px solid var(--primary);
      box-shadow: 0 4px 16px rgba(14, 165, 233, 0.35);
    }
    .btn-start:hover {
      background: var(--primary-hover);
      border-color: var(--primary-hover);
      box-shadow: 0 6px 24px rgba(56, 189, 248, 0.5);
      transform: translateY(-1px);
    }
    .btn-docs {
      background: var(--btn-secondary);
      color: #ffffff;
      border: 1px solid var(--border);
    }
    .btn-docs:hover {
      background: var(--btn-secondary-hover);
      border-color: var(--border-focus);
      color: #ffffff;
      transform: translateY(-1px);
    }
  </style>
</head>
<body>
<div class="hero-glow-sphere"></div>
<main class="landing-wrapper">
  <div class="brand-icon-box">
    <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon></svg>
  </div>

  <h1 class="hero-title">Welcome to HydroGo</h1>
  <p class="hero-subtitle">Manage your content, memberships, and publishing workflows from one fast, secure workspace.</p>

  <div class="actions-group">
    <a href="/admin" class="btn-hero btn-start">Get Started</a>
    <a href="/docs" class="btn-hero btn-docs">Documentation</a>
  </div>
</main>
</body>
</html>`, template.HTMLEscapeString(locale), styleNonceAttr)
}

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
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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

func migrate(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	dryRun := flags.Bool("dry-run", false, "Report pending migrations without changing the schema")
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	if cfg.Database.Dialect == database.SQLite && !*dryRun {
		if err := database.RecoverSQLite(ctx, db); err != nil {
			return fmt.Errorf("SQLite recovery check failed: %w", err)
		}
	}
	if *dryRun {
		pending, err := database.PendingMigrations(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			_, err = fmt.Fprintln(stderr, "migration-plan no pending migrations")
			return err
		}
		_, err = fmt.Fprintf(stderr, "migration-plan pending=%s\n", strings.Join(pending, ","))
		return err
	}
	if cfg.Database.Dialect == database.SQLite {
		dbPath, err := sqliteDatabasePath(cfg.Database.DSN)
		if err != nil {
			return err
		}
		snapshot := dbPath + ".bak-" + time.Now().UTC().Format("20060102T150405.000000000Z")
		if err := backupstore.SnapshotSQLiteFile(ctx, db, snapshot); err != nil {
			return fmt.Errorf("create pre-migration snapshot: %w", err)
		}
		if stderr != nil {
			_, _ = fmt.Fprintf(stderr, "migration-snapshot-created path=%s\n", snapshot)
		}
	}
	return database.ApplyMigrationsForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
}

func sqliteDatabasePath(dsn string) (string, error) {
	value := strings.TrimSpace(dsn)
	if strings.HasPrefix(value, "file:") {
		value = strings.TrimPrefix(value, "file:")
	}
	// database.SQLiteDSN encodes an absolute path as a URI opaque part, which
	// prepends "/" before a Windows drive letter (file:/C:/path) so the value
	// parses as a valid file URI. filepath.Abs does not understand that
	// leading slash and would otherwise anchor it under the current drive,
	// producing a doubled "C:\C:\..." path.
	if runtime.GOOS == "windows" && len(value) >= 4 && value[0] == '/' && value[2] == ':' &&
		((value[1] >= 'a' && value[1] <= 'z') || (value[1] >= 'A' && value[1] <= 'Z')) {
		value = value[1:]
	}
	if index := strings.IndexByte(value, '?'); index >= 0 {
		query := value[index+1:]
		value = value[:index]
		if strings.Contains(query, "mode=memory") {
			return "", errors.New("automatic migration snapshots require a file-backed SQLite database")
		}
	}
	if value == "" || value == ":memory:" {
		return "", errors.New("automatic migration snapshots require a file-backed SQLite database")
	}
	return filepath.Abs(value)
}

func rollbackCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("rollback", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	snapshotFlag := flags.String("snapshot", "", "Pre-migration SQLite snapshot file (defaults to newest adjacent snapshot)")
	confirm := flags.Bool("confirm", false, "Confirm HydroGo is stopped and database writes may be rolled back")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if !*confirm {
		return errors.New("rollback requires --confirm after stopping HydroGo and reviewing the snapshot")
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	if cfg.Database.Dialect != database.SQLite {
		return errors.New("hydro rollback currently supports SQLite only")
	}
	dbPath, err := sqliteDatabasePath(cfg.Database.DSN)
	if err != nil {
		return err
	}
	snapshot := strings.TrimSpace(*snapshotFlag)
	if snapshot == "" {
		matches, err := filepath.Glob(dbPath + ".bak-*")
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return errors.New("no pre-migration snapshots found; pass --snapshot PATH")
		}
		sort.Strings(matches)
		snapshot = matches[len(matches)-1]
	}
	snapshot, err = filepath.Abs(snapshot)
	if err != nil {
		return err
	}
	if snapshot == dbPath {
		return errors.New("snapshot and active database paths must differ")
	}
	if err := validateSQLiteFile(snapshot); err != nil {
		return fmt.Errorf("snapshot verification failed: %w", err)
	}
	if err := validateSQLiteFile(dbPath); err != nil {
		return fmt.Errorf("active database verification failed: %w", err)
	}
	timestamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	previous := dbPath + ".before-rollback-" + timestamp
	temporary := dbPath + ".rollback-" + timestamp
	if err := copyRegularFile(snapshot, temporary); err != nil {
		return fmt.Errorf("stage rollback database: %w", err)
	}
	defer os.Remove(temporary)
	if err := validateSQLiteFile(temporary); err != nil {
		return fmt.Errorf("staged rollback database verification failed: %w", err)
	}
	if err := os.Rename(dbPath, previous); err != nil {
		return fmt.Errorf("preserve current database: %w", err)
	}
	var movedSidecars []struct{ source, destination string }
	for _, suffix := range []string{"-wal", "-shm"} {
		source, destination := dbPath+suffix, previous+suffix
		if err := os.Rename(source, destination); err == nil {
			movedSidecars = append(movedSidecars, struct{ source, destination string }{source, destination})
		} else if !errors.Is(err, os.ErrNotExist) {
			_ = os.Rename(previous, dbPath)
			for _, sidecar := range movedSidecars {
				_ = os.Rename(sidecar.destination, sidecar.source)
			}
			return fmt.Errorf("preserve SQLite sidecar: %w", err)
		}
	}
	if err := os.Rename(temporary, dbPath); err != nil {
		_ = os.Rename(previous, dbPath)
		for _, sidecar := range movedSidecars {
			_ = os.Rename(sidecar.destination, sidecar.source)
		}
		return fmt.Errorf("activate rollback database: %w", err)
	}
	if err := validateSQLiteFile(dbPath); err != nil {
		return fmt.Errorf("activated rollback database failed integrity check; prior database preserved at %s: %w", previous, err)
	}
	_, err = fmt.Fprintf(stdout, "rollback-complete database=%s snapshot=%s previous=%s\n", dbPath, snapshot, previous)
	return err
}

func validateSQLiteFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("database path must be a regular file")
	}
	db, err := database.Open(context.Background(), database.SQLite, path)
	if err != nil {
		return err
	}
	recoveryErr := database.RecoverSQLite(context.Background(), db)
	closeErr := db.Close()
	if recoveryErr != nil {
		return recoveryErr
	}
	return closeErr
}

func copyRegularFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("snapshot must be a regular file")
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	syncErr := output.Sync()
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(destination)
		return copyErr
	}
	if syncErr != nil {
		_ = os.Remove(destination)
		return syncErr
	}
	if closeErr != nil {
		_ = os.Remove(destination)
		return closeErr
	}
	return nil
}

func ipBanCommand(args []string, remove bool, stdout, stderr io.Writer) error {
	name := "ban"
	if remove {
		name = "unban"
	}
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	duration := flags.Duration("for", 24*time.Hour, "Temporary ban duration; zero means permanent")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("%s requires one IP address or CIDR", name)
	}
	value := strings.TrimSpace(flags.Arg(0))
	prefix, err := parseIPBanPrefix(value)
	if err != nil {
		return err
	}
	if *duration < 0 || *duration > 365*24*time.Hour {
		return errors.New("ban duration must be between zero and 365 days")
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	store := securitystore.BanStore{Dialect: cfg.Database.Dialect}
	if remove {
		err = store.Remove(ctx, db, prefix)
	} else {
		var expires time.Time
		if *duration > 0 {
			expires = time.Now().UTC().Add(*duration)
		}
		err = store.Put(ctx, db, prefix, expires)
	}
	if err != nil {
		return fmt.Errorf("update IP blocklist (run hydro migrate if this is a new database): %w", err)
	}
	verb := "banned"
	if remove {
		verb = "unbanned"
	}
	_, err = fmt.Fprintf(stdout, "ip-%s prefix=%s\n", verb, prefix)
	return err
}

func parseIPBanPrefix(value string) (netip.Prefix, error) {
	if prefix, err := netip.ParsePrefix(value); err == nil {
		return prefix.Masked(), nil
	}
	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, errors.New("invalid IP address or CIDR")
	}
	bits := 128
	if address.Is4() {
		bits = 32
	}
	return netip.PrefixFrom(address, bits), nil
}

func upgrade(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "check" {
		return errors.New("upgrade requires the check subcommand")
	}
	flags := flag.NewFlagSet("upgrade check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		return fmt.Errorf("upgrade is not ready: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "upgrade-ready profile=%s database=%s\n", cfg.Profile, cfg.Database.Dialect)
	return err
}

func seedDemo(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	flags.SetOutput(stderr)
	confirmDemo := flags.Bool("confirm-demo", false, "Confirm that this database is a local demo target")
	forceProd := flags.Bool("force-prod", false, "Allow demo writes when the deployment environment is production")
	count := flags.Int("count", 20, "Number of demo news records to create (1-200)")
	locale := flags.String("locale", "", "Demo content locale: tr or en (defaults to configured locale)")
	clean := flags.Bool("clean", false, "Remove HydroGo demo records before seeding")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*confirmDemo {
		return errors.New("seed requires --confirm-demo; use only on a disposable local database")
	}
	if *count < 1 || *count > 200 {
		return errors.New("--count must be between 1 and 200")
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	demoLocale := strings.TrimSpace(cfg.I18N.DefaultLocale)
	if demoLocale == "" {
		demoLocale = "tr"
	}
	if *locale != "" {
		demoLocale = strings.ToLower(strings.TrimSpace(*locale))
	}
	if demoLocale != "tr" && demoLocale != "en" {
		return errors.New("--locale must be tr or en")
	}
	if isProductionEnvironment() && !*forceProd {
		return errors.New("seed is disabled in production; pass --force-prod only for an intentional demo deployment")
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	if *clean {
		if err := cleanDemoData(ctx, db, cfg.Database.Dialect, cfg.Media.Dir); err != nil {
			return fmt.Errorf("clean demo data: %w", err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Demo member data is deliberately isolated from admin users. It gives the
	// public member portal a real, repeatable account flow without exposing any
	// administrator capability.
	demoMemberHash, err := auth.HashPassword("memberpass123", auth.DefaultArgon2Params)
	if err != nil {
		return fmt.Errorf("seed demo member password: %w", err)
	}
	if cfg.Database.Dialect == database.MySQL {
		if err := seedExecMySQL(ctx, db,
			`INSERT IGNORE INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES (?,?,?,1,?,?,?,?)`,
			"demo-member-portal", "member@hydrogo.local", demoMemberHash, now, now, now, now); err != nil {
			return fmt.Errorf("seed demo member: %w", err)
		}
	} else if err := seedExec(ctx, db, cfg.Database.Dialect,
		`INSERT INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES ($1,$2,$3,TRUE,$4,$5,$6,$7) ON CONFLICT (id) DO NOTHING`,
		`INSERT OR IGNORE INTO members (id,email,password_hash,active,email_verified_at,last_login_at,created_at,updated_at) VALUES (?,?,?,1,?,?,?,?)`,
		"demo-member-portal", "member@hydrogo.local", demoMemberHash, now, now, now, now); err != nil {
		return fmt.Errorf("seed demo member: %w", err)
	}

	if err := os.MkdirAll(cfg.Media.Dir, 0o750); err != nil {
		return fmt.Errorf("seed media directory: %w", err)
	}

	minUint8 := func(a, b uint8) uint8 {
		if a < b {
			return a
		}
		return b
	}

	seedMedia := func(id, filename string, width, height int, r, g, b uint8) error {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		// Rich dark background with radial mesh gradient
		cx, cy := width/2, height/2
		maxDist := math.Hypot(float64(cx), float64(cy))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dist := math.Hypot(float64(x-cx), float64(y-cy))
				glow := 1.0 - (dist / maxDist)
				if glow < 0 {
					glow = 0
				}
				// Dark background with subtle color tint
				bgR := uint8(float64(10) + float64(r)*0.35*glow)
				bgG := uint8(float64(12) + float64(g)*0.35*glow)
				bgB := uint8(float64(18) + float64(b)*0.45*glow)
				img.Set(x, y, color.RGBA{R: bgR, G: bgG, B: bgB, A: 255})
			}
		}

		// Technical grid pattern
		gridStep := 32
		if width > 500 {
			gridStep = 40
		}
		for x := 0; x < width; x += gridStep {
			for y := 0; y < height; y++ {
				if y%4 == 0 {
					orig := img.RGBAAt(x, y)
					img.Set(x, y, color.RGBA{R: minUint8(orig.R+15, 255), G: minUint8(orig.G+20, 255), B: minUint8(orig.B+30, 255), A: 255})
				}
			}
		}
		for y := 0; y < height; y += gridStep {
			for x := 0; x < width; x++ {
				if x%4 == 0 {
					orig := img.RGBAAt(x, y)
					img.Set(x, y, color.RGBA{R: minUint8(orig.R+15, 255), G: minUint8(orig.G+20, 255), B: minUint8(orig.B+30, 255), A: 255})
				}
			}
		}

		// Central architectural badge card
		cardW, cardH := width*6/10, height*5/10
		if cardW < 120 {
			cardW = 120
		}
		if cardH < 80 {
			cardH = 80
		}
		cardRect := image.Rect(cx-cardW/2, cy-cardH/2, cx+cardW/2, cy+cardH/2)
		draw.Draw(img, cardRect, &image.Uniform{color.RGBA{R: 20, G: 26, B: 38, A: 220}}, image.Point{}, draw.Over)

		// Border of card
		for x := cardRect.Min.X; x <= cardRect.Max.X; x++ {
			img.Set(x, cardRect.Min.Y, color.RGBA{r, g, b, 180})
			img.Set(x, cardRect.Max.Y, color.RGBA{r, g, b, 100})
		}
		for y := cardRect.Min.Y; y <= cardRect.Max.Y; y++ {
			img.Set(cardRect.Min.X, y, color.RGBA{r, g, b, 180})
			img.Set(cardRect.Max.X, y, color.RGBA{r, g, b, 100})
		}

		// Decorative tech circles / nodes
		nodeRadius := 8
		if width < 500 {
			nodeRadius = 5
		}
		for _, nodeX := range []int{cardRect.Min.X + 24, cx, cardRect.Max.X - 24} {
			for dy := -nodeRadius; dy <= nodeRadius; dy++ {
				for dx := -nodeRadius; dx <= nodeRadius; dx++ {
					if dx*dx+dy*dy <= nodeRadius*nodeRadius {
						img.Set(nodeX+dx, cy+dy, color.RGBA{R: minUint8(r+60, 255), G: minUint8(g+60, 255), B: minUint8(b+60, 255), A: 255})
					}
				}
			}
		}

		// Connecting line between nodes
		for x := cardRect.Min.X + 24; x <= cardRect.Max.X-24; x++ {
			img.Set(x, cy, color.RGBA{R: minUint8(r+80, 255), G: minUint8(g+80, 255), B: minUint8(b+80, 255), A: 180})
			img.Set(x, cy+1, color.RGBA{R: minUint8(r+80, 255), G: minUint8(g+80, 255), B: minUint8(b+80, 255), A: 100})
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return fmt.Errorf("encode seed image %s: %w", id, err)
		}
		payload := buf.Bytes()
		hasher := sha256.New()
		hasher.Write(payload)
		storageKey := hex.EncodeToString(hasher.Sum(nil)) + ".png"
		filePath := filepath.Join(cfg.Media.Dir, storageKey)
		if err := os.WriteFile(filePath, payload, 0o600); err != nil {
			return fmt.Errorf("write seed media file %s: %w", filePath, err)
		}
		sizeBytes := int64(len(payload))
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES (?,?,?,'image/png',?,?,?,'public','ready',?,?)`, id, storageKey, filename, sizeBytes, width, height, now, now); err != nil {
				return fmt.Errorf("seed media asset %s: %w", id, err)
			}
			if err := seedExecMySQL(ctx, db, `UPDATE media_assets SET storage_key=?,original_filename=?,size_bytes=?,width=?,height=?,visibility='public',status='ready',updated_at=? WHERE id=?`, storageKey, filename, sizeBytes, width, height, now, id); err != nil {
				return err
			}
		} else {
			if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES ($1,$2,$3,'image/png',$4,$5,$6,'public','ready',$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES (?1,?2,?3,'image/png',?4,?5,?6,'public','ready',?7,?7)`, id, storageKey, filename, sizeBytes, width, height, now); err != nil {
				return fmt.Errorf("seed media asset %s: %w", id, err)
			}
			if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE media_assets SET storage_key=$1,original_filename=$2,size_bytes=$3,width=$4,height=$5,visibility='public',status='ready',updated_at=$6 WHERE id=$7`, `UPDATE media_assets SET storage_key=?1,original_filename=?2,size_bytes=?3,width=?4,height=?5,visibility='public',status='ready',updated_at=?6 WHERE id=?7`, storageKey, filename, sizeBytes, width, height, now, id); err != nil {
				return err
			}
		}
		var storedKey string
		query := `SELECT storage_key FROM media_assets WHERE id=$1`
		if cfg.Database.Dialect == database.SQLite {
			query = `SELECT storage_key FROM media_assets WHERE id=?1`
		} else if cfg.Database.Dialect == database.MySQL {
			query = `SELECT storage_key FROM media_assets WHERE id=?`
		}
		if err := db.QueryRowContext(ctx, query, id).Scan(&storedKey); err != nil {
			return fmt.Errorf("seed media asset %s was not persisted: %w", id, err)
		}
		if storedKey != storageKey {
			return fmt.Errorf("seed media asset %s storage key collision: got %q, want %q", id, storedKey, storageKey)
		}
		return nil
	}

	mediaFixtures := []struct {
		id, filename string
		w, h         int
		r, g, b      uint8
	}{
		{"demo-media-hero-home", "hero-home.png", 800, 450, 21, 94, 239},
		{"demo-media-hero-showcase", "hero-showcase.png", 800, 450, 15, 23, 42},
		{"demo-media-news-1", "news-2-0.png", 800, 450, 37, 99, 235},
		{"demo-media-news-2", "news-guide.png", 800, 450, 14, 116, 144},
		// Keep the generated payload distinct from hero-showcase; storage keys
		// are content hashes and duplicate bytes would be ignored by the
		// unique-key constraint, silently dropping this fixture.
		{"demo-media-news-3", "news-security.png", 800, 450, 16, 24, 43},
		{"demo-media-news-4", "news-db.png", 800, 450, 79, 70, 229},
		{"demo-media-news-5", "news-jobs.png", 800, 450, 5, 150, 105},
		{"demo-media-news-6", "news-cache.png", 800, 450, 217, 119, 6},
		{"demo-media-news-7", "news-theme.png", 800, 450, 192, 38, 211},
		{"demo-media-news-8", "news-commerce.png", 800, 450, 225, 29, 72},
		{"demo-media-prod-1", "hydrogo-starter.png", 400, 300, 30, 41, 59},
		{"demo-media-prod-2", "hydrogo-growth.png", 400, 300, 37, 99, 235},
		{"demo-media-prod-3", "hydrogo-enterprise.png", 400, 300, 15, 23, 42},
		{"demo-media-prod-4", "hydro-bottle.png", 400, 300, 2, 132, 199},
		{"demo-media-prod-5", "hydro-shirt.png", 400, 300, 71, 85, 105},
		{"demo-media-prod-6", "hydro-keyboard.png", 400, 300, 51, 65, 85},
		{"demo-media-prod-7", "hydro-backpack.png", 400, 300, 79, 70, 229},
		{"demo-media-prod-8", "hydro-edge-hub.png", 400, 300, 16, 185, 129},
	}
	for _, mf := range mediaFixtures {
		if err := seedMedia(mf.id, mf.filename, mf.w, mf.h, mf.r, mf.g, mf.b); err != nil {
			return fmt.Errorf("seed media %s: %w", mf.id, err)
		}
	}

	seedPage := func(id, slug, title, body, blocks string) error {
		if demoLocale == "en" {
			switch id {
			case "demo-home":
				title, body = "HydroGo Platform", "Manage your site pages, articles, and media from one workspace."
				blocks = `[{"type":"hero","payload":{"title":"Manage your site in one place","subtitle":"A focused workspace for pages, news, and media.","media_asset_id":"demo-media-hero-home","button_text":"Explore the examples","button_url":"/public/pages/en/showcase/html"}},{"type":"faq","payload":{"items":[{"question":"What can I manage?","answer":"Manage pages, news, and media. Additional features depend on enabled modules."},{"question":"Is this production content?","answer":"No. This page contains demo content for evaluation."}]}}]`
			case "demo-showcase":
				title, body = "HydroGo Showcase", "Examples of pages and reusable content blocks."
				blocks = `[{"type":"hero","payload":{"title":"HydroGo component examples","subtitle":"A sample gallery of reusable page blocks.","media_asset_id":"demo-media-hero-showcase","button_text":"Browse articles","button_url":"/public/news/en/html"}},{"type":"faq","payload":{"items":[{"question":"What does this page show?","answer":"It shows how reusable page blocks appear in a published theme."},{"question":"Are these commercial terms?","answer":"No. Demo content does not define prices or service commitments."}]}}]`
			case "demo-about":
				title, body = "About HydroGo", "A configurable application core for modular content and commerce projects."
				blocks = `[{"type":"hero","payload":{"title":"HydroGo architecture","subtitle":"A modular foundation for content and commerce.","media_asset_id":"demo-media-hero-about","button_text":"Read the updates","button_url":"/public/news/en/html"}},{"type":"faq","payload":{"items":[{"question":"Can modules be enabled separately?","answer":"Editions choose the modules they need."},{"question":"Is this a commercial offer?","answer":"No. Demo content is illustrative only."}]}}]`
			}
		}
		localePath := url.PathEscape(demoLocale)
		blocks = strings.ReplaceAll(strings.ReplaceAll(blocks, "/public/pages/tr/", "/public/pages/"+localePath+"/"), "/public/news/tr/", "/public/news/"+localePath+"/")
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES (?,?,1,?,?,?,'published',1,1,1,?,?,?,?)`, id, title, demoLocale, slug, body, blocks, blocks, now, now); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES ($1,$2,1,$3,$4,$5,'published',1,1,1,$6,$6,$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES (?1,?2,1,?3,?4,?5,'published',1,1,1,?6,?6,?7,?7)`, id, title, demoLocale, slug, body, blocks, now); err != nil {
			return err
		}
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `UPDATE pages SET locale=?,slug=?,title=?,body=?,blocks=?,published_locale=?,published_slug=?,published_title=?,published_body=?,published_blocks=? WHERE id=?`, demoLocale, slug, title, body, blocks, demoLocale, slug, title, body, blocks, id); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE pages SET locale=$1,slug=$2,title=$3,body=$4,blocks=$5,published_locale=$1,published_slug=$2,published_title=$3,published_body=$4,published_blocks=$5 WHERE id=$6`, `UPDATE pages SET locale=?1,slug=?2,title=?3,body=?4,blocks=?5,published_locale=?1,published_slug=?2,published_title=?3,published_body=?4,published_blocks=?5 WHERE id=?6`, demoLocale, slug, title, body, blocks, id); err != nil {
			return err
		}
		if cfg.Database.Dialect == database.MySQL {
			return seedExecMySQL(ctx, db, `INSERT IGNORE INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES (?,?,1,?,?,?,?,?,?)`, id+"-rev-1", id, demoLocale, slug, title, body, blocks, now)
		}
		return seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES ($1,$2,1,$3,$4,$5,$6,$7,$8) ON CONFLICT (page_id,revision) DO NOTHING`, `INSERT OR IGNORE INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES (?1,?2,1,?3,?4,?5,?6,?7,?8)`, id+"-rev-1", id, demoLocale, slug, title, body, blocks, now)
	}

	homeBlocks := `[{"type":"hero","payload":{"title":"İçeriğinizi ve mağazanızı tek yerde yönetin","subtitle":"Sayfalar, haberler, medya ve ürün kataloğu için sade bir yönetim deneyimi.","media_asset_id":"demo-media-hero-home","button_text":"Yetenekleri incele →","button_url":"/public/pages/tr/showcase/html"}},{"type":"faq","payload":{"items":[{"question":"Hangi içerikleri yönetebilirim?","answer":"Sayfalar, haberler, medya varlıkları ve Commerce modülü açıksa ürünler, varyantlar ve siparişler yönetilebilir."},{"question":"SQLite ve PostgreSQL destekleniyor mu?","answer":"Her iki veritabanı için aynı davranış sözleşmeleri ve migration akışları bulunur; deployment öncesi ortamınıza uygun doğrulama yapılmalıdır."},{"question":"Üye hesabı nasıl çalışır?","answer":"Üyeler public frontend üzerinden kayıt olabilir, giriş yapabilir, sepet ve kendi sipariş geçmişlerini görebilir. Admin Studio erişimleri ayrıdır."},{"question":"Ödeme canlı mı?","answer":"Demo ortamında ödeme sağlayıcı akışı yalnızca yapılandırılmış sandbox veya test sağlayıcısı ile çalışır; canlı anahtarlar ayrıca kurulmalıdır."}]}},{"type":"cta","payload":{"title":"Kendi içerik akışınızı oluşturun","button_text":"Studio'ya git →","button_url":"/admin"}}]`
	showcaseBlocks := `[{"type":"hero","payload":{"title":"HydroGo bileşen örnekleri","subtitle":"Bu sayfa, kurulumla gelen örnek blokların bir vitrini; ticari teklif, müşteri yorumu veya hizmet taahhüdü değildir.","media_asset_id":"demo-media-hero-showcase","button_text":"Mağaza örneğini incele →","button_url":"/shop"}},{"type":"faq","payload":{"items":[{"question":"Bu sayfa neyi gösterir?","answer":"Hero, CTA ve SSS gibi yayınlanabilir sayfa bloklarının tema tarafından nasıl sunulduğunu gösterir."},{"question":"Fiyatlar veya hizmet koşulları burada tanımlı mı?","answer":"Hayır. Ticari fiyat, sözleşme, SLA ve hukuki koşullar ilgili işletme tarafından ayrıca yayınlanmalıdır."},{"question":"Özel bloklar nasıl eklenir?","answer":"Tema slotları ve sürümlü modül sözleşmeleri üzerinden, proje gereksinimine göre geliştirilir."}]}},{"type":"cta","payload":{"title":"Yayınlanan içerik örneklerini inceleyin","button_text":"Haberlere göz at →","button_url":"/public/news/tr/html"}}]`
	aboutBlocks := `[{"type":"hero","payload":{"title":"HydroGo Vizyonu ve Mimari Felsefesi","subtitle":"Modüler içerik ve ticaret ihtiyaçları için yapılandırılabilir bir uygulama çekirdeği.","media_asset_id":"demo-media-hero-about","button_text":"Haber Merkezini Gör →","button_url":"/public/news/tr/html"}},{"type":"faq","payload":{"items":[{"question":"Modüler CMS + Commerce yaklaşımı nedir?","answer":"HydroCore temel sözleşmeleri tanımlar. Pages, News ve Commerce modülleri gerektiğinde etkinleştirilebilir."},{"question":"Güvenlik nasıl ele alınır?","answer":"Kimlik, oturum, CSRF ve içerik güvenliği kontrolleri yapılandırılan modüllerde uygulanır. Canlıya çıkmadan önce ortam ve entegrasyon doğrulaması yapılmalıdır."},{"question":"Headless ve API desteği var mı?","answer":"Etkin API modülleri REST ve GraphQL uç noktaları sunar; kullanılabilirlik kurulum yapılandırmasına bağlıdır."}]}},{"type":"cta","payload":{"title":"Geliştirici veya İçerik Yöneticisi misiniz?","button_text":"Studio Çalışma Alanı →","button_url":"/admin"}}]`

	if err := seedPage("demo-home", "demo", "HydroGo Platformu", "HydroGo kurumsal içerik yönetim ve e-ticaret platformuna hoş geldiniz.", homeBlocks); err != nil {
		return fmt.Errorf("seed demo home: %w", err)
	}
	if err := seedPage("demo-showcase", "showcase", "HydroGo Showcase", "Tüm blok tiplerini ve mimari yetenekleri keşfedin.", showcaseBlocks); err != nil {
		return fmt.Errorf("seed demo showcase: %w", err)
	}
	if err := seedPage("demo-about", "about", "HydroGo Hakkında", "HydroCore, modüller ve Client Edition yaklaşımı.", aboutBlocks); err != nil {
		return fmt.Errorf("seed demo about: %w", err)
	}

	newsList := []demoNewsSeed{
		{
			id:           "demo-news-release-v2",
			slug:         "hydrogo-2-0-yayinda",
			title:        "HydroGo 2.0 Yayında: Zengin Editör, Gelişmiş Güvenlik ve Modern UI",
			excerpt:      "Yeni sürüm ile birlikte Quill tabanlı zengin metin editörü, key takeaways ve etiket yönetimi, CSP sıkılaştırması ve baştan tasarlanan arayüz kullanıma sunuldu.",
			takeaways:    "- Quill tabanlı zengin metin editörü tüm biçimlendirme araçlarıyla entegre edildi.\n- XSS ve SVG data URL vektörlerine karşı çok katmanlı sanitizasyon getirildi.\n- Kategori, etiket ve öne çıkan içerik rozetleri public haber odasına eklendi.\n- Dev server ve Admin Studio modern mavi tasarım sistemine kavuştu.",
			category:     "Ürün Güncellemeleri",
			author:       "HydroGo Editoryal",
			coverMediaID: "demo-media-news-1",
			featured:     true,
			tags:         "hydrogo, release, security, richtext, go",
			body:         `{"ops":[{"insert":"HydroGo 2.0 İle İçerik Yönetimi Yeni Bir Boyuta Taşınıyor\n","attributes":{"header":1}},{"insert":"HydroGo ekibi olarak, platformun en kapsamlı güncellemesi olan 2.0 sürümünü duyurmaktan büyük heyecan duyuyoruz. Bu sürümde editoryal deneyimi güçlendirirken kurumsal güvenlik gereksinimlerini en üst seviyede karşılamayı hedefledik.\n\n"},{"insert":"Öne Çıkan Başlıca Yenilikler:\n","attributes":{"header":2}},{"insert":"Zengin Metin Editörü: ","attributes":{"bold":true}},{"insert":"Başlık hiyerarşisi, kalın/italik/altı çizili stiller, kod blokları, alıntılar ve medya entegrasyonu.\n"},{"insert":"Editoryal Metadata: ","attributes":{"bold":true}},{"insert":"Haber kategorileri, etiketler, yazar bilgisi, özet ve önemli çıkarımlar (Key Takeaways).\n"},{"insert":"Güvenlik Sertleştirmesi: ","attributes":{"bold":true}},{"insert":"Sıkı Content Security Policy (CSP), CSRF token koruması ve XSS temizleme filtreleri.\n\n"},{"insert":"\"Güvenilir, tip güvenli ve modern bir CMS deneyimi için HydroCore mimarisini her geçen gün daha da ileri taşıyoruz.\"\n","attributes":{"blockquote":true}},{"insert":"Geliştirici deneyimini keşfetmek için Admin Studio'ya giriş yapabilir veya API dokümantasyonunu inceleyebilirsiniz.\n"}]}`,
		},
		{
			id:           "demo-news-guide",
			slug:         "hydrogo-ile-baslangic-rehberi",
			title:        "Geliştiriciler İçin HydroGo Başlangıç ve Entegrasyon Rehberi",
			excerpt:      "Admin Studio üzerinden içerik ve e-ticaret kataloğu yönetimi, REST ve GraphQL API uç noktalarının tüketimi hakkında adım adım kılavuz.",
			takeaways:    "- Admin Studio içerisinden Sayfalar, Haberler ve Commerce ürünleri yönetilebilir.\n- REST API üzerinden tüm yayınlanmış verilere anında JSON formatında erişilir.\n- Headless starter'lar bağımsız frontend mimarileri kurmayı kolaylaştırır.",
			category:     "Rehberler",
			author:       "HydroGo Core Team",
			coverMediaID: "demo-media-news-2",
			featured:     false,
			tags:         "guide, tutorial, api, headless, cms",
			body:         `{"ops":[{"insert":"HydroGo Mimarisine Hızlı Bir Bakış\n","attributes":{"header":1}},{"insert":"HydroGo, modern web uygulamaları ve e-ticaret projeleri için geliştirilmiş yüksek performanslı bir Go CMS çekirdeğidir.\n\n"},{"insert":"1. Adım: Admin Studio ile İçerik Oluşturma\n","attributes":{"header":2}},{"insert":"Admin paneline erişerek yeni sayfalar oluşturabilir, hero, CTA, FAQ gibi blokları sürükleyip yapılandırabilir ve haberlerinizi yayına alabilirsiniz.\n\n"},{"insert":"2. Adım: API Uç Noktalarını Bağlama\n","attributes":{"header":2}},{"insert":"Frontend uygulamanızdan ","attributes":{}},{"insert":"/api/pages","attributes":{"code":true}},{"insert":" veya ","attributes":{}},{"insert":"/api/commerce/products","attributes":{"code":true}},{"insert":" adreslerine istek atarak içerikleri doğrudan tüketebilirsiniz.\n"}]}`,
		},
		{
			id:           "demo-news-security",
			slug:         "kurumsal-seviyede-cms-guvenligi",
			title:        "HydroGo Güvenlik Mimarisi: Çok Katmanlı Koruma ve Sıfır Sızıntı",
			excerpt:      "Argon2id parola şifreleme, TOTP MFA, CSP nonces ve SQL enjeksiyonlarına karşı parameterized query standartları.",
			takeaways:    "- Tüm formlar dinamik CSRF token ve zaman aşımı korumalıdır.\n- Parolalar endüstri standardı Argon2id ile hash'lenir.\n- Çift veritabanı desteği (PostgreSQL & SQLite) kontrat testleri ile korunur.",
			category:     "Güvenlik & Mimari",
			author:       "Güvenlik Mühendisliği",
			coverMediaID: "demo-media-news-3",
			featured:     false,
			tags:         "security, mfa, argon2id, csp, database",
			body:         `{"ops":[{"insert":"Kurumsal Güvenlik Standartlarımız\n","attributes":{"header":1}},{"insert":"HydroGo, finansal ve hassas veri taşıyan uygulamalarda güvenle çalışacak şekilde tasarlanmıştır. Kimlik doğrulama, yetkilendirme ve veri yaşam döngüsü kuralları ADR-0013 ve ADR-0014 standartlarına uyum hedefiyle tasarlanmıştır.\n\n"},{"insert":"Denetim ve İzlenebilirlik:\n","attributes":{"header":2}},{"insert":"Tüm kritik yönetici hareketleri (içerik yayınlama, kullanıcı davetleri, iade işlemleri) değiştirilemez audit log tablosunda kayıt altına alınır.\n"}]}`,
		},
		{
			id:           "demo-news-db-portability",
			slug:         "veritabani-tasinabilirligi-postgresql-sqlite",
			title:        "Veritabanı Taşınabilirliği: SQLite'tan PostgreSQL'e Kesintisiz Geçiş",
			excerpt:      "Dialect bağımsız repository katmanı ve SQL injection riskini azaltan parameterized query standartları.",
			takeaways:    "- Kod tabanında dialect sızıntısına izin verilmez.\n- SQLite yerel geliştirme ve kompakt profil için idealdir.\n- PostgreSQL yüksek eşzamanlılık ve kümeleme desteği sunar.\n- Çift veritabanı davranış testleri her CI adımında doğrulanır.",
			category:     "Veritabanı",
			author:       "Veri Altyapı Ekibi",
			coverMediaID: "demo-media-news-4",
			featured:     false,
			tags:         "database, postgresql, sqlite, migrations, adr0002",
			body:         `{"ops":[{"insert":"Dialect Bağımsız Veri Katmanı Mimarisi\n","attributes":{"header":1}},{"insert":"HydroGo'da tüm SQL sorguları ve migrasyonlar hem SQLite hem de PostgreSQL motorlarında eşit seviyede test edilir. Bu sayede geliştiriciler yerel ortamda SQLite ile sıfır kurulumla başlarken prodüksiyonda PostgreSQL'e güvenle geçebilirler.\n\n"},{"insert":"Sıfır Dialect Sızıntısı Prensibi:\n","attributes":{"header":2}},{"insert":"Domain ve servis katmanları veritabanı sürücülerine doğrudan bağımlı değildir; transactor ve repository arayüzleri üzerinden soyutlanmıştır.\n"}]}`,
		},
		{
			id:           "demo-news-job-engine",
			slug:         "arka-plan-is-yonetimi-ve-outbox-pattern",
			title:        "Arka Plan İş Yönetimi: Güvenilir Olay Dağıtımı ve Dead Letter Kuyrukları",
			excerpt:      "Outbox pattern ile veritabanı işlemlerini atomik olarak olaylara dönüştüren, yeniden deneme ve hata kurtarma yetenekli worker motoru.",
			takeaways:    "- Veritabanı işlemi ve olay kaydı aynı transaction içinde tamamlanır.\n- Başarısız işler üstel geri çekilme ile yeniden denenir.\n- Çözülemeyen işler dead-letter kuyruğuna alınır ve admin tarafından incelenebilir.",
			category:     "Mühendislik",
			author:       "Sistem Mühendisliği",
			coverMediaID: "demo-media-news-5",
			featured:     false,
			tags:         "jobs, outbox, workers, reliability, queues",
			body:         `{"ops":[{"insert":"Güvenilir Asenkron İşleme: Outbox Pattern\n","attributes":{"header":1}},{"insert":"HydroGo'nun arka plan iş motoru, e-posta gönderimi, webhook tetiklemeleri ve arama indeksi güncellemeleri gibi operasyonları ana HTTP iş parçacığından ayırarak web yanıt sürelerini milisaniyeler seviyesinde tutar.\n"}]}`,
		},
		{
			id:           "demo-news-cache-strategy",
			slug:         "onbellekleme-stratejileri-ve-damgalanma-korumasi",
			title:        "Önbellekleme Stratejileri: Stampede Koruması ve Çok Katmanlı Geçersiz Kılma",
			excerpt:      "Yüksek trafikli sayfa ve katalog sorgularında bellek içi önbellekleme, singleflight stampede koruması ve otomatik invalidation.",
			takeaways:    "- Singleflight ile eşzamanlı sorgu birleştirme uygulanır.\n- İçerik yayınlandığında ilgili önbellek anahtarları anında düşürülür.\n- Kompakt profilde bellek sınırı aşılmayacak şekilde bounded cache kullanılır.",
			category:     "Performans",
			author:       "Performans Ekibi",
			coverMediaID: "demo-media-news-6",
			featured:     false,
			tags:         "cache, performance, singleflight, memory, scale",
			body:         `{"ops":[{"insert":"Yüksek Trafik Altında Dayanıklı Önbellekleme\n","attributes":{"header":1}},{"insert":"Cache stampede problemi, aynı anda gelen binlerce isteğin veritabanını kilitlemesine yol açabilir. HydroGo'nun yerleşik önbellek katmanı bu durumu singleflight mekanizması ile tamamen engeller.\n"}]}`,
		},
		{
			id:           "demo-news-theme-engine",
			slug:         "dosya-sistemi-tabanli-tema-motoru-ve-slotlar",
			title:        "Dosya Sistemi Tabanlı Tema Motoru: HTML Şablonları ve Dinamik Slotlar",
			excerpt:      "Harici şablon derleyicilerine ihtiyaç duymadan Go'nun güvenli html/template motoru ile tam özelleştirilebilir tema altyapısı.",
			takeaways:    "- Temalar layouts ve blocks klasör yapısı ile düzenlenir.\n- Dinamik slot mekanizması içerik bloklarının esnek yerleşimini sağlar.\n- Güvenli HTML çıktısı ve CSP uyumu korunur.",
			category:     "Frontend & Tasarım",
			author:       "Tasarım Ekibi",
			coverMediaID: "demo-media-news-7",
			featured:     false,
			tags:         "theme, templates, slots, blocks, frontend",
			body:         `{"ops":[{"insert":"Esnek ve Güvenli Tema Geliştirme\n","attributes":{"header":1}},{"insert":"HydroGo tema motoru, kurumsal markaların kurumsal kimliklerine uygun arayüzler oluşturmasını kolaylaştırır. themes/corporate klasörü altındaki şablonlar sıfır derleme adımı ile anında güncellenir.\n"}]}`,
		},
		{
			id:           "demo-news-commerce-pilot",
			slug:         "e-ticaret-ve-odeme-saglayici-sozlesmeleri",
			title:        "E-Ticaret Mimarisi: Varyantlar, Stok Rezervasyonu ve Ödeme Entegrasyonları",
			excerpt:      "Stripe, iyzico ve PayTR entegrasyonlarını destekleyen sipariş, sepet ve ödeme provizyon akışları.",
			takeaways:    "- Renk, beden ve lisans bazlı esnek varyant desteği.\n- Stok tükenmesi durumunda sepet seviyesinde koruma.\n- Idempotent sipariş oluşturma ve güvenli ödeme intent akışı.",
			category:     "E-Ticaret",
			author:       "Commerce Ekibi",
			coverMediaID: "demo-media-news-8",
			featured:     false,
			tags:         "commerce, products, variants, payments, shop",
			body:         `{"ops":[{"insert":"Güvenilir ve Esnek E-Ticaret Altyapısı\n","attributes":{"header":1}},{"insert":"Commerce modülü modern e-ticaret operasyonlarının tüm ihtiyaçlarını karşılar. Müşteriler sepete ürün ekleyebilir, varyant seçebilir ve güvenli ödeme adımlarını tamamlayabilir.\n"}]}`,
		},
	}
	if *count < len(newsList) {
		newsList = newsList[:*count]
	} else {
		baseCount := len(newsList)
		for i := baseCount; i < *count; i++ {
			item := newsList[(i-baseCount)%baseCount]
			item.id = fmt.Sprintf("demo-news-generated-%03d", i+1)
			item.slug = fmt.Sprintf("demo-article-%03d", i+1)
			item.title = fmt.Sprintf("%s · Örnek %02d", item.title, i+1)
			item.excerpt = fmt.Sprintf("%s Bu içerik demo kataloğunu göstermek için üretildi.", item.excerpt)
			item.featured = false
			newsList = append(newsList, item)
		}
	}
	if demoLocale == "en" {
		for i := range newsList {
			newsList[i] = localizeDemoNews(newsList[i], i+1)
		}
	}

	for _, item := range newsList {
		featuredInt := 0
		if item.featured {
			featuredInt = 1
		}
		var featuredValue any = featuredInt
		if cfg.Database.Dialect == database.Postgres {
			featuredValue = item.featured
		}
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,'published',1,1,1,0,?,?)`, item.id, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, now, now); err != nil {
				return fmt.Errorf("seed news %s: %w", item.id, err)
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'published',1,1,1,0,$13,$13) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,'published',1,1,1,0,?13,?13)`, item.id, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, now); err != nil {
			return fmt.Errorf("seed news %s: %w", item.id, err)
		}
		if cfg.Database.Dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db, `UPDATE news_items SET locale=?,slug=?,title=?,excerpt=?,key_takeaways=?,category=?,author_name=?,cover_media_id=?,featured=?,tags=?,body=?,status='published',published_revision=revision WHERE id=?`, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, item.id); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE news_items SET locale=$1,slug=$2,title=$3,excerpt=$4,key_takeaways=$5,category=$6,author_name=$7,cover_media_id=$8,featured=$9,tags=$10,body=$11,status='published',published_revision=revision WHERE id=$12`, `UPDATE news_items SET locale=?1,slug=?2,title=?3,excerpt=?4,key_takeaways=?5,category=?6,author_name=?7,cover_media_id=?8,featured=?9,tags=?10,body=?11,status='published',published_revision=revision WHERE id=?12`, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredValue, item.tags, item.body, item.id); err != nil {
			return fmt.Errorf("publish news %s: %w", item.id, err)
		}
	}
	if err := seedDemoTaxonomy(ctx, db, cfg.Database.Dialect, newsList); err != nil {
		return fmt.Errorf("seed demo categories: %w", err)
	}
	if cfg.Database.Dialect == database.MySQL {
		if err := seedExecMySQL(ctx, db, `INSERT IGNORE INTO site_settings (`+"`key`"+`,`+"`value`"+`,is_public,updated_at) VALUES (?,?,1,?)`, "site.title", "HydroGo Demo", now); err != nil {
			return fmt.Errorf("seed setting: %w", err)
		}
	} else if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO site_settings (key,value,is_public,updated_at) VALUES ($1,$2,TRUE,$3) ON CONFLICT (tenant_id,key) DO NOTHING`, `INSERT OR IGNORE INTO site_settings (key,value,is_public,updated_at) VALUES (?,?,1,?)`, "site.title", "HydroGo Demo", now); err != nil {
		return fmt.Errorf("seed setting: %w", err)
	}
	if cfg.Database.Dialect == database.MySQL {
		if err := seedExecMySQL(ctx, db, `DELETE FROM navigation_items WHERE id LIKE ?`, "demo-nav-%"); err != nil {
			return fmt.Errorf("remove legacy demo navigation: %w", err)
		}
	} else if err := seedExec(ctx, db, cfg.Database.Dialect, `DELETE FROM navigation_items WHERE id LIKE $1`, `DELETE FROM navigation_items WHERE id LIKE ?`, "demo-nav-%"); err != nil {
		return fmt.Errorf("remove legacy demo navigation: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "demo-seeded pages=%d news=%d media=%d locale=%s navigation=0\n", 3, len(newsList), len(mediaFixtures), demoLocale)
	return err
}

func cleanDemoCommand(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("clean", flag.ContinueOnError)
	flags.SetOutput(stderr)
	demo := flags.Bool("demo", false, "Remove only HydroGo demo records")
	forceProd := flags.Bool("force-prod", false, "Allow demo cleanup in production")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if !*demo {
		return errors.New("clean requires --demo; no other data is removed")
	}
	if isProductionEnvironment() && !*forceProd {
		return errors.New("demo cleanup is disabled in production; pass --force-prod only for an intentional cleanup")
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return err
	}
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		return err
	}
	if err := cleanDemoData(ctx, db, cfg.Database.Dialect, cfg.Media.Dir); err != nil {
		return fmt.Errorf("clean demo data: %w", err)
	}
	_, err = fmt.Fprintln(stdout, "demo-data-cleaned")
	return err
}

func isProductionEnvironment() bool {
	for _, key := range []string{"HYDRO_ENV", "APP_ENV", "GO_ENV"} {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "production") || strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "prod") {
			return true
		}
	}
	return false
}

func cleanDemoData(ctx context.Context, db *sql.DB, dialect database.Dialect, mediaDir string) error {
	if db == nil {
		return errors.New("database is required")
	}
	query := `SELECT storage_key FROM media_assets WHERE id LIKE 'demo-media-%'`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, key := range keys {
		if filepath.Base(key) != key {
			return errors.New("unsafe demo media storage key")
		}
		if err := os.Remove(filepath.Join(mediaDir, key)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, statement := range []string{
		`DELETE FROM page_revisions WHERE page_id LIKE 'demo-%'`,
		`DELETE FROM news_taxonomy_relations WHERE content_id LIKE 'demo-news-%'`,
		`DELETE FROM news_items WHERE id LIKE 'demo-news-%'`,
		`DELETE FROM news_taxonomy_terms WHERE id LIKE 'demo-category-%'`,
		`DELETE FROM pages WHERE id LIKE 'demo-%'`,
		`DELETE FROM members WHERE id='demo-member-portal'`,
		`DELETE FROM media_assets WHERE id LIKE 'demo-media-%'`,
		`DELETE FROM navigation_items WHERE id LIKE 'demo-nav-%'`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

type demoNewsSeed struct {
	id, slug, title, excerpt, takeaways, category, author, coverMediaID, tags, body string
	featured                                                                        bool
}

func localizeDemoNews(item demoNewsSeed, number int) demoNewsSeed {
	item.slug = fmt.Sprintf("demo-story-%03d", number)
	item.title = fmt.Sprintf("HydroGo Demo Story %02d", number)
	item.excerpt = "A sample article created to demonstrate the HydroGo publishing workflow."
	item.takeaways = "- Sample content for demonstrations\n- Safe to remove with hydro seed --clean"
	item.category = "Demo updates"
	item.author = "HydroGo Demo Team"
	item.tags = "demo, hydrogo, example"
	item.body = "This sample article demonstrates publishing, categories, and search in the HydroGo demo workspace. The content is synthetic and does not describe a customer or commercial offer."
	return item
}

func seedDemoTaxonomy(ctx context.Context, db database.Executor, dialect database.Dialect, items []demoNewsSeed) error {
	if len(items) == 0 {
		return nil
	}
	categories := make(map[string]string)
	for _, item := range items {
		name := strings.TrimSpace(item.category)
		if name != "" {
			categories[name] = "demo-category-" + demoTaxonomySlug(name)
		}
	}
	names := make([]string, 0, len(categories))
	for name := range categories {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		id := categories[name]
		labels, err := json.Marshal(map[string]string{"tr": demoCategoryTurkish(name), "en": name})
		if err != nil {
			return err
		}
		if dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db,
				`INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,'category',NULL,?) ON DUPLICATE KEY UPDATE slug=VALUES(slug),labels_json=VALUES(labels_json)`, id, demoTaxonomySlug(name), string(labels)); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, dialect,
			`INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES ($1,$2,'category',NULL,$3) ON CONFLICT(id) DO UPDATE SET slug=EXCLUDED.slug,labels_json=EXCLUDED.labels_json`,
			`INSERT INTO news_taxonomy_terms (id,slug,kind,parent_id,labels_json) VALUES (?,?,'category',NULL,?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug,labels_json=excluded.labels_json`, id, demoTaxonomySlug(name), string(labels)); err != nil {
			return err
		}
	}
	for _, item := range items {
		categoryID := categories[strings.TrimSpace(item.category)]
		if categoryID == "" {
			continue
		}
		if dialect == database.MySQL {
			if err := seedExecMySQL(ctx, db,
				`INSERT IGNORE INTO news_taxonomy_relations (content_id,term_id) VALUES (?,?)`, item.id, categoryID); err != nil {
				return err
			}
		} else if err := seedExec(ctx, db, dialect,
			`INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES ($1,$2) ON CONFLICT(content_id,term_id) DO NOTHING`,
			`INSERT INTO news_taxonomy_relations (content_id,term_id) VALUES (?,?) ON CONFLICT(content_id,term_id) DO NOTHING`, item.id, categoryID); err != nil {
			return err
		}
	}
	return nil
}

func demoCategoryTurkish(english string) string {
	if english == "Demo updates" {
		return "Demo Güncellemeleri"
	}
	return english
}

func demoTaxonomySlug(value string) string {
	var slug strings.Builder
	separator := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			slug.WriteRune(char)
			separator = false
		case char == ' ' || char == '-' || char == '_' || char == '/' || char == '&':
			if slug.Len() > 0 && !separator {
				slug.WriteByte('-')
				separator = true
			}
		}
	}
	return strings.Trim(slug.String(), "-")
}

func seedExec(ctx context.Context, db database.Executor, dialect database.Dialect, postgresQuery, sqliteQuery string, args ...any) error {
	query := postgresQuery
	if dialect == database.SQLite {
		query = sqliteQuery
	}
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// seedExecMySQL mirrors seedExec for MySQL, which needs its own query text
// (plain, non-numbered "?" placeholders, "INSERT IGNORE"/"ON DUPLICATE KEY
// UPDATE" instead of SQLite/Postgres upsert syntax) and its own arg list,
// since MySQL placeholders cannot be referenced by number and reused the way
// "$7"/"?7" are reused above for a value written to two columns.
func seedExecMySQL(ctx context.Context, db database.Executor, mysqlQuery string, args ...any) error {
	_, err := db.ExecContext(ctx, mysqlQuery, args...)
	return err
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

	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	defer db.Close()

	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.NewMobileAuthService(db, cfg.Database.Dialect), auth.DefaultArgon2Params)

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

const (
	maxReleaseManifestBytes = 1 << 20
	maxTrustedKeyFileBytes  = 4 << 10
)

func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d-byte limit", limit)
	}
	return data, nil
}

func readVerifiedReleaseManifest(manifestPath, publicKeyPath string) (release.Manifest, error) {
	data, err := readBoundedFile(manifestPath, maxReleaseManifestBytes)
	if err != nil {
		return release.Manifest{}, err
	}
	var signed release.Signed
	if err := json.Unmarshal(data, &signed); err != nil {
		return release.Manifest{}, fmt.Errorf("parse signed manifest: %w", err)
	}
	keyData, err := readBoundedFile(publicKeyPath, maxTrustedKeyFileBytes)
	if err != nil {
		return release.Manifest{}, err
	}
	publicKey, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(keyData)))
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return release.Manifest{}, errors.New("invalid trusted Ed25519 public key")
	}
	if signed.PublicKey != base64.RawStdEncoding.EncodeToString(publicKey) {
		return release.Manifest{}, errors.New("manifest public key does not match trusted public key")
	}
	return release.Verify(signed)
}

func releaseCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("release command requires a subcommand (verify)")
	}
	switch args[0] {
	case "verify":
		flags := flag.NewFlagSet("release verify", flag.ContinueOnError)
		flags.SetOutput(stderr)
		manifestPath := flags.String("manifest", "", "Path to signed release manifest file")
		publicKeyPath := flags.String("public-key", "", "Path to trusted Ed25519 public key (base64)")
		artifactRoot := flags.String("root", "", "Optional extracted artifact root to verify before execution")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return fmt.Errorf("unexpected arguments: %v", flags.Args())
		}
		if *manifestPath == "" || *publicKeyPath == "" {
			return errors.New("--manifest and --public-key are required")
		}
		data, err := readBoundedFile(*manifestPath, maxReleaseManifestBytes)
		if err != nil {
			return fmt.Errorf("read manifest: %w", err)
		}
		var signed release.Signed
		if err := json.Unmarshal(data, &signed); err != nil {
			return fmt.Errorf("parse signed manifest: %w", err)
		}
		keyData, err := readBoundedFile(*publicKeyPath, maxTrustedKeyFileBytes)
		if err != nil {
			return fmt.Errorf("read public key: %w", err)
		}
		publicKey, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(keyData)))
		if err != nil || len(publicKey) != ed25519.PublicKeySize {
			return errors.New("invalid trusted Ed25519 public key")
		}
		if signed.PublicKey != base64.RawStdEncoding.EncodeToString(publicKey) {
			return errors.New("manifest public key does not match trusted public key")
		}
		verified, err := release.Verify(signed)
		if err != nil {
			return fmt.Errorf("verify manifest: %w", err)
		}
		verifiedArtifacts := 0
		if strings.TrimSpace(*artifactRoot) != "" {
			if err := release.VerifyArtifacts(*artifactRoot, verified); err != nil {
				return fmt.Errorf("verify release artifacts: %w", err)
			}
			verifiedArtifacts = len(verified.Artifacts)
		}
		_, err = fmt.Fprintf(stdout, "verified edition=%s core=%s sdk=%s artifacts=%d artifacts_verified=%d\n", verified.Edition, verified.CoreVersion, verified.SDKVersion, len(verified.Artifacts), verifiedArtifacts)
		return err
	default:
		return fmt.Errorf("unknown release subcommand %q", args[0])
	}
}

// runtimeCommand exposes installer-facing runtime checks while keeping the
// release namespace as the backwards-compatible implementation entrypoint.
func runtimeCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("runtime requires a subcommand (verify or cache)")
	}
	if args[0] == "cache" {
		return runtimeCacheCommand(args[1:], stdout, stderr)
	}
	if args[0] != "verify" {
		return errors.New("runtime requires a subcommand (verify or cache)")
	}
	return releaseCommand(args, stdout, stderr)
}

func runtimeCacheCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || (args[0] != "list" && args[0] != "prune") {
		return errors.New("runtime cache requires list or prune")
	}
	flags := flag.NewFlagSet("runtime cache "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "", "Runtime cache root")
	keep := flags.String("keep", "", "Comma-separated runtime versions to retain")
	dryRun := flags.Bool("dry-run", false, "Plan removals without changing the filesystem")
	confirm := flags.Bool("confirm", false, "Confirm cache deletion")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if strings.TrimSpace(*root) == "" {
		return errors.New("--root is required")
	}
	entries, err := sdkdist.ListCache(*root)
	if err != nil {
		return fmt.Errorf("list runtime cache: %w", err)
	}
	if args[0] == "list" {
		for _, entry := range entries {
			if _, err := fmt.Fprintf(stdout, "runtime-cache-entry version=%s bytes=%d path=%s\n", entry.Version, entry.Bytes, entry.Path); err != nil {
				return err
			}
		}
		return nil
	}
	if *dryRun == *confirm {
		return errors.New("runtime cache prune requires exactly one of --dry-run or --confirm")
	}
	removed, err := sdkdist.PruneCache(*root, parseRuntimeVersions(*keep), !*confirm)
	if err != nil {
		return fmt.Errorf("prune runtime cache: %w", err)
	}
	if *dryRun {
		_, err = fmt.Fprintf(stdout, "runtime-cache-prune-plan root=%s entries=%d\n", *root, len(removed))
	} else {
		_, err = fmt.Fprintf(stdout, "runtime-cache-pruned root=%s entries=%d\n", *root, len(removed))
	}
	return err
}

func parseRuntimeVersions(value string) map[string]bool {
	result := make(map[string]bool)
	for _, version := range strings.Split(value, ",") {
		if version = strings.TrimSpace(version); version != "" {
			result[version] = true
		}
	}
	return result
}

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

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  hydro version
  hydro version --json
  hydro icons --source LOGO --output DIRECTORY
  hydro migrate [runtime flags]
	  hydro run [--address ADDR] [--app-address ADDR] [--database-dsn DSN] [--redis-url URL] [--dependencies auto|none|required] [--compose-file PATH] [--poll DUR]
	  hydro dev [same options]  (backward-compatible alias for hydro run)
  hydro upgrade check [runtime flags]
  hydro create-admin --email EMAIL --password-stdin [runtime flags]
  hydro seed --count=20 --locale=tr|en --clean --confirm-demo [--force-prod] [runtime flags]
  hydro clean --demo [--target demo|staging|production] [--force-prod] [runtime flags]
  hydro doctor [--json] [--fix] [runtime flags]
  hydro validate [--json] [runtime flags]
  hydro validate generator --spec SPEC --root OUTPUT
  hydro validate api --baseline OPENAPI.json --current OPENAPI.json
  hydro frontend check --manifest PATH [--json]
  hydro test [--package ./...] [--timeout 5m] [--race] [--count N]
  hydro backup sqlite --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup postgres --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup prune --root PATH --keep N --dry-run|--confirm [--pinned NAME,...]
  hydro restore verify --snapshot PATH
  hydro restore unpack --archive FILE.tar.gz --destination NEW_DIRECTORY --public-key HEX [--max-bytes N]
  hydro restore stage --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
  hydro restore postgres --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked [runtime flags]
  hydro restore activate-media --staging PATH --active PATH --rollback PATH
  hydro portable export --path PATH [runtime flags]
  hydro portable import --path PATH --dry-run [--max-bytes BYTES] [runtime flags]
  hydro portable import --path PATH --confirm [--max-bytes BYTES] [runtime flags]
  hydro release verify --manifest PATH --public-key PATH [--root PATH]
  hydro runtime verify --manifest PATH --public-key PATH [--root PATH]
  hydro runtime cache list --root PATH
  hydro runtime cache prune --root PATH --keep VERSION,... --dry-run|--confirm
  hydro-issuer issue --customer NAME --project-id ID --private-key PATH [--edition ED] [--features F1,F2] [--expires-in DUR] [--grace-days N] [--out PATH]
  hydro license verify --license PATH [--public-key PATH] [--project-id ID] [--edition ED] [--feature F1]
  hydro license status --license PATH [--public-key PATH] [--project-id ID] [--json]
  hydro-issuer keygen [--public-out PATH] [--private-out PATH]
  hydro new PROJECT [--preset=news] [--database=sqlite] [--profile=compact] [--git]
  hydro new resource [NAME] [--fields name:type[:required][:unique],...] [--out DIR]
  hydro project new PROJECT [--preset=news] [--database=sqlite] [--profile=compact] [--git]
  hydro project check [--config config.yaml] [--lock hydrogo.lock] [--json]
  hydro check [--config config.yaml] [--lock hydrogo.lock] [--json]
  hydro tenant validate [runtime flags]
  hydro tenant provision --tenant-id ID --admin-email EMAIL --admin-password-stdin [runtime flags]
  hydro tenant backfill (--tenant-id ID|--active) [--dry-run|--confirm] [runtime flags]
  hydro tenant list [runtime flags]
  hydro events catalog
  hydro fonts add FAMILY [--out public/fonts]
  hydro scan-links --url https://example.test [--max-pages 500] [--concurrency 4] [--delay 100ms]
  hydro module create --spec SPEC --out OUTPUT
  hydro module new --spec SPEC --out OUTPUT
  hydro resource create --spec SPEC --out OUTPUT
  hydro resource new --spec SPEC --out OUTPUT
  hydro resource check --root OUTPUT [--spec SPEC]
  hydro collections sync --confirm-admin-crud [--config config.yaml] [--generated generated] [--tenant ID]
  hydro api diff --baseline OPENAPI.json --current OPENAPI.json
  hydro schema diff --baseline RELEASE.json --current RELEASE.json --public-key KEY
  hydro permissions check --spec SPEC [--json]
  hydro resource create --name NAME --field name:type[:nullable] [--field ...] [--out OUTPUT]
  hydro migration create --id YYYYMMDDHHMM_name --module MODULE --out migrations
  hydro generate module|resource|crud|page|form|api|admin --spec SPEC --out OUTPUT
  hydro jobs dead-letter list [--limit N] [--json] [runtime flags]
  hydro jobs dead-letter requeue --id ID --confirm [runtime flags]
  hydro jobs dead-letter purge --older-than 720h --limit N --confirm [runtime flags]
  hydro serve [--cgi | --fastcgi --fastcgi-address 127.0.0.1:9001] [--dev] [runtime flags]
  hydro help

Examples:
  hydro validate --config config.yaml
  hydro test --package ./modules/news/... --timeout 2m
  hydro collections sync --config config.yaml --generated generated --confirm-admin-crud
  hydro module create --spec ./module.yaml --out ./modules/catalog
  hydro resource create --name products --field title:text --field price:integer
	  hydro run --address 0.0.0.0:8090

Failure remediation:
  config/database errors: run "hydro validate --config PATH", then check the
    dialect, DSN, and file permissions before retrying.
  migration errors: run "hydro migrate" against a disposable database first;
    never edit an applied migration—create a new migration instead.
  test timeout or race failures: narrow "--package", increase "--timeout",
    and reproduce with "go test -run TestName -count=1 ./path".
  dev port already in use: pass another "--address"/"--app-address", or stop
    the process that owns the port.

Runtime flags:
  --config PATH
  --capacity-profile compact|standard|distributed
  --address HOST:PORT
  --database-dialect sqlite|postgres
  --database-dsn DSN

`)
}

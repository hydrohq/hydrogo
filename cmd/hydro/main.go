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
	"net/http"
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
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	analyticsstore "github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	formstore "github.com/ardaozyildirim/hydrogo/internal/forms/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/generator"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/licensing"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
	"github.com/ardaozyildirim/hydrogo/internal/portable"
	"github.com/ardaozyildirim/hydrogo/internal/projectgen"
	"github.com/ardaozyildirim/hydrogo/internal/release"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/sdkdist"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/migrations"
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
	case "migration":
		return migrationCommand(args[1:], stdout, stderr)
	case "upgrade":
		return upgrade(args[1:], stdout, stderr)
	case "create-admin":
		return createAdmin(args[1:], stdin, stdout, stderr)
	case "seed-demo":
		return seedDemo(args[1:], stdout, stderr)
	case "generate":
		return generateCommand(args[1:], stdout, stderr)
	case "module":
		return moduleCommand(args[1:], stdout, stderr)
	case "resource":
		return resourceCommand(args[1:], stdout, stderr)
	case "api":
		return apiCommand(args[1:], stdout, stderr)
	case "frontend":
		return frontendCommand(args[1:], stdout, stderr)
	case "schema":
		return schemaCommand(args[1:], stdout, stderr)
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
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func devCommand(args []string, stdout, stderr io.Writer) error {
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
		_, err = fmt.Fprintf(stdout, "resource-compatible collection=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked)
		return err
	}
	reports, err := generator.CheckGeneratedCompatibility(*root)
	if err != nil {
		return err
	}
	for _, report := range reports {
		if _, err := fmt.Fprintf(stdout, "resource-compatible collection=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked); err != nil {
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
	}
	return filter
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
		db, err := database.OpenWithOptions(context.Background(), cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
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
		db, err := database.OpenWithOptions(context.Background(), cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
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
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
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
		return errors.New("restore command requires a subcommand (verify, stage, postgres or activate-media)")
	}
	switch args[0] {
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

func serve(args []string, stderr io.Writer) error {
	if stderr == nil {
		stderr = io.Discard
	}
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
	if _, err := validateRuntimeLicense(cfg, stderr); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
	if err != nil {
		return err
	}
	defer db.Close()
	transactioner := database.NewTenantTransactor(db, cfg.Database.Dialect, requireTenantScope)
	analyticsStore := analyticsstore.Store{Executor: db, Dialect: cfg.Database.Dialect}
	contentViewQueue := analytics.NewContentViewQueue(analyticsStore, 256, 1)
	defer contentViewQueue.Close()

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
	authHTTP := auth.HTTPHandler{Service: authSvc, Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope), Secure: cfg.Security.SecureCookies, MFA: mfaService, MFADB: db, Transactions: transactioner, LoginLimiter: newRateLimiter(5, time.Minute), MFALimiter: newRateLimiter(5, time.Minute), RecoveryLimiter: newRateLimiter(3, time.Hour), RecoveryConfirmLimiter: newRateLimiter(10, time.Hour), InviteAcceptLimiter: newRateLimiter(10, time.Hour), Recovery: recoveryService, RecoveryDB: db, Invitation: invitationService, InviteDB: db}
	mobileAuthHTTP := auth.MobileHTTPHandler{Auth: authSvc, Mobile: mobileAuth, MFA: mfaService, MFADB: db, LoginLimiter: newRateLimiter(5, time.Minute), MFALimiter: newRateLimiter(5, time.Minute), Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope)}
	memberStore := memberstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	memberSessionStore := memberstore.SessionStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	memberAuditStore := memberstore.AuditStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	modules := enabledModuleSet(cfg.Modules)
	var memberService members.Service
	var memberHTTP membertransport.Handler
	if modules["members"] {
		memberService = members.Service{Members: memberStore, Sessions: memberSessionStore, Audit: memberAuditStore, ArgonParams: auth.DefaultArgon2Params}
	}
	if modules["members"] {
		memberHTTP = membertransport.Handler{Service: memberService, Mobile: &members.MobileAuthService{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, Secure: cfg.Security.SecureCookies, Limiter: newRateLimiter(10, time.Minute), Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope), DB: db, DefaultLocale: cfg.I18N.DefaultLocale}
	}
	if cfg.Mail.Host != "" {
		sender, err := mailer.NewSMTP(mailer.SMTPConfig{Host: cfg.Mail.Host, Port: cfg.Mail.Port, From: cfg.Mail.From, Username: cfg.Mail.Username, Password: cfg.Mail.Password, StartTLS: cfg.Mail.StartTLS})
		if err != nil {
			return err
		}
		notifier := mailer.NotificationSender{Sender: sender}
		authHTTP.Notifier = notifier
		authHTTP.InviteNotice = notifier
	}
	publicPageCache, err := newPageCache(modules["pages"], budget, redisClient, cfg.Redis.KeyPrefix)
	if err != nil {
		return fmt.Errorf("configure redis page cache: %w", err)
	}
	contentService := pageservice.ContentService{
		Transactions: transactioner,
		Pages:        pagesstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		Audit:        auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		Outbox:       outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		PublicCache:  publicPageCache,
		Sanitize:     ui.NewHTMLSanitizer().Sanitize,
	}
	var sqliteCacheConn *sql.Conn
	if cfg.Database.SQLiteExternalWriteWatch && cfg.Database.Dialect == database.SQLite && publicPageCache != nil {
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
		monitor.OnChange = publicPageCache.Clear
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
	if modules["media"] {
		uploader, err := media.NewUploaderWithConcurrency(cfg.Media.Dir, budget.MediaMaxBytes, budget.WorkerConcurrency)
		if err != nil {
			return err
		}
		mediaOptimizer := media.NewOptimizer(cfg.Media.Dir, minInt(budget.WorkerConcurrency, 2), mediaStore, db, os.Getenv("HYDRO_MEDIA_FFMPEG"))
		var blobStore media.BlobStore
		if cfg.Media.StorageDriver == "s3" {
			blobStore, err = media.NewS3BlobStore(media.S3Config{Endpoint: cfg.Media.S3Endpoint, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, UseSSL: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle, PublicBaseURL: cfg.Media.CDNBaseURL})
			if err != nil {
				return fmt.Errorf("configure media object storage: %w", err)
			}
		}
		mediaService = media.Service{Transactions: transactioner, Uploader: uploader, BlobStore: blobStore, Assets: mediaStore, Optimizer: mediaOptimizer}
		go func() { <-ctx.Done(); mediaOptimizer.Close() }()
	}
	mediaHTTP := media.HTTPHandler{Service: mediaService, DB: db}
	pageStore := pagesstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	siteStore := sitestore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	var siteService site.Service
	var siteHTTP sitetransport.Handler
	if modules["site"] {
		siteService = site.Service{Transactions: transactioner, Store: siteStore, Audit: auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}}
		if modules["forms"] {
			siteService.Forms = formstore.Store{Executor: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		}
		sitemapOrigin := cfg.HTTP.PublicOrigin
		if sitemapOrigin == "" && cfg.HTTP.Address != "" {
			sitemapOrigin = "http://" + cfg.HTTP.Address
		}
		siteHTTP = sitetransport.Handler{DB: db, Store: siteStore, Service: &siteService, SubmissionLimiter: newRateLimiter(5, time.Minute), AllowedForms: map[string]struct{}{"contact": {}}, SitemapOrigin: sitemapOrigin, SitemapProvider: func(providerCtx context.Context) ([]site.SitemapEntry, error) {
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
	}
	var newsService newsservice.Service
	var newsWorker newsservice.Worker
	if moduleEnabled(cfg.Modules, "news") {
		newsStore := newsstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		newsService = newsservice.Service{Transactions: transactioner, Repository: newsStore, Audit: auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, WorkflowRole: func(ctx context.Context) string {
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
		{ID: "news", Label: "Haberler", Description: "Taslak, zamanlama ve yayın akışı.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "cover_media_id", Label: "Kapak görseli (herkese açık asset)", Kind: adminstudio.FieldText, Widget: "media"}, {ID: "excerpt", Label: "Kısa özet", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "key_takeaways", Label: "Önemli Noktalar (Key Takeaways)", Kind: adminstudio.FieldText, Widget: "list"}, {ID: "category", Label: "Kategori", Kind: adminstudio.FieldText}, {ID: "author_name", Label: "Yazar adı", Kind: adminstudio.FieldText}, {ID: "featured", Label: "Öne çıkan", Kind: adminstudio.FieldText, Widget: "checkbox"}, {ID: "tags", Label: "Etiketler", Kind: adminstudio.FieldText}, {ID: "seo_title", Label: "SEO başlığı", Kind: adminstudio.FieldText}, {ID: "seo_description", Label: "SEO açıklaması", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}}, Actions: []adminstudio.Action{{Label: "Tüm haberler", Href: "/admin/news"}, {Label: "Release notes", Href: "/admin/news/demo-news"}, {Label: "Başlangıç rehberi", Href: "/admin/news/demo-news-guide"}}},
	})
	if err != nil {
		return fmt.Errorf("admin studio registry: %w", err)
	}
	activeStudioCollections := func() []adminstudio.Collection {
		return adminstudio.FilterActiveCollections(studioRegistry.Collections(), enabledModuleSet(cfg.Modules))
	}
	var pageHTTP pagetransport.Handler
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
		pageHTTP = pagetransport.Handler{DB: db, Content: contentService, Theme: selectedTheme, Media: mediaService, AdminCollection: pageCollection, DefaultLocale: cfg.I18N.DefaultLocale, AllowDemoFrontend: demoFrontendEnabled()}
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
		newsHTTP = newstransport.Handler{Service: newsService, DB: db, Media: mediaService, AdminCollection: newsCollection, DefaultLocale: cfg.I18N.DefaultLocale}
	}
	routes := append([]router.Route{}, authHTTP.RouterRoutes()...)
	routes = append(routes, mobileAuthHTTP.RouterRoutes()...)
	if modules["pages"] {
		pageRoutes := pageRouterRoutes(pageHTTP.Routes())
		for i := range pageRoutes {
			if strings.HasPrefix(pageRoutes[i].Pattern, "GET /public/pages/") {
				pageRoutes[i].Handler = analytics.ContentViewMiddleware(pageRoutes[i].Handler, contentViewQueue, "page")
			}
		}
		routes = append(routes, pageRoutes...)
	}
	if enabledModuleSet(cfg.Modules)["members"] {
		routes = append(routes, memberHTTP.Routes()...)
	}
	routes = append(routes, router.Route{Pattern: "GET /", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
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
			_, _ = io.WriteString(w, technicalHomeHTML(cfg.I18N.DefaultLocale, publicHomePath(cfg.I18N.DefaultLocale), demoFrontendEnabled(), security.NonceFromContext(r.Context())))
			return
		}
		http.NotFound(w, r)
	})})
	routes = append(routes, router.Route{Pattern: "GET /docs", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, docsComingSoonHTML(security.NonceFromContext(r.Context())))
	})})
	routes = append(routes, router.Route{Pattern: "POST /analytics/consent", Access: router.Public, Handler: analytics.ConsentHandler(analyticsStore, cfg.Security.SecureCookies)})
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
		extensionRegistry.ServeExtensionSurface(w, r, r.PathValue("extension_id"), r.PathValue("surface_id"), cfg.I18N.DefaultLocale, principal)
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
	routes = append(routes, router.Route{Pattern: "GET /login", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		returnTo := security.SafeReturnPath(r.URL.Query().Get("return_to"), "/admin")
		if user := auth.UserFromContext(r.Context()); user != nil && user.Active && user.Role == auth.RoleAdmin {
			http.Redirect(w, r, returnTo, http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Safe fallback: "Giriş şu anda tamamlanamadı. Lütfen tekrar deneyin." is rendered via ui.RenderAdminLogin
		if err := ui.RenderAdminLogin(w, security.CSRFTokenFromContext(r.Context()), security.NonceFromContext(r.Context()), returnTo); err != nil {
			http.Error(w, "admin login unavailable", http.StatusInternalServerError)
		}
	})})
	var runtimeMetrics *observability.Metrics
	adminDashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
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
			contentFilter := contentViewFilterFromRequest(r, 100)
			contentViews, _ := analyticsStore.ListContentViews(r.Context(), contentFilter)
			contentViewRows := make([]ui.ContentViewReportRow, 0, len(contentViews))
			for _, row := range contentViews {
				contentViewRows = append(contentViewRows, ui.ContentViewReportRow{Day: row.Day, TenantID: row.TenantID, ContentType: row.ContentType, ContentID: row.ContentID, Locale: row.Locale, ViewCount: row.ViewCount})
			}
			if err := ui.RenderAdminReports(w, ui.AdminDashboardData{
				Email:        user.Email,
				Collections:  activeStudioCollections(),
				Nonce:        security.NonceFromContext(r.Context()),
				Active:       "reports",
				Metrics:      runtimeMetrics,
				MemberStats:  stats,
				DataTime:     dataTime,
				DataSource:   "Proses İçi Bellek Telemetrisi",
				DateFilter:   dateRange,
				Locale:       adminLocaleFromRequest(r),
				ContentViews: contentViewRows,
			}); err != nil {
				http.Error(w, "admin reports unavailable", http.StatusInternalServerError)
			}
			return
		}
		if err := ui.RenderAdminDashboard(w, ui.AdminDashboardData{Email: user.Email, Collections: activeStudioCollections(), Nonce: security.NonceFromContext(r.Context()), Active: "dashboard", Locale: adminLocaleFromRequest(r)}); err != nil {
			http.Error(w, "admin dashboard unavailable", http.StatusInternalServerError)
		}
	})
	routes = append(routes, router.Route{Pattern: "GET /admin", Access: router.Protected, Handler: adminDashboardHandler})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports", Access: router.Protected, Handler: adminDashboardHandler})
	routes = append(routes, router.Route{Pattern: "POST /admin/account/locale", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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
			uniqueVisitors := 0
			var startedAt time.Time
			if runtimeMetrics != nil {
				totalReq = runtimeMetrics.TotalRequests()
				totalViews = runtimeMetrics.TotalViews()
				totalErrors = runtimeMetrics.TotalErrors()
				avgLatency = runtimeMetrics.AvgLatencyMS()
				uniqueVisitors = runtimeMetrics.UniqueVisitorsCount()
				startedAt = runtimeMetrics.StartedAt()
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
					rows, _ := analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 500))
					return rows
				}(),
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
		if rows, err := analyticsStore.ListContentViews(r.Context(), contentViewFilterFromRequest(r, 500)); err == nil && len(rows) > 0 {
			_ = writer.Write([]string{})
			_ = writer.Write([]string{"ContentViews", "Day", "TenantID", "ContentType", "ContentID", "Locale", "ViewCount"})
			for _, row := range rows {
				_ = writer.Write([]string{"", row.Day, row.TenantID, row.ContentType, row.ContentID, row.Locale, strconv.FormatInt(row.ViewCount, 10)})
			}
		}
		writer.Flush()
	})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports/export", Access: router.Protected, Handler: adminReportsExportHandler})

	adminControlHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
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
			if err := ui.RenderAdminArea(w, ui.AdminAreaData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), CSRF: security.CSRFTokenFromContext(r.Context()), Area: "developer", Locale: adminLocaleFromRequest(r)}); err != nil {
				http.Error(w, "developer tools unavailable", http.StatusInternalServerError)
			}
			return
		}
		if r.URL.Path == "/admin/developer/resources" {
			if err := ui.RenderAdminResourceUsage(w, ui.AdminResourceUsageData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), Snapshot: runtimeMetrics.Snapshot(), Locale: adminLocaleFromRequest(r)}); err != nil {
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
			err = ui.RenderAdminControl(w, ui.AdminControlData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), CSRF: security.CSRFTokenFromContext(r.Context()), Active: active, Locale: adminLocaleFromRequest(r)})
		} else {
			areaData := ui.AdminAreaData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), CSRF: security.CSRFTokenFromContext(r.Context()), Area: active, Locale: adminLocaleFromRequest(r), LocaleSaved: r.URL.Query().Get("locale_saved") == "1"}
			if active == "users" {
				if repo, ok := any(userStore).(auth.UserAdminRepository); ok {
					areaData.Users, err = repo.List(r.Context())
				} else {
					err = fmt.Errorf("admin user repository unavailable")
				}
			}
			if err == nil {
				err = ui.RenderAdminArea(w, areaData)
			}
		}
		if err != nil {
			http.Error(w, "admin control center unavailable", http.StatusInternalServerError)
		}
	})
	for _, pattern := range adminControlRoutePatterns(modules) {
		routes = append(routes, router.Route{Pattern: pattern, Access: router.Protected, Handler: adminControlHandler})
	}
	if modules["media"] {
		routes = append(routes, mediaRouterRoutes(mediaHTTP.AdminRoutes())...)
		routes = append(routes, mediaRouterRoutes(mediaHTTP.Routes())...)
	}
	if modules["site"] {
		routes = append(routes, siteRouterRoutes(siteHTTP.Routes())...)
	}
	if modules["news"] {
		newsRoutes := newsRouterRoutes(newsHTTP.Routes())
		for i := range newsRoutes {
			if strings.HasPrefix(newsRoutes[i].Pattern, "GET /public/news/") {
				newsRoutes[i].Handler = analytics.ContentViewMiddleware(newsRoutes[i].Handler, contentViewQueue, "news")
			}
		}
		routes = append(routes, newsRoutes...)
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
	go runSessionCleanupObserved(ctx, sessionStore, stderr, metrics)
	if modules["news"] {
		go runNewsWorker(ctx, db, newsWorker, metrics, stderr, budget.WorkerConcurrency)
	}
	webhookEndpoints := make([]webhooks.Endpoint, 0, len(cfg.Webhooks))
	for _, configured := range cfg.Webhooks {
		topics := make(map[string]struct{}, len(configured.Topics))
		for _, topic := range configured.Topics {
			topics[strings.TrimSpace(topic)] = struct{}{}
		}
		webhookEndpoints = append(webhookEndpoints, webhooks.Endpoint{ID: configured.ID, URL: configured.URL, Secret: configured.Secret, Topics: topics})
	}
	if modules["pages"] || modules["news"] || modules["commerce"] {
		go runOutboxWorker(ctx, webhooks.Worker{DB: db, Store: outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, TenantScopes: tenantScopes, Dispatcher: webhooks.Dispatcher{Endpoints: webhookEndpoints}, WorkerID: "outbox-worker-0", Lease: time.Minute, RetryAfter: time.Minute}, stderr)
	}
	logWriter := stderr
	if logWriter == nil {
		logWriter = io.Discard
	}
	logger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return server.Serve(ctx, server.Config{
		Address:          cfg.HTTP.Address,
		ShutdownTimeout:  budget.ShutdownTimeout,
		AuthSvc:          authSvc,
		Metrics:          metrics,
		Logger:           logger,
		EnableHSTS:       cfg.Security.EnableHSTS,
		SecureCookie:     cfg.Security.SecureCookies,
		HTTPBodyLimit:    budget.HTTPBodyBytes,
		PublicOrigin:     cfg.HTTP.PublicOrigin,
		PublicAPIOrigins: cfg.Security.PublicAPIOrigins,
		TenantResolver:   tenantResolver,
		Routes:           routes,
		Ready: func(checkCtx context.Context) error {
			if err := db.PingContext(checkCtx); err != nil {
				return err
			}
			return database.CheckCompatibilityForModules(checkCtx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
		},
	})
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
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		return fmt.Errorf("schema compatibility: %w", err)
	}
	if err := os.MkdirAll(cfg.Media.Dir, 0o750); err != nil {
		return fmt.Errorf("media directory: %w", err)
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
			Version           string `json:"version"`
			Commit            string `json:"commit"`
			BuildTime         string `json:"build_time"`
			SchemaMigration   string `json:"schema_migration"`
			Profile           string `json:"profile"`
			Database          string `json:"database"`
			BackupTools       string `json:"backup_tools"`
			CacheEntries      int    `json:"cache_entries"`
			QueueEntries      int    `json:"queue_entries"`
			WorkerConcurrency int    `json:"worker_concurrency"`
			DeadLetteredJobs  int64  `json:"dead_lettered_jobs"`
			MediaMaxBytes     int64  `json:"media_max_bytes"`
			MediaWritable     bool   `json:"media_writable"`
			SecureCookies     bool   `json:"secure_cookies"`
			HSTS              bool   `json:"hsts"`
		}{Version: version.Version, Commit: version.Commit, BuildTime: version.BuildTime, SchemaMigration: version.SchemaMigration, Profile: string(cfg.Profile), Database: string(cfg.Database.Dialect), BackupTools: backupTools, CacheEntries: budget.CacheEntries, QueueEntries: budget.QueueEntries, WorkerConcurrency: budget.WorkerConcurrency, DeadLetteredJobs: deadLetteredJobs, MediaMaxBytes: budget.MediaMaxBytes, MediaWritable: true, SecureCookies: cfg.Security.SecureCookies, HSTS: cfg.Security.EnableHSTS})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, string(data))
		return err
	}
	_, err = fmt.Fprintf(stdout, "profile=%s database=%s backup_tools=%s cache_entries=%d queue_entries=%d worker_concurrency=%d dead_lettered_jobs=%d media_max_bytes=%d\n", cfg.Profile, cfg.Database.Dialect, backupTools, budget.CacheEntries, budget.QueueEntries, budget.WorkerConcurrency, deadLetteredJobs, budget.MediaMaxBytes)
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
	_, err = fmt.Fprintf(stdout, "compatible collection=%s generator=%s spec_sha256=%s files=%d\n", report.Collection, report.GeneratorVersion, report.SpecSHA256, report.FilesChecked)
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
	db, err := database.OpenWithOptions(context.Background(), cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
  <p class="hero-subtitle">Manage your content, memberships, and commerce operations from one fast, secure workspace.</p>

  <div class="actions-group">
    <a href="/admin" class="btn-hero btn-start">Get Started</a>
    <a href="/docs" class="btn-hero btn-docs">Documentation</a>
  </div>
</main>
</body>
</html>`, template.HTMLEscapeString(locale), styleNonceAttr)
}

func docsComingSoonHTML(nonce ...string) string {
	styleNonceAttr := ""
	if len(nonce) > 0 && strings.TrimSpace(nonce[0]) != "" {
		styleNonceAttr = fmt.Sprintf(` nonce="%s"`, template.HTMLEscapeString(nonce[0]))
	}
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Documentation · Coming Soon</title>
  <style%s>
    :root {
      --bg: #090a0f;
      --card: #0e1118;
      --border: #1e2533;
      --text: #ffffff;
      --text-muted: #94a3b8;
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
    }
    .docs-wrapper {
      max-width: 520px;
      width: 100%%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
    }
    .badge {
      font-size: 0.75rem;
      font-weight: 700;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: #a855f7;
      background: rgba(168, 85, 247, 0.1);
      border: 1px solid rgba(168, 85, 247, 0.25);
      padding: 6px 14px;
      border-radius: 9999px;
      margin-bottom: 20px;
    }
    h1 {
      font-size: clamp(1.85rem, 3.5vw, 2.3rem);
      font-weight: 600;
      letter-spacing: -0.015em;
      color: #ffffff;
      margin-bottom: 12px;
    }
    p {
      color: var(--text-muted);
      font-size: 1rem;
      margin-bottom: 30px;
      line-height: 1.6;
    }
    .btn-back {
      display: inline-flex;
      align-items: center;
      padding: 10px 22px;
      font-size: 0.9rem;
      font-weight: 600;
      color: #ffffff;
      background: #1e2533;
      border: 1px solid #334155;
      border-radius: 8px;
      text-decoration: none;
      transition: all 0.15s ease;
    }
    .btn-back:hover {
      background: #334155;
      color: #ffffff;
    }
  </style>
</head>
<body>
<main class="docs-wrapper">
  <span class="badge">Documentation</span>
  <h1>Coming Soon</h1>
  <p>HydroGo developer documentation and guides are currently being crafted.</p>
  <a href="/" class="btn-back">&larr; Back to Home</a>
</main>
</body>
</html>`, styleNonceAttr)
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

func newAuditRecorder(db *sql.DB, dialect database.Dialect, requireTenantScope bool) auth.AuditRecorder {
	store := auditstore.Store{Dialect: dialect, RequireTenantScope: requireTenantScope}
	return auth.AuditRecorderFunc(func(ctx context.Context, action, targetID string) error {
		return database.NewTenantTransactor(db, dialect, requireTenantScope).Within(ctx, func(tx database.Executor) error {
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
	if err != nil {
		return err
	}
	defer db.Close()
	return database.ApplyMigrationsForModules(ctx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
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
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
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
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*confirmDemo {
		return errors.New("seed-demo requires --confirm-demo; use only on a disposable local database")
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
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
	if err != nil {
		return err
	}
	defer db.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Demo member data is deliberately isolated from admin users. It gives the
	// public member portal a real, repeatable account flow without exposing any
	// administrator capability.
	demoMemberHash, err := auth.HashPassword("memberpass123", auth.DefaultArgon2Params)
	if err != nil {
		return fmt.Errorf("seed demo member password: %w", err)
	}
	if err := seedExec(ctx, db, cfg.Database.Dialect,
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
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES ($1,$2,$3,'image/png',$4,$5,$6,'public','ready',$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES (?1,?2,?3,'image/png',?4,?5,?6,'public','ready',?7,?7)`, id, storageKey, filename, sizeBytes, width, height, now); err != nil {
			return fmt.Errorf("seed media asset %s: %w", id, err)
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE media_assets SET storage_key=$1,original_filename=$2,size_bytes=$3,width=$4,height=$5,visibility='public',status='ready',updated_at=$6 WHERE id=$7`, `UPDATE media_assets SET storage_key=?1,original_filename=?2,size_bytes=?3,width=?4,height=?5,visibility='public',status='ready',updated_at=?6 WHERE id=?7`, storageKey, filename, sizeBytes, width, height, now, id); err != nil {
			return err
		}
		var storedKey string
		query := `SELECT storage_key FROM media_assets WHERE id=$1`
		if cfg.Database.Dialect == database.SQLite {
			query = `SELECT storage_key FROM media_assets WHERE id=?1`
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
		localePath := url.PathEscape(demoLocale)
		blocks = strings.ReplaceAll(strings.ReplaceAll(blocks, "/public/pages/tr/", "/public/pages/"+localePath+"/"), "/public/news/tr/", "/public/news/"+localePath+"/")
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES ($1,$2,1,$3,$4,$5,'published',1,1,1,$6,$6,$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES (?1,?2,1,?3,?4,?5,'published',1,1,1,?6,?6,?7,?7)`, id, title, demoLocale, slug, body, blocks, now); err != nil {
			return err
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE pages SET locale=$1,slug=$2,title=$3,body=$4,blocks=$5,published_locale=$1,published_slug=$2,published_title=$3,published_body=$4,published_blocks=$5 WHERE id=$6`, `UPDATE pages SET locale=?1,slug=?2,title=?3,body=?4,blocks=?5,published_locale=?1,published_slug=?2,published_title=?3,published_body=?4,published_blocks=?5 WHERE id=?6`, demoLocale, slug, title, body, blocks, id); err != nil {
			return err
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

	newsList := []struct {
		id, slug, title, excerpt, takeaways, category, author, coverMediaID, tags, body string
		featured                                                                        bool
	}{
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

	for _, item := range newsList {
		featuredInt := 0
		if item.featured {
			featuredInt = 1
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'published',1,1,1,0,$13,$13) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,cover_media_id,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,'published',1,1,1,0,?13,?13)`, item.id, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredInt, item.tags, item.body, now); err != nil {
			return fmt.Errorf("seed news %s: %w", item.id, err)
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE news_items SET locale=$1,slug=$2,title=$3,excerpt=$4,key_takeaways=$5,category=$6,author_name=$7,cover_media_id=$8,featured=$9,tags=$10,body=$11,status='published',published_revision=revision WHERE id=$12`, `UPDATE news_items SET locale=?1,slug=?2,title=?3,excerpt=?4,key_takeaways=?5,category=?6,author_name=?7,cover_media_id=?8,featured=?9,tags=?10,body=?11,status='published',published_revision=revision WHERE id=?12`, demoLocale, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, item.coverMediaID, featuredInt, item.tags, item.body, item.id); err != nil {
			return fmt.Errorf("publish news %s: %w", item.id, err)
		}
	}
	if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO site_settings (key,value,is_public,updated_at) VALUES ($1,$2,1,$3) ON CONFLICT (key) DO NOTHING`, `INSERT OR IGNORE INTO site_settings (key,value,is_public,updated_at) VALUES (?,?,1,?)`, "site.title", "HydroGo Demo", now); err != nil {
		return fmt.Errorf("seed setting: %w", err)
	}
	if err := seedExec(ctx, db, cfg.Database.Dialect, `DELETE FROM navigation_items WHERE id LIKE $1`, `DELETE FROM navigation_items WHERE id LIKE ?`, "demo-nav-%"); err != nil {
		return fmt.Errorf("remove legacy demo navigation: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "demo-seeded pages=%d news=%d media=%d navigation=0\n", 3, len(newsList), len(mediaFixtures))
	return err
}

func seedExec(ctx context.Context, db database.Executor, dialect database.Dialect, postgresQuery, sqliteQuery string, args ...any) error {
	query := postgresQuery
	if dialect == database.SQLite {
		query = sqliteQuery
	}
	_, err := db.ExecContext(ctx, query, args...)
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
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, database.OpenOptions{MaxOpenConns: budget.DBMaxOpenConns, MaxIdleConns: budget.DBMaxIdleConns})
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
}

func addRuntimeFlags(flags *flag.FlagSet) runtimeFlags {
	return runtimeFlags{
		capacityProfile: flags.String("capacity-profile", "", "compact, standard or distributed"),
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
	return config.Load(*values.configPath, os.LookupEnv, overrides)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  hydro version
  hydro version --json
  hydro migrate [runtime flags]
	  hydro run [--address ADDR] [--app-address ADDR] [--database-dsn DSN] [--redis-url URL] [--dependencies auto|none|required] [--compose-file PATH] [--poll DUR]
	  hydro dev [same options]  (backward-compatible alias for hydro run)
  hydro upgrade check [runtime flags]
  hydro create-admin --email EMAIL --password-stdin [runtime flags]
  hydro seed-demo --confirm-demo [runtime flags]
  hydro doctor [--json] [runtime flags]
  hydro validate [--json] [runtime flags]
  hydro validate generator --spec SPEC --root OUTPUT
  hydro validate api --baseline OPENAPI.json --current OPENAPI.json
  hydro frontend check --manifest PATH [--json]
  hydro test [--package ./...] [--timeout 5m] [--race] [--count N]
  hydro backup sqlite --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup postgres --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup prune --root PATH --keep N --dry-run|--confirm [--pinned NAME,...]
  hydro restore verify --snapshot PATH
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
  hydro project new PROJECT [--preset=news] [--database=sqlite] [--profile=compact] [--git]
  hydro project check [--config config.yaml] [--lock hydrogo.lock] [--json]
  hydro check [--config config.yaml] [--lock hydrogo.lock] [--json]
  hydro tenant validate [runtime flags]
  hydro tenant provision --tenant-id ID --admin-email EMAIL --admin-password-stdin [runtime flags]
  hydro tenant backfill (--tenant-id ID|--active) [--dry-run|--confirm] [runtime flags]
  hydro tenant list [runtime flags]
  hydro events catalog
  hydro module create --spec SPEC --out OUTPUT
  hydro module new --spec SPEC --out OUTPUT
  hydro resource create --spec SPEC --out OUTPUT
  hydro resource new --spec SPEC --out OUTPUT
  hydro resource check --root OUTPUT [--spec SPEC]
  hydro api diff --baseline OPENAPI.json --current OPENAPI.json
  hydro schema diff --baseline RELEASE.json --current RELEASE.json --public-key KEY
  hydro permissions check --spec SPEC [--json]
  hydro resource create --name NAME --field name:type[:nullable] [--field ...] [--out OUTPUT]
  hydro migration create --id YYYYMMDDHHMM_name --module MODULE --out migrations
  hydro generate module|resource|crud|page|form|api|admin --spec SPEC --out OUTPUT
  hydro jobs dead-letter list [--limit N] [--json] [runtime flags]
  hydro jobs dead-letter requeue --id ID --confirm [runtime flags]
  hydro jobs dead-letter purge --older-than 720h --limit N --confirm [runtime flags]
  hydro serve [runtime flags]
  hydro help

Examples:
  hydro validate --config config.yaml
  hydro test --package ./modules/news/... --timeout 2m
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

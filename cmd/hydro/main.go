package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/generator"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/observability"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
	"github.com/ardaozyildirim/hydrogo/internal/portable"
	"github.com/ardaozyildirim/hydrogo/internal/release"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
	"github.com/ardaozyildirim/hydrogo/migrations"
	graphqlapi "github.com/ardaozyildirim/hydrogo/modules/api/graphql"
	commerce "github.com/ardaozyildirim/hydrogo/modules/commerce"
	iyzicoprovider "github.com/ardaozyildirim/hydrogo/modules/commerce/providers/iyzico"
	paytrprovider "github.com/ardaozyildirim/hydrogo/modules/commerce/providers/paytr"
	stripeprovider "github.com/ardaozyildirim/hydrogo/modules/commerce/providers/stripe"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
	commercetransport "github.com/ardaozyildirim/hydrogo/modules/commerce/transport"
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
	case "migrate":
		return migrate(args[1:], stderr)
	case "upgrade":
		return upgrade(args[1:], stdout, stderr)
	case "create-admin":
		return createAdmin(args[1:], stdin, stdout, stderr)
	case "seed-demo":
		return seedDemo(args[1:], stdout, stderr)
	case "generate":
		return generateCommand(args[1:], stdout, stderr)
	case "doctor":
		return doctor(args[1:], stdout, stderr)
	case "backup":
		return backupCommand(args[1:], stdout, stderr)
	case "restore":
		return restoreCommand(args[1:], stdout, stderr)
	case "portable":
		return portableCommand(args[1:], stdout, stderr)
	case "release":
		return releaseCommand(args[1:], stdout, stderr)
	case "jobs":
		return jobsCommand(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func generateCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "module" {
		return errors.New("generate requires the module subcommand")
	}
	flags := flag.NewFlagSet("generate module", flag.ContinueOnError)
	flags.SetOutput(stderr)
	specPath := flags.String("spec", "", "Versioned module spec (.yaml, .yml, or .json)")
	outputRoot := flags.String("out", "", "Output root for generated review artifacts")
	if err := flags.Parse(args[1:]); err != nil {
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
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.DefaultArgon2Params)
	mfaService := &auth.MFAService{Store: userStore, MasterKey: []byte(cfg.Security.MFAMasterKey)}
	recoveryService := &auth.PasswordRecovery{Users: userStore, Tokens: userStore, Sessions: sessionStore, ArgonParams: auth.DefaultArgon2Params}
	invitationService := &auth.InvitationService{Users: userStore, Invites: userStore, ArgonParams: auth.DefaultArgon2Params}
	authHTTP := auth.HTTPHandler{Service: authSvc, Audit: newAuditRecorder(db, cfg.Database.Dialect), Secure: cfg.Security.SecureCookies, MFA: mfaService, MFADB: db, Transactions: database.NewTransactor(db), LoginLimiter: security.NewAttemptLimiter(10000, 5, time.Minute), MFALimiter: security.NewAttemptLimiter(10000, 5, time.Minute), RecoveryLimiter: security.NewAttemptLimiter(10000, 3, time.Hour), RecoveryConfirmLimiter: security.NewAttemptLimiter(10000, 10, time.Hour), InviteAcceptLimiter: security.NewAttemptLimiter(10000, 10, time.Hour), Recovery: recoveryService, RecoveryDB: db, Invitation: invitationService, InviteDB: db}
	if cfg.Mail.Host != "" {
		sender, err := mailer.NewSMTP(mailer.SMTPConfig{Host: cfg.Mail.Host, Port: cfg.Mail.Port, From: cfg.Mail.From, Username: cfg.Mail.Username, Password: cfg.Mail.Password, StartTLS: cfg.Mail.StartTLS})
		if err != nil {
			return err
		}
		notifier := mailer.NotificationSender{Sender: sender}
		authHTTP.Notifier = notifier
		authHTTP.InviteNotice = notifier
	}
	contentService := pageservice.ContentService{
		Transactions: database.NewTransactor(db),
		Pages:        pagesstore.Store{Dialect: cfg.Database.Dialect},
		Audit:        auditstore.Store{Dialect: cfg.Database.Dialect},
		Outbox:       outboxstore.Store{Dialect: cfg.Database.Dialect},
		PublicCache:  cache.New[pages.Page](budget.CacheEntries, budget.CacheTTL),
		Sanitize:     ui.NewHTMLSanitizer().Sanitize,
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
	uploader, err := media.NewUploaderWithConcurrency(cfg.Media.Dir, budget.MediaMaxBytes, budget.WorkerConcurrency)
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
	newsWorker := newsservice.Worker{
		Durable: jobstore.Queue{Store: jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: budget.QueueEntries}, DB: db},
		Service: newsService,
		Lease:   time.Minute,
		Retry:   time.Minute,
	}
	graphqlHTTP := graphqlapi.Handler{DB: db, Pages: contentService}
	studioRegistry, err := adminstudio.NewRegistry([]adminstudio.Collection{
		{ID: "pages", Label: "Sayfalar", Description: "Taslak, bloklar ve yayınlanmış içerik.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}, {ID: "blocks", Label: "Bloklar", Kind: adminstudio.FieldRelation}}, Actions: []adminstudio.Action{{Label: "Tüm sayfalar", Href: "/admin/pages"}, {Label: "Ana sayfayı düzenle", Href: "/admin/pages/demo-home"}, {Label: "Showcase'i düzenle", Href: "/admin/pages/demo-showcase"}, {Label: "Hakkında sayfasını düzenle", Href: "/admin/pages/demo-about"}}},
		{ID: "news", Label: "Haberler", Description: "Taslak, zamanlama ve yayın akışı.", Fields: []adminstudio.Field{{ID: "locale", Label: "Dil", Kind: adminstudio.FieldText, Required: true}, {ID: "slug", Label: "Slug", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "title", Label: "Başlık", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "cover_media_id", Label: "Kapak görseli (herkese açık asset)", Kind: adminstudio.FieldText, Widget: "media"}, {ID: "excerpt", Label: "Kısa özet", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "key_takeaways", Label: "Önemli Noktalar (Key Takeaways)", Kind: adminstudio.FieldText, Widget: "list"}, {ID: "category", Label: "Kategori", Kind: adminstudio.FieldText}, {ID: "author_name", Label: "Yazar adı", Kind: adminstudio.FieldText}, {ID: "featured", Label: "Öne çıkan", Kind: adminstudio.FieldText, Widget: "checkbox"}, {ID: "tags", Label: "Etiketler", Kind: adminstudio.FieldText}, {ID: "seo_title", Label: "SEO başlığı", Kind: adminstudio.FieldText}, {ID: "seo_description", Label: "SEO açıklaması", Kind: adminstudio.FieldText, Widget: "textarea"}, {ID: "body", Label: "İçerik", Kind: adminstudio.FieldText, Widget: "textarea", Required: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, ReadOnly: true, VisibleInList: true}}, Actions: []adminstudio.Action{{Label: "Tüm haberler", Href: "/admin/news"}, {Label: "Release notes", Href: "/admin/news/demo-news"}, {Label: "Başlangıç rehberi", Href: "/admin/news/demo-news-guide"}}},
		{ID: "commerce", Label: "Ürünler", Description: "Katalog, görsel, fiyat, stok ve varyant yönetimi.", Fields: []adminstudio.Field{{ID: "title", Label: "Ürün adı", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "sku", Label: "SKU", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "media_id", Label: "Ürün görseli", Kind: adminstudio.FieldText, Widget: "media"}, {ID: "price_cents", Label: "Fiyat (kuruş)", Kind: adminstudio.FieldNumber, Required: true, VisibleInList: true}, {ID: "currency", Label: "Para birimi", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "stock", Label: "Stok", Kind: adminstudio.FieldNumber, Required: true, VisibleInList: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, Required: true, VisibleInList: true, Options: []string{"draft", "published"}}}, Actions: []adminstudio.Action{{Label: "Ürün kataloğu", Href: "/admin/commerce/products"}, {Label: "Varyant ve stok", Href: "/admin/commerce/products/demo-shirt/variants"}, {Label: "Siparişler ve iade", Href: "/admin/commerce/orders"}}},
		{ID: "commerce_variants", Label: "Varyantlar", Description: "Ürün varyantları ve stok seviyeleri.", Fields: []adminstudio.Field{{ID: "sku", Label: "SKU", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "title", Label: "Varyant adı", Kind: adminstudio.FieldText, Required: true, VisibleInList: true}, {ID: "price_cents", Label: "Fiyat", Kind: adminstudio.FieldNumber, Required: true, VisibleInList: true}, {ID: "currency", Label: "Para birimi", Kind: adminstudio.FieldText, ReadOnly: true, VisibleInList: true}, {ID: "stock", Label: "Stok", Kind: adminstudio.FieldNumber, Required: true, VisibleInList: true}, {ID: "status", Label: "Durum", Kind: adminstudio.FieldStatus, Required: true, VisibleInList: true, Options: []string{"draft", "published"}}}, Actions: []adminstudio.Action{{Label: "Ürün kataloğuna dön", Href: "/admin/commerce/products"}}},
	})
	if err != nil {
		return fmt.Errorf("admin studio registry: %w", err)
	}
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
	pageHTTP := pagetransport.Handler{DB: db, Content: contentService, Theme: selectedTheme, Media: mediaService, AdminCollection: pageCollection, DefaultLocale: cfg.I18N.DefaultLocale}
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
	newsHTTP := newstransport.Handler{Service: newsService, DB: db, AdminCollection: newsCollection, DefaultLocale: cfg.I18N.DefaultLocale}
	routes := append(authHTTP.RouterRoutes(), pageRouterRoutes(pageHTTP.Routes())...)
	routes = append(routes, router.Route{Pattern: "GET /login", Access: router.Public, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		returnTo := r.URL.Query().Get("return_to")
		if !strings.HasPrefix(returnTo, "/") || strings.HasPrefix(returnTo, "//") {
			returnTo = "/admin"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>HydroGo giriş</title><link rel="stylesheet" href="/theme/static/theme.css?v=20260910"></head><body><main class="theme-corporate-content login-page"><h1>Demo hesabıyla giriş</h1><p>Sepet, checkout ve admin ekranlarını deneyimlemek için giriş yapın.</p><form id="login-form"><input type="hidden" name="_csrf" value="%s"><label>E-posta <input name="email" type="email" required></label><label>Şifre <input name="password" type="password" required></label><button type="submit">Giriş yap</button><p id="login-status" role="status"></p></form><a href="%s">Mağazaya dön</a><script nonce="%s">document.getElementById('login-form').addEventListener('submit',async function(e){e.preventDefault();const form=e.currentTarget;const status=document.getElementById('login-status');status.textContent='Giriş yapılıyor…';const response=await fetch('/auth/login',{method:'POST',headers:{'X-CSRF-Token':form.elements._csrf.value},body:new URLSearchParams(new FormData(form))});if(response.ok){window.location.assign('%s');}else{status.textContent='Giriş başarısız. Demo bilgilerini kontrol edin.';}});</script></main></body></html>`, template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(returnTo), template.HTMLEscapeString(security.NonceFromContext(r.Context())), template.HTMLEscapeString(returnTo))
	})})
	adminDashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/admin/reports" {
			if err := ui.RenderAdminReports(w, ui.AdminDashboardData{Email: user.Email, Collections: studioRegistry.Collections(), Nonce: security.NonceFromContext(r.Context()), Active: "reports"}); err != nil {
				http.Error(w, "admin reports unavailable", http.StatusInternalServerError)
			}
			return
		}
		if err := ui.RenderAdminDashboard(w, ui.AdminDashboardData{Email: user.Email, Collections: studioRegistry.Collections(), Nonce: security.NonceFromContext(r.Context()), Active: "dashboard"}); err != nil {
			http.Error(w, "admin dashboard unavailable", http.StatusInternalServerError)
		}
	})
	routes = append(routes, router.Route{Pattern: "GET /admin", Access: router.Protected, Handler: adminDashboardHandler})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports", Access: router.Protected, Handler: adminDashboardHandler})
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
			if err := ui.RenderAdminArea(w, ui.AdminAreaData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), CSRF: security.CSRFTokenFromContext(r.Context()), Area: "developer"}); err != nil {
				http.Error(w, "developer tools unavailable", http.StatusInternalServerError)
			}
			return
		}
		if r.URL.Path == "/admin/account" {
			active = "account"
		}
		if r.URL.Path == "/admin/permissions" {
			active = "permissions"
		}
		var err error
		if active == "control" {
			err = ui.RenderAdminControl(w, ui.AdminControlData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), CSRF: security.CSRFTokenFromContext(r.Context()), Active: active})
		} else {
			err = ui.RenderAdminArea(w, ui.AdminAreaData{Email: user.Email, Nonce: security.NonceFromContext(r.Context()), CSRF: security.CSRFTokenFromContext(r.Context()), Area: active})
		}
		if err != nil {
			http.Error(w, "admin control center unavailable", http.StatusInternalServerError)
		}
	})
	for _, pattern := range []string{"GET /admin/control", "GET /admin/site", "GET /admin/security", "GET /admin/operations", "GET /admin/account", "GET /admin/permissions", "GET /admin/developer"} {
		routes = append(routes, router.Route{Pattern: pattern, Access: router.Protected, Handler: adminControlHandler})
	}
	routes = append(routes, mediaRouterRoutes(mediaHTTP.AdminRoutes())...)
	routes = append(routes, mediaRouterRoutes(mediaHTTP.Routes())...)
	routes = append(routes, siteRouterRoutes(siteHTTP.Routes())...)
	routes = append(routes, newsRouterRoutes(newsHTTP.Routes())...)
	if selectedTheme != nil {
		routes = append(routes, router.Route{Pattern: "GET /theme/static/{path...}", Access: router.Public, Handler: http.HandlerFunc(selectedTheme.ServeAsset)})
	}
	// Local editor assets are served from the versioned static vendor bundle.
	// The route is read-only and scoped below static/vendor by construction.
	routes = append(routes, router.Route{Pattern: "GET /vendor/{path...}", Access: router.Public, Handler: http.StripPrefix("/vendor/", http.FileServer(http.Dir("static/vendor")))})
	if moduleEnabled(cfg.Modules, "commerce") {
		commerceStore := commercestore.Store{Dialect: cfg.Database.Dialect}
		commerceService := commerce.Service{Transactions: database.NewTransactor(db), Store: commerceStore, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect}}
		commerceCatalog := commerce.CatalogService{Transactions: database.NewTransactor(db), Store: commerceStore, Variants: commerceStore}
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
		paymentService := commerce.PaymentService{Transactions: database.NewTransactor(db), Store: commerceStore, Providers: providers}
		commerceCollection, ok := studioRegistry.Collection("commerce")
		if !ok {
			return errors.New("commerce admin studio metadata missing")
		}
		commerceCollection.Fields = append(commerceCollection.Fields, adminstudio.Field{ID: "description", Label: "Ürün açıklaması", Kind: adminstudio.FieldText, Widget: "richtext"})
		commerceHTTP := commercetransport.Handler{DB: db, Commerce: commerceService, Catalog: commerceCatalog, CatalogVariants: commerceStore, Orders: commerceStore, Payments: providers, PaymentService: paymentService, Cart: commerce.CartService{Transactions: database.NewTransactor(db), Store: commerceStore}, AdminCollection: commerceCollection}
		variantCollection, ok := studioRegistry.Collection("commerce_variants")
		if !ok {
			return errors.New("commerce variants admin studio metadata missing")
		}
		commerceHTTP.VariantCollection = variantCollection
		paymentHTTP := commercetransport.PaymentWebhookHandler{Service: paymentService}
		routes = append(routes, commerceRouterRoutes(commerceHTTP.Routes())...)
		routes = append(routes, commerceRouterRoutes(paymentHTTP.Routes())...)
	}
	routes = append(routes, router.Route{Pattern: "POST /api/graphql", Access: router.Public, Handler: graphqlHTTP})

	metrics := observability.NewMetrics()
	metrics.SetWorkerConcurrency(budget.WorkerConcurrency)
	newsWorker.OnRetry = metrics.RecordWorkerRetry
	newsWorker.OnDeadLetter = metrics.RecordWorkerDeadLetter
	metrics.SetCacheStatsProvider(func() observability.CacheStats {
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
	go runNewsWorker(ctx, db, newsWorker, metrics, stderr, budget.WorkerConcurrency)
	webhookEndpoints := make([]webhooks.Endpoint, 0, len(cfg.Webhooks))
	for _, configured := range cfg.Webhooks {
		topics := make(map[string]struct{}, len(configured.Topics))
		for _, topic := range configured.Topics {
			topics[strings.TrimSpace(topic)] = struct{}{}
		}
		webhookEndpoints = append(webhookEndpoints, webhooks.Endpoint{ID: configured.ID, URL: configured.URL, Secret: configured.Secret, Topics: topics})
	}
	go runOutboxWorker(ctx, webhooks.Worker{DB: db, Store: outboxstore.Store{Dialect: cfg.Database.Dialect}, Dispatcher: webhooks.Dispatcher{Endpoints: webhookEndpoints}, WorkerID: "outbox-worker-0", Lease: time.Minute, RetryAfter: time.Minute}, stderr)
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

func moduleEnabled(modules []string, wanted string) bool {
	for _, module := range modules {
		if module == wanted {
			return true
		}
	}
	return false
}

func enabledModuleSet(modules []string) map[string]bool {
	result := make(map[string]bool, len(modules))
	for _, module := range modules {
		result[module] = true
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
	seedPage := func(id, slug, title, body, blocks string) error {
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES ($1,$2,1,'tr',$3,$4,'published',1,1,1,$5,$5,$6,$6) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO pages (id,title,version,locale,slug,body,status,revision,published_revision,publication_generation,blocks,published_blocks,created_at,updated_at) VALUES (?1,?2,1,'tr',?3,?4,'published',1,1,1,?5,?5,?6,?6)`, id, title, slug, body, blocks, now); err != nil {
			return err
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE pages SET locale='tr',slug=$1,title=$2,body=$3,blocks=$4,published_locale='tr',published_slug=$1,published_title=$2,published_body=$3,published_blocks=$4 WHERE id=$5`, `UPDATE pages SET locale='tr',slug=?1,title=?2,body=?3,blocks=?4,published_locale='tr',published_slug=?1,published_title=?2,published_body=?3,published_blocks=?4 WHERE id=?5`, slug, title, body, blocks, id); err != nil {
			return err
		}
		return seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES ($1,$2,1,'tr',$3,$4,$5,$6,$7) ON CONFLICT (page_id,revision) DO NOTHING`, `INSERT OR IGNORE INTO page_revisions (id,page_id,revision,locale,slug,title,body,blocks,created_at) VALUES (?1,?2,1,'tr',?3,?4,?5,?6,?7)`, id+"-rev-1", id, slug, title, body, blocks, now)
	}
	homeBlocks := `[{"type":"hero","payload":{"title":"Geleceğin Dijital Deneyimi: HydroGo","subtitle":"Modüler mimari, tip güvenli Go backend'i, tam yetenekli Admin Studio ve yüksek performanslı e-ticaret sözleşmeleri.","button_text":"Tüm Yetenekleri Keşfet","button_url":"/public/pages/tr/showcase/html"}},{"type":"cta","payload":{"title":"İçeriği ve Kataloğu Yönetin","button_text":"Admin Studio'ya Git →","button_url":"/admin"}},{"type":"testimonials","payload":{"items":[{"name":"Platform & Altyapı Ekip Lideri","quote":"HydroGo sayesinde içerik modellerimizi dakikalar içinde kurgulayıp doğrudan API üzerinden uygulamalarımıza servis edebiliyoruz."},{"name":"E-Ticaret Operasyon Yöneticisi","quote":"Varyant ve stok yönetimi ile sipariş akışı hem SQLite hem PostgreSQL üzerinde kusursuz çalışıyor."}]}},{"type":"faq","payload":{"items":[{"question":"HydroGo mimarisi hangi prensiplere dayanır?","answer":"Core ve Modül ayrımı, dialect bağımsız veri katmanı, outbox pattern ile asenkron olaylar ve sıfır dış bağımlılık prensibiyle geliştirilmiştir."},{"question":"Haberler ve sayfalar nasıl yönetilir?","answer":"Admin Studio içerisindeki zengin metin editörü (Quill), etiketler, anahtar çıkarımlar ve dinamik sayfa blokları ile tam kontrol sağlanır."},{"question":"E-Ticaret ve ödeme altyapısı hazır mı?","answer":"Evet; ürünler, varyantlar, sepet, sipariş ve Stripe, iyzico, PayTR ödeme sağlayıcı sözleşmeleri entegredir."}]}}]`
	showcaseBlocks := `[{"type":"hero","payload":{"title":"HydroGo Blok & Mimari Vitrini","subtitle":"Hero, CTA, Pricing Tablosu, Testimonials ve SSS (FAQ) bileşenleri ile dinamik web deneyimi.","button_text":"Mağazaya Göz At","button_url":"/shop"}},{"type":"pricing_table","payload":{"items":[{"name":"Starter Edition","price_cents":0,"currency":"TRY","features":["SQLite gömülü veritabanı","Admin Studio & Zengin Editör","REST & GraphQL API","Kompakt profil"]},{"name":"Growth Edition","price_cents":49900,"currency":"TRY","features":["Tüm CMS özellikleri","Commerce katalog ve varyantlar","Stok & Sipariş takibi","Headless Starter desteği"]},{"name":"Enterprise Edition","price_cents":149900,"currency":"TRY","features":["PostgreSQL yüksek erişilebilirlik","Arka plan worker & Outbox","Kapsamlı Audit & Kurtarma","Özelleştirilmiş SLA"]}]}},{"type":"testimonials","payload":{"items":[{"name":"Kıdemli Çözüm Mimarı","quote":"Admin Studio'nun blok bazlı sayfa kurgusu ve haberlerdeki key takeaways alanı editoryal üretkenliği üçe katladı."},{"name":"Başmühendis & Tech Lead","quote":"Veritabanı dialect testleri ve sıfır sızıntı politikası sayesinde deployment süreçlerimiz tamamen stressiz."}]}},{"type":"cta","payload":{"title":"Kendi projenizi HydroGo ile hemen başlatın","button_text":"Hemen Başla","button_url":"/public/news/tr/html"}},{"type":"faq","payload":{"items":[{"question":"PostgreSQL ve SQLite arasında geçiş nasıl yapılır?","answer":"Runtime yapılandırmasında database dialect parametresi ve connection string belirtilerek kolayca geçiş yapılır."},{"question":"Medya dosyaları nasıl optimize edilir?","answer":"Görsel ve dosyalar yerel veya güvenli medya deposunda SHA-256 bütünlük kontrolü ile saklanır."}]}}]`
	aboutBlocks := `[{"type":"hero","payload":{"title":"HydroGo Vizyonu ve Mimari Felsefesi","subtitle":"Performanslı, güvenli ve sürdürülebilir kurumsal içerik & ticaret altyapısı.","button_text":"Haber Merkezini Gör","button_url":"/public/news/tr/html"}},{"type":"cta","payload":{"title":"Geliştirici veya İçerik Yöneticisi misiniz?","button_text":"Studio Çalışma Alanı","button_url":"/admin"}},{"type":"faq","payload":{"items":[{"question":"Modüler CMS + Commerce nedir?","answer":"HydroCore genel sözleşmeleri tanımlar, Pages, News ve Commerce modülleri bağımsız olarak takılıp çalıştırılabilir."},{"question":"Güvenlik standartları nelerdir?","answer":"CSP, CSRF koruması, Argon2id parola şifreleme, TOTP MFA ve sıkı oturum izolasyonu mevcuttur."}]}}]`
	if err := seedPage("demo-home", "demo", "HydroGo Platformu", "HydroGo kurumsal içerik yönetim ve e-ticaret platformuna hoş geldiniz.", homeBlocks); err != nil {
		return fmt.Errorf("seed demo home: %w", err)
	}
	if err := seedPage("demo-showcase", "showcase", "HydroGo Showcase", "Tüm blok tiplerini ve mimari yetenekleri keşfedin.", showcaseBlocks); err != nil {
		return fmt.Errorf("seed demo showcase: %w", err)
	}
	if err := seedPage("demo-about", "about", "HydroGo Hakkında", "HydroCore, modüller ve Client Edition yaklaşımı.", aboutBlocks); err != nil {
		return fmt.Errorf("seed demo about: %w", err)
	}
	for _, product := range []struct {
		id, sku, title, desc string
		price, stock         int
	}{
		{"demo-bottle", "DEMO-BOTTLE", "HydroGo Akıllı Termos (750ml)", "Çift katmanlı paslanmaz çelik gövde, 24 saat sıcak/soğuk koruma ve dijital sıcaklık göstergesi.", 34900, 45},
		{"demo-shirt", "DEMO-SHIRT", "HydroGo Developer T-Shirt", "%100 organik pamuk, nefes alabilir kumaş ve modern geliştirici tasarım baskısı.", 49900, 30},
		{"demo-keyboard", "DEMO-KEYBOARD", "HydroGo Mekanik Klavye (Hot-Swap)", "Gateron Brown switch'ler, RGB aydınlatma, PBT tuş takımı ve alüminyum gövde.", 189900, 15},
		{"demo-backpack", "DEMO-BACKPACK", "HydroGo Tech Sırt Çantası", "Su geçirmez kumaş, 16 inç laptop bölmesi, USB şarj portu ve ergonomik sırt desteği.", 89900, 20},
	} {
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO commerce_products (id,sku,title,description,price_cents,currency,stock,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,'TRY',$6,'published',$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO commerce_products (id,sku,title,description,price_cents,currency,stock,status,created_at,updated_at) VALUES (?,?,?,?,?,'TRY',?,'published',?,?)`, product.id, product.sku, product.title, product.desc, product.price, product.stock, now, now); err != nil {
			return fmt.Errorf("seed product %s: %w", product.id, err)
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE commerce_products SET title=$1,description=$2,price_cents=$3,stock=$4,status='published',updated_at=$5 WHERE id=$6`, `UPDATE commerce_products SET title=?1,description=?2,price_cents=?3,stock=?4,status='published',updated_at=?5 WHERE id=?6`, product.title, product.desc, product.price, product.stock, now, product.id); err != nil {
			return fmt.Errorf("update seed product %s: %w", product.id, err)
		}
	}
	for _, variant := range []struct {
		id, product, sku, title string
		price, stock            int
	}{
		{"demo-shirt-blue-m", "demo-shirt", "DEMO-SHIRT-BLUE-M", "Okyanus Mavisi / M", 49900, 10},
		{"demo-shirt-black-l", "demo-shirt", "DEMO-SHIRT-BLACK-L", "Gece Siyahı / L", 49900, 12},
		{"demo-shirt-white-s", "demo-shirt", "DEMO-SHIRT-WHITE-S", "Saf Beyaz / S", 49900, 8},
		{"demo-bottle-matte", "demo-bottle", "DEMO-BOTTLE-MATTE", "Mat Siyah 750ml", 34900, 25},
		{"demo-bottle-silver", "demo-bottle", "DEMO-BOTTLE-SILVER", "Fırçalanmış Gümüş 750ml", 34900, 20},
		{"demo-kb-linear", "demo-keyboard", "DEMO-KB-RED", "Red Linear Switch (Sessiz)", 189900, 8},
		{"demo-kb-tactile", "demo-keyboard", "DEMO-KB-BROWN", "Brown Tactile Switch (Dengeli)", 189900, 7},
	} {
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO commerce_product_variants (id,product_id,sku,title,price_cents,stock,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,'published',$7,$7) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO commerce_product_variants (id,product_id,sku,title,price_cents,stock,status,created_at,updated_at) VALUES (?,?,?,?,?,?,'published',?,?)`, variant.id, variant.product, variant.sku, variant.title, variant.price, variant.stock, now, now); err != nil {
			return fmt.Errorf("seed variant %s: %w", variant.id, err)
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE commerce_product_variants SET title=$1,price_cents=$2,stock=$3,status='published',updated_at=$4 WHERE id=$5`, `UPDATE commerce_product_variants SET title=?1,price_cents=?2,stock=?3,status='published',updated_at=?4 WHERE id=?5`, variant.title, variant.price, variant.stock, now, variant.id); err != nil {
			return fmt.Errorf("update seed variant %s: %w", variant.id, err)
		}
	}
	for _, item := range []struct {
		id, slug, title, excerpt, takeaways, category, author, tags, body string
		featured                                                          bool
	}{
		{
			id:        "demo-news-release-v2",
			slug:      "hydrogo-2-0-yayinda",
			title:     "HydroGo 2.0 Yayında: Zengin Editör, Gelişmiş Güvenlik ve Modern UI",
			excerpt:   "Yeni sürüm ile birlikte Quill tabanlı zengin metin editörü, key takeaways ve etiket yönetimi, CSP sıkılaştırması ve baştan tasarlanan arayüz kullanıma sunuldu.",
			takeaways: "- Quill tabanlı zengin metin editörü tüm biçimlendirme araçlarıyla entegre edildi.\n- XSS ve SVG data URL vektörlerine karşı çok katmanlı sanitizasyon getirildi.\n- Kategori, etiket ve öne çıkan içerik rozetleri public haber odasına eklendi.\n- Dev server ve Admin Studio modern mavi tasarım sistemine kavuştu.",
			category:  "Ürün Güncellemeleri",
			author:    "HydroGo Editoryal",
			featured:  true,
			tags:      "hydrogo, release, security, richtext, go",
			body:      `{"ops":[{"insert":"HydroGo 2.0 İle İçerik Yönetimi Yeni Bir Boyuta Taşınıyor\n","attributes":{"header":1}},{"insert":"HydroGo ekibi olarak, platformun en kapsamlı güncellemesi olan 2.0 sürümünü duyurmaktan büyük heyecan duyuyoruz. Bu sürümde editoryal deneyimi güçlendirirken kurumsal güvenlik gereksinimlerini en üst seviyede karşılamayı hedefledik.\n\n"},{"insert":"Öne Çıkan Başlıca Yenilikler:\n","attributes":{"header":2}},{"insert":"Zengin Metin Editörü: ","attributes":{"bold":true}},{"insert":"Başlık hiyerarşisi, kalın/italik/altı çizili stiller, kod blokları, alıntılar ve medya entegrasyonu.\n"},{"insert":"Editoryal Metadata: ","attributes":{"bold":true}},{"insert":"Haber kategorileri, etiketler, yazar bilgisi, özet ve önemli çıkarımlar (Key Takeaways).\n"},{"insert":"Güvenlik Sertleştirmesi: ","attributes":{"bold":true}},{"insert":"Sıkı Content Security Policy (CSP), CSRF token koruması ve XSS temizleme filtreleri.\n\n"},{"insert":"\"Güvenilir, tip güvenli ve modern bir CMS deneyimi için HydroCore mimarisini her geçen gün daha da ileri taşıyoruz.\"\n","attributes":{"blockquote":true}},{"insert":"Geliştirici deneyimini keşfetmek için Admin Studio'ya giriş yapabilir veya API dokümantasyonunu inceleyebilirsiniz.\n"}]}`,
		},
		{
			id:        "demo-news-guide",
			slug:      "hydrogo-ile-baslangic-rehberi",
			title:     "Geliştiriciler İçin HydroGo Başlangıç ve Entegrasyon Rehberi",
			excerpt:   "Admin Studio üzerinden içerik ve e-ticaret kataloğu yönetimi, REST ve GraphQL API uç noktalarının tüketimi hakkında adım adım kılavuz.",
			takeaways: "- Admin Studio içerisinden Sayfalar, Haberler ve Commerce ürünleri yönetilebilir.\n- REST API üzerinden tüm yayınlanmış verilere anında JSON formatında erişilir.\n- Headless starter'lar bağımsız frontend mimarileri kurmayı kolaylaştırır.",
			category:  "Rehberler",
			author:    "HydroGo Core Team",
			featured:  false,
			tags:      "guide, tutorial, api, headless, cms",
			body:      `{"ops":[{"insert":"HydroGo Mimarisine Hızlı Bir Bakış\n","attributes":{"header":1}},{"insert":"HydroGo, modern web uygulamaları ve e-ticaret projeleri için geliştirilmiş yüksek performanslı bir Go CMS çekirdeğidir.\n\n"},{"insert":"1. Adım: Admin Studio ile İçerik Oluşturma\n","attributes":{"header":2}},{"insert":"Admin paneline erişerek yeni sayfalar oluşturabilir, hero, CTA, FAQ gibi blokları sürükleyip yapılandırabilir ve haberlerinizi yayına alabilirsiniz.\n\n"},{"insert":"2. Adım: API Uç Noktalarını Bağlama\n","attributes":{"header":2}},{"insert":"Frontend uygulamanızdan ","attributes":{}},{"insert":"/api/pages","attributes":{"code":true}},{"insert":" veya ","attributes":{}},{"insert":"/api/commerce/products","attributes":{"code":true}},{"insert":" adreslerine istek atarak içerikleri doğrudan tüketebilirsiniz.\n"}]}`,
		},
		{
			id:        "demo-news-security",
			slug:      "kurumsal-seviyede-cms-guvenligi",
			title:     "HydroGo Güvenlik Mimarisi: Çok Katmanlı Koruma ve Sıfır Sızıntı",
			excerpt:   "Argon2id parola şifreleme, TOTP MFA, CSP nonces ve SQL enjeksiyonlarına karşı parameterized query standartları.",
			takeaways: "- Tüm formlar dinamik CSRF token ve zaman aşımı korumalıdır.\n- Parolalar endüstri standardı Argon2id ile hash'lenir.\n- Çift veritabanı desteği (PostgreSQL & SQLite) kontrat testleri ile korunur.",
			category:  "Güvenlik & Mimari",
			author:    "Güvenlik Mühendisliği",
			featured:  false,
			tags:      "security, mfa, argon2id, csp, database",
			body:      `{"ops":[{"insert":"Kurumsal Güvenlik Standartlarımız\n","attributes":{"header":1}},{"insert":"HydroGo, finansal ve hassas veri taşıyan uygulamalarda güvenle çalışacak şekilde tasarlanmıştır. Kimlik doğrulama, yetkilendirme ve veri yaşam döngüsü kuralları ADR-0013 ve ADR-0014 standartlarına tam uyum sağlar.\n\n"},{"insert":"Denetim ve İzlenebilirlik:\n","attributes":{"header":2}},{"insert":"Tüm kritik yönetici hareketleri (içerik yayınlama, kullanıcı davetleri, iade işlemleri) değiştirilemez audit log tablosunda kayıt altına alınır.\n"}]}`,
		},
		{
			id:        "demo-news-commerce",
			slug:      "e-ticarette-stok-ve-varyant-yonetimi",
			title:     "E-Ticarette Yeni Dönem: Esnek Varyantlar, Stok Rezervasyonu ve Siparişler",
			excerpt:   "Tek bir ana ürün altında sınırsız varyant, otomatik stok takibi ve ödeme sağlayıcıları entegrasyonu.",
			takeaways: "- Renk, beden ve teknik özellik bazlı esnek varyant desteği.\n- Stok tükenmesi durumunda anında sepet koruması.\n- Stripe, iyzico ve PayTR ile sorunsuz ödeme alma kabiliyeti.",
			category:  "E-Ticaret",
			author:    "Commerce Ekibi",
			featured:  false,
			tags:      "commerce, products, variants, payments, shop",
			body:      `{"ops":[{"insert":"Esnek ve Güvenilir E-Ticaret Altyapısı\n","attributes":{"header":1}},{"insert":"Commerce modülü, modern bir e-ticaret operasyonunun ihtiyaç duyduğu tüm bileşenleri sunar. Müşterileriniz sepete ürün ekleyebilir, varyant seçebilir ve güvenli ödeme adımlarını tamamlayabilir.\n"}]}`,
		},
	} {
		featuredInt := 0
		if item.featured {
			featuredInt = 1
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES ($1,'tr',$2,$3,$4,$5,$6,$7,$8,$9,$10,'published',1,1,1,0,$11,$11) ON CONFLICT (id) DO NOTHING`, `INSERT OR IGNORE INTO news_items (id,locale,slug,title,excerpt,key_takeaways,category,author_name,featured,tags,body,status,version,revision,published_revision,schedule_generation,created_at,updated_at) VALUES (?1,'tr',?2,?3,?4,?5,?6,?7,?8,?9,?10,'published',1,1,1,0,?11,?11)`, item.id, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, featuredInt, item.tags, item.body, now); err != nil {
			return fmt.Errorf("seed news %s: %w", item.id, err)
		}
		if err := seedExec(ctx, db, cfg.Database.Dialect, `UPDATE news_items SET locale='tr',slug=$1,title=$2,excerpt=$3,key_takeaways=$4,category=$5,author_name=$6,featured=$7,tags=$8,body=$9,status='published',published_revision=revision WHERE id=$10`, `UPDATE news_items SET locale='tr',slug=?1,title=?2,excerpt=?3,key_takeaways=?4,category=?5,author_name=?6,featured=?7,tags=?8,body=?9,status='published',published_revision=revision WHERE id=?10`, item.slug, item.title, item.excerpt, item.takeaways, item.category, item.author, featuredInt, item.tags, item.body, item.id); err != nil {
			return fmt.Errorf("publish news %s: %w", item.id, err)
		}
	}
	if err := seedExec(ctx, db, cfg.Database.Dialect, `INSERT INTO site_settings (key,value,is_public,updated_at) VALUES ($1,$2,1,$3) ON CONFLICT (key) DO NOTHING`, `INSERT OR IGNORE INTO site_settings (key,value,is_public,updated_at) VALUES (?,?,1,?)`, "site.title", "HydroGo Demo", now); err != nil {
		return fmt.Errorf("seed setting: %w", err)
	}
	if err := seedExec(ctx, db, cfg.Database.Dialect, `DELETE FROM navigation_items WHERE id LIKE $1`, `DELETE FROM navigation_items WHERE id LIKE ?`, "demo-nav-%"); err != nil {
		return fmt.Errorf("remove legacy demo navigation: %w", err)
	}
	_, err = fmt.Fprintln(stdout, "demo-seeded pages=3 news=4 products=4 variants=7 navigation=0")
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
		_, err = fmt.Fprintf(stdout, "verified edition=%s core=%s sdk=%s artifacts=%d\n", verified.Edition, verified.CoreVersion, verified.SDKVersion, len(verified.Artifacts))
		return err
	default:
		return fmt.Errorf("unknown release subcommand %q", args[0])
	}
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
  hydro upgrade check [runtime flags]
  hydro create-admin --email EMAIL --password-stdin [runtime flags]
  hydro seed-demo [runtime flags]
  hydro doctor [--json] [runtime flags]
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
  hydro release verify --manifest PATH --public-key PATH
  hydro jobs dead-letter list [--limit N] [--json] [runtime flags]
  hydro jobs dead-letter requeue --id ID --confirm [runtime flags]
  hydro jobs dead-letter purge --older-than 720h --limit N --confirm [runtime flags]
  hydro serve [runtime flags]
  hydro help

Runtime flags:
  --config PATH
  --capacity-profile compact|standard|distributed
  --address HOST:PORT
  --database-dialect sqlite|postgres
  --database-dsn DSN

`)
}

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
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	"github.com/ardaozyildirim/hydrogo/migrations"
	commerce "github.com/ardaozyildirim/hydrogo/modules/commerce"
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
	pageHTTP := pagetransport.Handler{DB: db, Content: contentService}
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
	newsHTTP := newstransport.Handler{Service: newsService, DB: db}
	newsWorker := newsservice.Worker{
		Durable: jobstore.Queue{Store: jobstore.Store{Dialect: cfg.Database.Dialect, MaxEntries: budget.QueueEntries}, DB: db},
		Service: newsService,
		Lease:   time.Minute,
		Retry:   time.Minute,
	}
	routes := append(authHTTP.RouterRoutes(), pageRouterRoutes(pageHTTP.Routes())...)
	routes = append(routes, mediaRouterRoutes(mediaHTTP.Routes())...)
	routes = append(routes, siteRouterRoutes(siteHTTP.Routes())...)
	routes = append(routes, newsRouterRoutes(newsHTTP.Routes())...)
	if moduleEnabled(cfg.Modules, "commerce") {
		commerceStore := commercestore.Store{Dialect: cfg.Database.Dialect}
		commerceService := commerce.Service{Transactions: database.NewTransactor(db), Store: commerceStore, Outbox: outboxstore.Store{Dialect: cfg.Database.Dialect}}
		commerceHTTP := commercetransport.Handler{DB: db, Commerce: commerceService}
		paymentHTTP := commercetransport.PaymentWebhookHandler{Service: commerce.PaymentService{Transactions: database.NewTransactor(db), Store: commerceStore, Providers: map[string]commerce.PaymentProvider{}}}
		routes = append(routes, commerceRouterRoutes(commerceHTTP.Routes())...)
		routes = append(routes, commerceRouterRoutes(paymentHTTP.Routes())...)
	}

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
	logWriter := stderr
	if logWriter == nil {
		logWriter = io.Discard
	}
	logger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return server.Serve(ctx, server.Config{
		Address:         cfg.HTTP.Address,
		ShutdownTimeout: budget.ShutdownTimeout,
		AuthSvc:         authSvc,
		Metrics:         metrics,
		Logger:          logger,
		EnableHSTS:      cfg.Security.EnableHSTS,
		SecureCookie:    cfg.Security.SecureCookies,
		HTTPBodyLimit:   budget.HTTPBodyBytes,
		PublicOrigin:    cfg.HTTP.PublicOrigin,
		Routes:          routes,
		Ready: func(checkCtx context.Context) error {
			if err := db.PingContext(checkCtx); err != nil {
				return err
			}
			return database.CheckCompatibilityForModules(checkCtx, db, cfg.Database.Dialect, migrations.FS, enabledModuleSet(cfg.Modules))
		},
	})
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

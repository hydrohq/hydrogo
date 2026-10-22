package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
	"github.com/ardaozyildirim/hydrogo/internal/portable"
)

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
		if _, err := os.Stat(*path); err == nil { // #nosec G703 -- operator-supplied path from the command line, checked for symlinked directories
			return fmt.Errorf("portable export destination already exists: %s", *path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg, err := loadMaintenanceConfig(flags, values)
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
			_ = temporary.Close()
			return fmt.Errorf("protect portable export temp file: %w", err)
		}
		if _, err := temporary.Write(append(data, '\n')); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("write portable export: %w", err)
		}
		if err := temporary.Close(); err != nil {
			return fmt.Errorf("close portable export: %w", err)
		}
		if err := os.Rename(temporaryPath, *path); err != nil { // #nosec G703 -- operator-supplied path from the command line, checked for symlinked directories
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
		file, err := os.Open(*path) // #nosec G703 -- operator-supplied path from the command line, checked for symlinked directories
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
		cfg, err := loadMaintenanceConfig(flags, values)
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
	if info, err := os.Lstat(path); err == nil { // #nosec G703 -- operator-supplied path from the command line, checked for symlinked directories
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
	info, err := os.Lstat(path) // #nosec G703 -- operator-supplied path from the command line, checked for symlinked directories
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
	cfg, err := loadMaintenanceConfig(flags, values)
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
		cfg, err := loadMaintenanceConfig(flags, values)
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
	input, err := os.Open(source) // #nosec G304 -- operator-supplied snapshot path from the backup command
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("snapshot must be a regular file")
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- operator-supplied snapshot path from the backup command
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

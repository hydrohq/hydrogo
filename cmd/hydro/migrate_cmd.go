package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

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
	cfg, err := loadMaintenanceConfig(flags, values)
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
	cfg, err := loadMaintenanceConfig(flags, values)
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
			return fmt.Errorf("preserve SQLite sidecar: %w; %s", err, restoreOnFailure(dbPath, previous, movedSidecars))
		}
	}
	if err := os.Rename(temporary, dbPath); err != nil {
		return fmt.Errorf("activate rollback database: %w; %s", err, restoreOnFailure(dbPath, previous, movedSidecars))
	}
	if err := validateSQLiteFile(dbPath); err != nil {
		return fmt.Errorf("activated rollback database failed integrity check; prior database preserved at %s: %w", previous, err)
	}
	_, err = fmt.Fprintf(stdout, "rollback-complete database=%s snapshot=%s previous=%s\n", dbPath, snapshot, previous)
	return err
}

// restoreOnFailure undoes a partial rollback (the preserved original
// database and any moved WAL/SHM sidecars) and reports plainly if that
// restore itself fails, instead of swallowing it: a silently failed
// restore can leave dbPath missing entirely, which the caller's own error
// (about the step that triggered the restore) would never mention.
func restoreOnFailure(dbPath, previous string, movedSidecars []struct{ source, destination string }) string {
	var problems []string
	if err := os.Rename(previous, dbPath); err != nil {
		problems = append(problems, fmt.Sprintf("restoring %s from %s also failed: %v", dbPath, previous, err))
	}
	for _, sidecar := range movedSidecars {
		if err := os.Rename(sidecar.destination, sidecar.source); err != nil {
			problems = append(problems, fmt.Sprintf("restoring sidecar %s also failed: %v", sidecar.source, err))
		}
	}
	if len(problems) == 0 {
		return "the original database was restored"
	}
	return "ADDITIONALLY, " + strings.Join(problems, "; ") + " -- the database may now be missing or in an inconsistent state; recover manually from " + previous
}

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
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- migration source directory meant to be committed
		return err
	}
	metadata := fmt.Sprintf("id: %q\nmodule: %q\nclass:\n  sqlite: \"expand\"\n  postgres: \"expand\"\ntransactional:\n  sqlite: true\n  postgres: true\ndependencies: []\nchecksum_sha256:\n  sqlite: \"REPLACE_AFTER_REVIEW\"\n  postgres: \"REPLACE_AFTER_REVIEW\"\nschema:\n  readable_epoch: 1\n  writable_epoch: 1\ncompatibility: \"Describe additive/compatibility behavior before merge.\"\nrollback: \"No production down migration; restore or forward migration only.\"\n", *id, strings.TrimSpace(*module))
	files := map[string]string{"migration.yaml": metadata, "sqlite.sql": "-- Write the SQLite migration after review.\n", "postgres.sql": "-- Write the PostgreSQL migration after review.\n"}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil { // #nosec G306 -- migration source file meant to be committed
			return err
		}
	}
	_, err := fmt.Fprintf(stdout, "migration-created id=%s path=%s\n", *id, dir)
	return err
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
	cfg, err := loadMaintenanceConfig(flags, values)
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
	cfg, err := loadMaintenanceConfig(flags, values)
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
	cfg, err := loadMaintenanceConfig(flags, values)
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

package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	jobstore "github.com/ardaozyildirim/hydrogo/internal/jobs/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/release"
	securitystore "github.com/ardaozyildirim/hydrogo/internal/security/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

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
			if err := os.Chmod(cfg.Media.Dir, 0o750); err != nil { // #nosec G302 -- a directory needs the execute bit; 0750 is the access doctor --fix restores
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
	command := exec.CommandContext(ctx, "go", commandArgs...) // #nosec G204 -- fixed program (go); the arguments are this command's own flags, no shell
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
		return errors.New("runtime requires the verify subcommand")
	}
	if args[0] != "verify" {
		return errors.New("runtime requires the verify subcommand")
	}
	return releaseCommand(args, stdout, stderr)
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

func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path) // #nosec G304 -- operator-supplied path, read through a size limit
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

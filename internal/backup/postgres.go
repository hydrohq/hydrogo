package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type PostgresTools struct {
	DumpBinary    string
	RestoreBinary string
}

// SnapshotPostgres creates a custom-format database dump and copies immutable
// media into the same manifest root. The destination must not already exist.
func SnapshotPostgres(ctx context.Context, tools PostgresTools, dsn, mediaDir, destination string) (Manifest, error) {
	if strings.TrimSpace(destination) == "" || strings.TrimSpace(mediaDir) == "" {
		return Manifest{}, errors.New("media directory and destination are required")
	}
	mediaDir, err := filepath.Abs(mediaDir)
	if err != nil {
		return Manifest{}, err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return Manifest{}, err
	}
	if _, err := os.Stat(destination); err == nil {
		return Manifest{}, fmt.Errorf("destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	if err := os.MkdirAll(destination, 0o750); err != nil {
		return Manifest{}, err
	}
	cleanup := func() { _ = os.RemoveAll(destination) }
	databasePath := filepath.Join(destination, "database")
	if err := tools.Dump(ctx, dsn, databasePath); err != nil {
		cleanup()
		return Manifest{}, err
	}
	databaseDigest, err := digestFile(databasePath, "database")
	if err != nil {
		cleanup()
		return Manifest{}, err
	}
	media, err := digestTree(mediaDir)
	if err != nil {
		cleanup()
		return Manifest{}, err
	}
	for _, file := range media {
		target := filepath.Join(destination, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			cleanup()
			return Manifest{}, err
		}
		if err := copyFile(filepath.Join(mediaDir, filepath.FromSlash(strings.TrimPrefix(file.Path, "media/"))), target); err != nil {
			cleanup()
			return Manifest{}, err
		}
	}
	manifest := Manifest{Version: 1, CreatedAt: time.Now().UTC(), Database: databaseDigest, Media: media}
	if err := WriteManifest(filepath.Join(destination, "manifest.json"), manifest); err != nil {
		cleanup()
		return Manifest{}, err
	}
	if err := Verify(destination, manifest); err != nil {
		cleanup()
		return Manifest{}, fmt.Errorf("verify snapshot: %w", err)
	}
	return manifest, nil
}

// RestorePostgresToStaging restores a dump into an already-created empty
// PostgreSQL database and copies manifest-verified media into a new inactive
// staging root. Activation remains the caller's responsibility after applying
// restore controls and running smoke checks.
func RestorePostgresToStaging(ctx context.Context, tools PostgresTools, dsn, source, destination string, manifest Manifest, controls RestoreControls) error {
	if !controls.SessionsRevoked || !controls.EgressQuarantined || !controls.OldWritesBlocked {
		return errors.New("restore controls are incomplete")
	}
	if strings.TrimSpace(source) == "" || strings.TrimSpace(destination) == "" {
		return errors.New("restore source and destination are required")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err := validateRestorePaths(source, destination); err != nil {
		return err
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("restore destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := Verify(source, manifest); err != nil {
		return fmt.Errorf("verify restore source: %w", err)
	}
	if err := copyTree(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	if err := tools.Restore(ctx, dsn, filepath.Join(source, manifest.Database.Path)); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	marker := RestoreMarker{ManifestVersion: manifest.Version, VerifiedAt: time.Now().UTC(), Controls: controls}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	if err := os.WriteFile(filepath.Join(destination, "RESTORE_QUARANTINED.json"), append(data, '\n'), 0o640); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	return nil
}

func LocatePostgresTools() (PostgresTools, error) {
	dump := os.Getenv("HYDRO_PG_DUMP")
	if dump == "" {
		var err error
		dump, err = exec.LookPath("pg_dump")
		if err != nil {
			return PostgresTools{}, fmt.Errorf("pg_dump is required: %w", err)
		}
	}
	restore := os.Getenv("HYDRO_PG_RESTORE")
	if restore == "" {
		var err error
		restore, err = exec.LookPath("pg_restore")
		if err != nil {
			return PostgresTools{}, fmt.Errorf("pg_restore is required: %w", err)
		}
	}
	for name, path := range map[string]string{"pg_dump": dump, "pg_restore": restore} {
		if _, err := os.Stat(path); err != nil {
			return PostgresTools{}, fmt.Errorf("%s binary %q: %w", name, path, err)
		}
	}
	return PostgresTools{DumpBinary: dump, RestoreBinary: restore}, nil
}

func (t PostgresTools) Dump(ctx context.Context, dsn, destination string) error {
	if t.DumpBinary == "" {
		return errors.New("pg_dump binary is required")
	}
	return t.run(ctx, t.DumpBinary, "dump", dsn, destination)
}

func (t PostgresTools) Restore(ctx context.Context, dsn, source string) error {
	if t.RestoreBinary == "" {
		return errors.New("pg_restore binary is required")
	}
	return t.run(ctx, t.RestoreBinary, "restore", dsn, source)
}

func (t PostgresTools) run(ctx context.Context, binary, operation, dsn, path string) error {
	if strings.TrimSpace(dsn) == "" || strings.TrimSpace(path) == "" {
		return errors.New("PostgreSQL DSN and backup path are required")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" || parsed.User == nil {
		return errors.New("PostgreSQL DSN must be a URL with a user")
	}
	password, hasPassword := parsed.User.Password()
	username := parsed.User.Username()
	parsed.User = url.User(username)
	redactedDSN := parsed.String()
	passFile, err := os.CreateTemp("", "hydro-pgpass-*")
	if err != nil {
		return err
	}
	passPath := passFile.Name()
	defer os.Remove(passPath)
	if err := passFile.Chmod(0o600); err != nil {
		passFile.Close()
		return err
	}
	if hasPassword {
		host := parsed.Hostname()
		port := parsed.Port()
		if port == "" {
			port = "5432"
		}
		database := strings.TrimPrefix(parsed.Path, "/")
		fields := []string{host, port, database, username, password}
		for i, field := range fields {
			escaped, err := escapePGPassField(field)
			if err != nil {
				passFile.Close()
				return err
			}
			fields[i] = escaped
		}
		line := strings.Join(fields, ":") + "\n"
		if _, err := passFile.WriteString(line); err != nil {
			passFile.Close()
			return err
		}
	}
	if err := passFile.Close(); err != nil {
		return err
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var args []string
	if operation == "dump" {
		if _, err := os.Stat(absolutePath); err == nil {
			return fmt.Errorf("backup destination already exists: %s", absolutePath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		args = []string{"--format=custom", "--no-owner", "--file", absolutePath, redactedDSN}
	} else {
		if _, err := os.Stat(absolutePath); err != nil {
			return fmt.Errorf("backup source: %w", err)
		}
		args = []string{"--exit-on-error", "--no-owner", "--dbname", redactedDSN, absolutePath}
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Env = append(os.Environ(), "PGPASSFILE="+passPath)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %w: %s", operation, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func escapePGPassField(field string) (string, error) {
	if strings.ContainsAny(field, "\r\n") {
		return "", errors.New("PostgreSQL DSN contains a newline that cannot be represented in PGPASSFILE")
	}
	field = strings.ReplaceAll(field, `\`, `\\`)
	field = strings.ReplaceAll(field, ":", `\:`)
	return field, nil
}

func postgresAdminDSN(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.User == nil {
		return "", errors.New("PostgreSQL DSN must be a URL with a user")
	}
	parsed.Path = "/postgres"
	return parsed.String(), nil
}

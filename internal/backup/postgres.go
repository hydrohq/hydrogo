package backup

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PostgresTools struct {
	DumpBinary    string
	RestoreBinary string
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
		line := strings.Join([]string{host, port, database, username, password}, ":") + "\n"
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

func postgresAdminDSN(dsn string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.User == nil {
		return "", errors.New("PostgreSQL DSN must be a URL with a user")
	}
	parsed.Path = "/postgres"
	return parsed.String(), nil
}

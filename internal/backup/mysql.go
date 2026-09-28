package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
)

// MySQLTools uses the server-family client utilities for consistent InnoDB
// snapshots and restores. Install matching client tools on the backup host.
type MySQLTools struct {
	DumpBinary    string
	RestoreBinary string
}

func SnapshotMySQL(ctx context.Context, tools MySQLTools, dsn, mediaDir, destination string) (Manifest, error) {
	return snapshotDatabase(ctx, tools, "mysql", dsn, mediaDir, destination)
}

func RestoreMySQLToStaging(ctx context.Context, tools MySQLTools, dsn, source, destination string, manifest Manifest, controls RestoreControls) error {
	return restoreDatabaseToStaging(ctx, tools, "mysql", dsn, source, destination, manifest, controls)
}

func LocateMySQLTools() (MySQLTools, error) {
	dump := os.Getenv("HYDRO_MYSQLDUMP")
	if dump == "" {
		var err error
		dump, err = exec.LookPath("mysqldump")
		if err != nil {
			return MySQLTools{}, fmt.Errorf("mysqldump is required: %w", err)
		}
	}
	restore := os.Getenv("HYDRO_MYSQL")
	if restore == "" {
		var err error
		restore, err = exec.LookPath("mysql")
		if err != nil {
			return MySQLTools{}, fmt.Errorf("mysql client is required: %w", err)
		}
	}
	for name, path := range map[string]string{"mysqldump": dump, "mysql": restore} {
		if _, err := os.Stat(path); err != nil {
			return MySQLTools{}, fmt.Errorf("%s binary %q: %w", name, path, err)
		}
	}
	return MySQLTools{DumpBinary: dump, RestoreBinary: restore}, nil
}

func (t MySQLTools) Dump(ctx context.Context, dsn, destination string) error {
	return t.run(ctx, dsn, destination, true)
}

func (t MySQLTools) Restore(ctx context.Context, dsn, source string) error {
	return t.run(ctx, dsn, source, false)
}

func (t MySQLTools) run(ctx context.Context, dsn, path string, dump bool) error {
	if strings.TrimSpace(dsn) == "" || strings.TrimSpace(path) == "" {
		return errors.New("MySQL DSN and backup path are required")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil || cfg.User == "" || cfg.DBName == "" {
		return errors.New("MySQL DSN must include a user and database")
	}
	binary := t.RestoreBinary
	if dump {
		binary = t.DumpBinary
	}
	if binary == "" {
		return errors.New("MySQL client binary is required")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if dump {
		if _, err := os.Stat(absolutePath); err == nil {
			return fmt.Errorf("backup destination already exists: %s", absolutePath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else if info, err := os.Stat(absolutePath); err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = errors.New("backup source is not a regular file")
		}
		return fmt.Errorf("backup source: %w", err)
	}
	optionFile, err := mysqlClientOptionFile(cfg)
	if err != nil {
		return err
	}
	defer os.Remove(optionFile)
	args := []string{"--defaults-extra-file=" + optionFile}
	tlsArgs, err := mysqlTLSArgs(ctx, binary, cfg)
	if err != nil {
		return err
	}
	args = append(args, tlsArgs...)
	if dump {
		args = append(args, "--single-transaction", "--routines", "--events", "--triggers", "--hex-blob", "--no-tablespaces", "--result-file="+absolutePath, cfg.DBName)
	} else {
		args = append(args, cfg.DBName)
	}
	command := exec.CommandContext(ctx, binary, args...)
	var source *os.File
	if !dump {
		source, err = os.Open(absolutePath)
		if err != nil {
			return err
		}
		defer source.Close()
		command.Stdin = source
	}
	var output boundedCommandOutput
	output.remaining = maxPostgresCommandOutput
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		if dump {
			_ = os.Remove(absolutePath)
		}
		return fmt.Errorf("MySQL backup command failed: %w: %s", err, strings.TrimSpace(output.String()))
	}
	return nil
}

func mysqlTLSArgs(ctx context.Context, binary string, cfg *mysql.Config) ([]string, error) {
	if cfg.TLS != nil {
		return nil, errors.New("MySQL backup CLI cannot represent a custom TLS configuration")
	}
	command := exec.CommandContext(ctx, binary, "--version")
	var output boundedCommandOutput
	output.remaining = 1024
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("inspect MySQL client version: %w", err)
	}
	mariaDB := strings.Contains(strings.ToLower(output.String()), "mariadb")
	if cfg.TLSConfig == "false" || cfg.TLSConfig == "" {
		if mariaDB {
			return []string{"--skip-ssl"}, nil
		}
		return []string{"--ssl-mode=DISABLED"}, nil
	}
	if cfg.TLSConfig != "true" && cfg.TLSConfig != "preferred" {
		return nil, fmt.Errorf("MySQL backup CLI does not support TLS config %q", cfg.TLSConfig)
	}
	if mariaDB {
		return []string{"--ssl"}, nil
	}
	if cfg.TLSConfig == "preferred" {
		return []string{"--ssl-mode=PREFERRED"}, nil
	}
	return []string{"--ssl-mode=REQUIRED"}, nil
}

func mysqlClientOptionFile(cfg *mysql.Config) (string, error) {
	file, err := os.CreateTemp("", "hydro-mysql-client-*.cnf")
	if err != nil {
		return "", err
	}
	path := file.Name()
	cleanup := func(err error) (string, error) { _ = file.Close(); _ = os.Remove(path); return "", err }
	if err := file.Chmod(0o600); err != nil {
		return cleanup(err)
	}
	address := cfg.Addr
	host := address
	port := ""
	if cfg.Net == "tcp" {
		if parsedHost, parsedPort, err := net.SplitHostPort(address); err == nil {
			host, port = parsedHost, parsedPort
		}
	}
	for _, value := range []string{cfg.User, cfg.Passwd, host, address} {
		if strings.ContainsAny(value, "\r\n\x00") {
			return cleanup(errors.New("MySQL DSN contains a control character"))
		}
	}
	fields := []string{"user=" + mysqlOptionValue(cfg.User), "password=" + mysqlOptionValue(cfg.Passwd)}
	if cfg.Net == "tcp" {
		fields = append(fields, "host="+mysqlOptionValue(host), "port="+port)
	} else if cfg.Net == "unix" {
		fields = append(fields, "socket="+mysqlOptionValue(address))
	} else {
		return cleanup(fmt.Errorf("unsupported MySQL network %q", cfg.Net))
	}
	if _, err := io.WriteString(file, "[client]\n"+strings.Join(fields, "\n")+"\n"); err != nil {
		return cleanup(err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func mysqlOptionValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

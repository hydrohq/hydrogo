//go:build !hydro_compact

package database

import (
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

func prepareMySQLDSN(dsn string) (string, error) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	// Embedded MySQL migration bundles contain multiple trusted DDL statements.
	config.MultiStatements = true
	// Store code across the codebase scans DATETIME/TIMESTAMP columns expecting
	// a native time.Time (as Postgres and modernc.org/sqlite already return);
	// without ParseTime the driver hands back raw MySQL-format byte strings
	// that silently fail time.Parse(time.RFC3339Nano, ...) call sites.
	config.ParseTime = true
	config.Loc = time.UTC
	return config.FormatDSN(), nil
}

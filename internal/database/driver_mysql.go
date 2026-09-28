//go:build !hydro_compact

package database

import mysql "github.com/go-sql-driver/mysql"

func prepareMySQLDSN(dsn string) (string, error) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	// Embedded MySQL migration bundles contain multiple trusted DDL statements.
	config.MultiStatements = true
	return config.FormatDSN(), nil
}

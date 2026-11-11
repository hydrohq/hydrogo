//go:build !hydro_compact

package database

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func openPostgres(ctx context.Context, dsn string, options OpenOptions) (*sql.DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(stdlib.GetConnector(*config, stdlib.OptionResetSession(func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "DISCARD TEMP; RESET ALL;")
		return err
	})))
	db.SetMaxOpenConns(options.MaxOpenConns)
	db.SetMaxIdleConns(options.MaxIdleConns)
	return db, nil
}

//go:build !hydro_compact

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

func openPostgres(ctx context.Context, dsn string, options OpenOptions) (*sql.DB, error) {
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if options.MaxOpenConns > 0 {
		poolConfig.MaxConns = int32(options.MaxOpenConns)
	}
	if options.MaxIdleConns > 0 {
		poolConfig.MinConns = int32(options.MaxIdleConns)
	}

	// AfterRelease hook resets connection state when released back to the pool
	// to prevent any tenant context or session settings leak.
	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		resetCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := conn.Exec(resetCtx, "DISCARD TEMP; RESET ALL;")
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDBFromPool(pool, stdlib.OptionResetSession(func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "DISCARD TEMP; RESET ALL;")
		return err
	}))
	db.SetMaxOpenConns(options.MaxOpenConns)
	db.SetMaxIdleConns(options.MaxIdleConns)
	return db, nil
}

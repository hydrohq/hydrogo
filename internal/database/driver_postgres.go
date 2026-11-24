//go:build !hydro_compact

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/stdlib"
)

func openPostgres(ctx context.Context, dsn string, options OpenOptions) (*sql.DB, error) {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	// pgx's default only sets a deadline on the client socket when a context
	// is cancelled, so Postgres keeps running the query for a client that is
	// gone. Ask the server to cancel it; the socket deadline stays as fallback.
	// A per-statement limit can be set in the DSN (statement_timeout=30s).
	config.BuildContextWatcherHandler = func(conn *pgconn.PgConn) ctxwatch.Handler {
		return &pgconn.CancelRequestContextWatcherHandler{Conn: conn, DeadlineDelay: 2 * time.Second}
	}
	db := sql.OpenDB(stdlib.GetConnector(*config, stdlib.OptionResetSession(func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "DISCARD TEMP; RESET ALL;")
		return err
	})))
	db.SetMaxOpenConns(options.MaxOpenConns)
	db.SetMaxIdleConns(options.MaxIdleConns)
	return db, nil
}

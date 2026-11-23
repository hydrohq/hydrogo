package database

import (
	"context"
	"database/sql"
)

// DB represents an abstract database handle that supports queries, execution,
// transactions, and connection lifecycle management. *sql.DB implements it.
type DB interface {
	QueryExecutor
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	PingContext(ctx context.Context) error
	Close() error
}

package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"

	"modernc.org/sqlite"
)

// SQLite has one writer. With several pooled connections, waiting writers sit
// in SQLite's busy handler, which polls with sleeps: measured with 400
// concurrent read-then-write transactions, 10 connections ran at 2,956 tx/s
// (p99 82ms) against 21,175 tx/s (p99 9.5ms) on one connection, while reads
// scaled 6x with 10 connections. writeGate keeps the read concurrency and
// queues writers in Go instead: a write transaction, or a write outside one,
// holds the gate on its connection until it ends. Read-only transactions and
// queries outside a transaction never take it.
type writeGate chan struct{}

func (g writeGate) acquire(ctx context.Context) error {
	select {
	case g <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g writeGate) release() { <-g }

// openGatedSQLite opens dsn with every connection sharing one write gate.
func openGatedSQLite(dsn string) *sql.DB {
	return sql.OpenDB(gatedConnector{dsn: dsn, gate: make(writeGate, 1)})
}

type gatedConnector struct {
	dsn  string
	gate writeGate
}

func (c gatedConnector) Connect(context.Context) (driver.Conn, error) {
	inner, err := (&sqlite.Driver{}).Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return &gatedConn{inner: inner, gate: c.gate}, nil
}

func (c gatedConnector) Driver() driver.Driver { return &sqlite.Driver{} }

// innerConn is the part of modernc's connection the gate wraps.
type innerConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ExecerContext
	driver.QueryerContext
	driver.ConnPrepareContext
	driver.Pinger
	driver.SessionResetter
	driver.Validator
}

// gatedConn is used by one goroutine at a time (database/sql guarantees it),
// so inTx needs no lock.
type gatedConn struct {
	inner driver.Conn
	gate  writeGate
	inTx  bool
}

func (c *gatedConn) full() (innerConn, error) {
	full, ok := c.inner.(innerConn)
	if !ok {
		return nil, errors.New("sqlite driver connection lacks context methods")
	}
	return full, nil
}

func (c *gatedConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	full, err := c.full()
	if err != nil {
		return nil, err
	}
	if opts.ReadOnly {
		return full.BeginTx(ctx, opts)
	}
	if err := c.gate.acquire(ctx); err != nil {
		return nil, err
	}
	tx, err := full.BeginTx(ctx, opts)
	if err != nil {
		c.gate.release()
		return nil, err
	}
	c.inTx = true
	return &gatedTx{inner: tx, conn: c}, nil
}

func (c *gatedConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *gatedConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	full, err := c.full()
	if err != nil {
		return nil, err
	}
	if !c.inTx {
		if err := c.gate.acquire(ctx); err != nil {
			return nil, err
		}
		defer c.gate.release()
	}
	return full.ExecContext(ctx, query, args)
}

func (c *gatedConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	full, err := c.full()
	if err != nil {
		return nil, err
	}
	return full.QueryContext(ctx, query, args)
}

func (c *gatedConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	full, err := c.full()
	if err != nil {
		return nil, err
	}
	return full.PrepareContext(ctx, query)
}

func (c *gatedConn) Prepare(query string) (driver.Stmt, error) { return c.inner.Prepare(query) }

func (c *gatedConn) Close() error {
	if c.inTx {
		// A connection closed mid-transaction (driver.ErrBadConn) must not
		// keep the gate.
		c.inTx = false
		c.gate.release()
	}
	return c.inner.Close()
}

func (c *gatedConn) Ping(ctx context.Context) error {
	full, err := c.full()
	if err != nil {
		return err
	}
	return full.Ping(ctx)
}

func (c *gatedConn) ResetSession(ctx context.Context) error {
	full, err := c.full()
	if err != nil {
		return err
	}
	return full.ResetSession(ctx)
}

func (c *gatedConn) IsValid() bool {
	full, err := c.full()
	return err == nil && full.IsValid()
}

type gatedTx struct {
	inner driver.Tx
	conn  *gatedConn
	once  sync.Once
}

func (t *gatedTx) done() {
	t.once.Do(func() {
		if t.conn.inTx {
			t.conn.inTx = false
			t.conn.gate.release()
		}
	})
}

func (t *gatedTx) Commit() error {
	defer t.done()
	return t.inner.Commit()
}

func (t *gatedTx) Rollback() error {
	defer t.done()
	return t.inner.Rollback()
}

package database

import (
	"context"
	"database/sql"
	"errors"
	"runtime"
	"strings"
	"sync"
)

// DB represents an abstract database handle that supports queries, execution,
// transactions, and connection lifecycle management. Both *sql.DB and *SplitPool
// implement this interface.
type DB interface {
	QueryExecutor
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	PingContext(ctx context.Context) error
	Close() error
}

// SplitPool decouples read queries and write operations across two *sql.DB pools.
// For SQLite in WAL mode:
// - writer: serialized connection pool (MaxOpenConns = 1) for mutations and transactions
// - reader: multi-connection pool for concurrent SELECT queries
// For PostgreSQL and MySQL:
// - reader and writer share the same connection pool
type SplitPool struct {
	writer  *sql.DB
	reader  *sql.DB
	dialect Dialect
	writeMu sync.Mutex
}

// OpenSplit opens a database connection pool. For SQLite, it decouples the writer
// and reader connection pools to enable concurrent reads alongside serialized writes.
func OpenSplit(ctx context.Context, dialect Dialect, dsn string, options OpenOptions) (*SplitPool, error) {
	if dialect != SQLite {
		db, err := OpenWithOptions(ctx, dialect, dsn, options)
		if err != nil {
			return nil, err
		}
		return &SplitPool{
			writer:  db,
			reader:  db,
			dialect: dialect,
		}, nil
	}

	// 1. Writer pool (serialized: MaxOpenConns = 1)
	writerOptions := options
	writerOptions.MaxOpenConns = 1
	writerOptions.MaxIdleConns = 1
	writerDB, err := OpenWithOptions(ctx, SQLite, dsn, writerOptions)
	if err != nil {
		return nil, err
	}

	// 2. Reader pool (concurrent: MaxOpenConns = max(4, runtime.NumCPU() * 2))
	readerOptions := options
	readerOptions.ReadOnly = true
	numReaders := runtime.NumCPU() * 2
	if numReaders < 4 {
		numReaders = 4
	}
	if options.MaxOpenConns > 1 {
		numReaders = options.MaxOpenConns
	}
	readerOptions.MaxOpenConns = numReaders
	readerOptions.MaxIdleConns = numReaders / 2
	if readerOptions.MaxIdleConns < 2 {
		readerOptions.MaxIdleConns = 2
	}

	readerDB, err := OpenWithOptions(ctx, SQLite, dsn, readerOptions)
	if err != nil {
		_ = writerDB.Close()
		return nil, err
	}

	return &SplitPool{
		writer:  writerDB,
		reader:  readerDB,
		dialect: SQLite,
	}, nil
}

func (p *SplitPool) Writer() *sql.DB {
	return p.writer
}

func (p *SplitPool) Reader() *sql.DB {
	return p.reader
}

func (p *SplitPool) Dialect() Dialect {
	return p.dialect
}

func isReadOnly(query string) bool {
	q := strings.TrimSpace(query)
	for len(q) > 0 && (q[0] == '-' || q[0] == '/') {
		if strings.HasPrefix(q, "--") {
			if idx := strings.IndexByte(q, '\n'); idx >= 0 {
				q = strings.TrimSpace(q[idx+1:])
			} else {
				break
			}
		} else if strings.HasPrefix(q, "/*") {
			if idx := strings.Index(q, "*/"); idx >= 0 {
				q = strings.TrimSpace(q[idx+2:])
			} else {
				break
			}
		} else {
			break
		}
	}
	upper := strings.ToUpper(q)
	return strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "PRAGMA")
}

func (p *SplitPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.writer.ExecContext(ctx, query, args...)
}

func (p *SplitPool) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if isReadOnly(query) {
		return p.reader.QueryRowContext(ctx, query, args...)
	}
	return p.writer.QueryRowContext(ctx, query, args...)
}

func (p *SplitPool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if isReadOnly(query) {
		return p.reader.QueryContext(ctx, query, args...)
	}
	return p.writer.QueryContext(ctx, query, args...)
}

func (p *SplitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.writer.BeginTx(ctx, opts)
}

func (p *SplitPool) PingContext(ctx context.Context) error {
	if err := p.writer.PingContext(ctx); err != nil {
		return err
	}
	if p.reader != p.writer {
		return p.reader.PingContext(ctx)
	}
	return nil
}

func (p *SplitPool) Close() error {
	var errs []error
	if err := p.writer.Close(); err != nil {
		errs = append(errs, err)
	}
	if p.reader != p.writer {
		if err := p.reader.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

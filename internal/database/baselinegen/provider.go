package baselinegen

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	mysql "github.com/go-sql-driver/mysql"
)

// Provider hands out empty, throwaway databases of one dialect.
type Provider interface {
	Dialect() database.Dialect
	// New returns an empty database and a function that closes and removes it.
	New(ctx context.Context) (*sql.DB, func(), error)
}

// NewProvider returns a provider for dialect. adminDSN names a server account
// that may create and drop databases (ignored for SQLite, which uses temporary
// files under dir).
func NewProvider(dialect database.Dialect, adminDSN, dir string) (Provider, error) {
	switch dialect {
	case database.SQLite:
		return &sqliteProvider{dir: dir}, nil
	case database.Postgres:
		if adminDSN == "" {
			return nil, fmt.Errorf("a PostgreSQL DSN is required")
		}
		return &serverProvider{dialect: dialect, adminDSN: adminDSN}, nil
	case database.MySQL:
		if adminDSN == "" {
			return nil, fmt.Errorf("a MySQL DSN is required")
		}
		return &serverProvider{dialect: dialect, adminDSN: adminDSN}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q", dialect)
	}
}

func randomName() string {
	var suffix [6]byte
	_, _ = rand.Read(suffix[:])
	return "hydro_bl_" + hex.EncodeToString(suffix[:])
}

type sqliteProvider struct{ dir string }

func (p *sqliteProvider) Dialect() database.Dialect { return database.SQLite }

func (p *sqliteProvider) New(ctx context.Context) (*sql.DB, func(), error) {
	dir := p.dir
	if dir == "" {
		dir = os.TempDir()
	}
	path := filepath.Join(dir, randomName()+".sqlite")
	db, err := database.Open(ctx, database.SQLite, path)
	if err != nil {
		return nil, nil, err
	}
	return db, func() {
		_ = db.Close()
		for _, suffix := range []string{"", "-wal", "-shm"} {
			_ = os.Remove(path + suffix)
		}
	}, nil
}

type serverProvider struct {
	dialect  database.Dialect
	adminDSN string
}

func (p *serverProvider) Dialect() database.Dialect { return p.dialect }

func (p *serverProvider) New(ctx context.Context) (*sql.DB, func(), error) {
	name := randomName()
	admin, err := database.Open(ctx, p.dialect, p.adminDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s admin connection: %w", p.dialect, err)
	}
	var create, drop, dsn string
	switch p.dialect {
	case database.Postgres:
		parsed, err := url.Parse(p.adminDSN)
		if err != nil {
			admin.Close()
			return nil, nil, err
		}
		parsed.Path = "/" + name
		dsn = parsed.String()
		create = `CREATE DATABASE "` + name + `"`
		drop = `DROP DATABASE IF EXISTS "` + name + `" WITH (FORCE)`
	default:
		config, err := mysql.ParseDSN(p.adminDSN)
		if err != nil {
			admin.Close()
			return nil, nil, err
		}
		config.DBName = name
		dsn = config.FormatDSN()
		create = "CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4"
		drop = "DROP DATABASE IF EXISTS `" + name + "`"
	}
	if _, err := admin.ExecContext(ctx, create); err != nil {
		admin.Close()
		return nil, nil, fmt.Errorf("create %s database: %w", p.dialect, err)
	}
	db, err := database.Open(ctx, p.dialect, dsn)
	if err != nil {
		_, _ = admin.ExecContext(ctx, drop)
		admin.Close()
		return nil, nil, err
	}
	return db, func() {
		_ = db.Close()
		_, _ = admin.ExecContext(context.Background(), drop)
		_ = admin.Close()
	}, nil
}

// Package testmysql creates isolated databases for opt-in MySQL integration
// contracts. The configured account needs CREATE and DROP DATABASE privileges.
package testmysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	mysql "github.com/go-sql-driver/mysql"
)

func Open(t *testing.T) *sql.DB {
	t.Helper()
	if err := database.ValidateDialectAvailable(database.MySQL); err != nil {
		t.Skipf("MySQL test driver is unavailable in this build: %v", err)
	}
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse MySQL test URL: %v", err)
	}
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	databaseName := "hydro_contract_" + hex.EncodeToString(suffix[:])
	adminConfig := *cfg
	adminConfig.DBName = "mysql"
	admin, err := database.Open(context.Background(), database.MySQL, adminConfig.FormatDSN())
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	quoted := "`" + databaseName + "`"
	if _, err := admin.ExecContext(context.Background(), "CREATE DATABASE "+quoted+" CHARACTER SET utf8mb4"); err != nil {
		_ = admin.Close()
		t.Fatalf("create MySQL test database: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.Close(); err != nil {
			t.Errorf("close MySQL admin connection: %v", err)
		}
		cleanupConfig := *cfg
		cleanupConfig.DBName = "mysql"
		cleanup, err := database.Open(context.Background(), database.MySQL, cleanupConfig.FormatDSN())
		if err != nil {
			t.Errorf("open MySQL cleanup connection: %v", err)
			return
		}
		defer cleanup.Close()
		if _, err := cleanup.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+quoted); err != nil {
			t.Errorf("drop MySQL test database: %v", err)
		}
	})

	testConfig := *cfg
	testConfig.DBName = databaseName
	db, err := database.Open(context.Background(), database.MySQL, testConfig.FormatDSN())
	if err != nil {
		t.Fatalf("open isolated MySQL test database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close isolated MySQL test database: %v", err)
		}
	})
	return db
}

func MigrateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := database.ApplyMigrations(context.Background(), db, database.MySQL, migrations.FS); err != nil {
		t.Fatalf("apply MySQL test migrations: %v", err)
	}
}

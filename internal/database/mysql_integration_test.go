package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestMySQLPoolAndTransactionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := db.Stats().MaxOpenConnections; got != 10 {
		t.Fatalf("default MySQL pool max = %d, want 10", got)
	}

	bounded, err := database.OpenWithOptions(ctx, database.MySQL, dsn, database.OpenOptions{MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer bounded.Close()
	if got := bounded.Stats().MaxOpenConnections; got != 4 {
		t.Fatalf("configured MySQL pool max = %d, want 4", got)
	}
	name := fmt.Sprintf("hydro_mysql_tx_%d", os.Getpid())
	if _, err := bounded.ExecContext(ctx, "CREATE TABLE "+name+" (id INT PRIMARY KEY) ENGINE=InnoDB"); err != nil {
		t.Fatal(err)
	}
	defer bounded.ExecContext(context.Background(), "DROP TABLE "+name)
	wantErr := errors.New("rollback contract")
	err = database.NewTransactor(bounded).Within(ctx, func(tx database.Executor) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO "+name+" (id) VALUES (?)", 1); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("transaction error = %v, want %v", err, wantErr)
	}
	var count int
	if err := bounded.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+name).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rows after rollback = %d, want 0", count)
	}
}

func TestMySQLMigrationRunnerContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_MYSQL_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_MYSQL_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.MySQL, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var ledgerExists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='schema_migrations'`).Scan(&ledgerExists); err != nil {
		t.Fatal(err)
	}
	if ledgerExists != 0 {
		t.Skip("migration runner contract requires an empty dedicated MySQL database")
	}
	table := fmt.Sprintf("hydro_mysql_migration_%d", os.Getpid())
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table)
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS schema_migrations")
	bundle := fstest.MapFS{
		"000000000001_mysql_contract/migration.yaml": &fstest.MapFile{Data: []byte("module: core\ntransactional:\n  mysql: false\n")},
		"000000000001_mysql_contract/mysql.sql":      &fstest.MapFile{Data: []byte("CREATE TABLE " + table + " (id VARCHAR(64) PRIMARY KEY) ENGINE=InnoDB; CREATE INDEX " + table + "_id_idx ON " + table + " (id);")},
	}
	for range 2 {
		if err := database.ApplyMigrations(ctx, db, database.MySQL, bundle); err != nil {
			t.Fatal(err)
		}
	}
	pending, err := database.PendingMigrations(ctx, db, database.MySQL, bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending MySQL migrations = %v, want none", pending)
	}
	if err := database.CheckCompatibility(ctx, db, database.MySQL, bundle); err != nil {
		t.Fatalf("MySQL migration compatibility: %v", err)
	}
}

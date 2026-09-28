package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
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

func TestMySQLNavigationMigrationContract(t *testing.T) {
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
		t.Skip("navigation migration contract requires an empty dedicated MySQL database")
	}
	table := fmt.Sprintf("hydro_mysql_navigation_%d", os.Getpid())
	baseTable := table + "_base"
	if _, err := db.ExecContext(ctx, "CREATE TABLE "+table+" (id VARCHAR(128) PRIMARY KEY) ENGINE=InnoDB"); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+table)
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS schema_migrations")
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+baseTable)
	metadata, err := migrations.FS.ReadFile("202609280023_navigation_tree/migration.yaml")
	if err != nil {
		t.Fatal(err)
	}
	script, err := migrations.FS.ReadFile("202609280023_navigation_tree/mysql.sql")
	if err != nil {
		t.Fatal(err)
	}
	script = []byte(strings.ReplaceAll(string(script), "navigation_items", table))
	bundle := fstest.MapFS{
		"000000000001_mysql_contract/migration.yaml":  &fstest.MapFile{Data: []byte("module: core\ntransactional:\n  mysql: false\n")},
		"000000000001_mysql_contract/mysql.sql":       &fstest.MapFile{Data: []byte("CREATE TABLE " + baseTable + " (id VARCHAR(64) PRIMARY KEY) ENGINE=InnoDB")},
		"202609280023_navigation_tree/migration.yaml": &fstest.MapFile{Data: metadata},
		"202609280023_navigation_tree/mysql.sql":      &fstest.MapFile{Data: script},
	}
	for range 2 {
		if err := database.ApplyMigrations(ctx, db, database.MySQL, bundle); err != nil {
			t.Fatal(err)
		}
	}
	for column, length := range map[string]int{"parent_id": 128, "target": 8} {
		var dataType string
		var actualLength int
		if err := db.QueryRowContext(ctx, `SELECT data_type,character_maximum_length FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`, table, column).Scan(&dataType, &actualLength); err != nil {
			t.Fatal(err)
		}
		if dataType != "varchar" || actualLength != length {
			t.Errorf("%s type = %s(%d), want varchar(%d)", column, dataType, actualLength, length)
		}
	}
}

func TestMySQLSchemaDiffFindsPhysicalDrift(t *testing.T) {
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
	var tableCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Skip("schema diff contract requires an empty dedicated MySQL database")
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE parent (id VARCHAR(64) PRIMARY KEY, name VARCHAR(64) NOT NULL) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE child (id VARCHAR(64) PRIMARY KEY, parent_id VARCHAR(64) NOT NULL) ENGINE=InnoDB`); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS child")
	defer db.ExecContext(context.Background(), "DROP TABLE IF EXISTS parent")
	bundle := fstest.MapFS{
		"001_reference/migration.yaml": &fstest.MapFile{Data: []byte("module: core\nclass:\n  mysql: expand\ntransactional:\n  mysql: false\n")},
		"001_reference/mysql.sql":      &fstest.MapFile{Data: []byte("CREATE TABLE parent (id VARCHAR(64) PRIMARY KEY, name VARCHAR(128) NOT NULL) ENGINE=InnoDB; CREATE INDEX idx_parent_name ON parent(name); CREATE TABLE child (id VARCHAR(64) PRIMARY KEY, parent_id VARCHAR(64) NOT NULL, CONSTRAINT fk_child_parent FOREIGN KEY (parent_id) REFERENCES parent(id)) ENGINE=InnoDB")},
	}
	changes, err := database.MySQLSchemaDiff(ctx, db, bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"definition_mismatch:column:parent.name":   false,
		"missing:index:parent.idx_parent_name":     false,
		"missing:constraint:child.fk_child_parent": false,
	}
	for _, change := range changes {
		key := change.Kind + ":" + change.Object + ":" + change.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("schema diff missing %s; got %#v", key, changes)
		}
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

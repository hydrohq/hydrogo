package portable

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteContentExportImportRoundTripExcludesAuthState(t *testing.T) {
	ctx := context.Background()
	open := func(name string) *sql.DB {
		db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
			db.Close()
			t.Fatal(err)
		}
		return db
	}
	source := open("source.sqlite")
	defer source.Close()
	target := open("target.sqlite")
	defer target.Close()
	if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('p1','Home',2,'en','home','body','published',2,1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('r1','p1',2,'en','home','Home','body','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.ExecContext(ctx, `INSERT INTO users (id,email,password_hash,role,active,created_at,updated_at) VALUES ('u1','u@example.test','secret-hash','admin',1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	document, err := Export(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-hash") || strings.Contains(string(data), "password_hash") || strings.Contains(string(data), "sessions") {
		t.Fatalf("export contains auth state: %s", data)
	}
	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.NewTransactor(target).Within(ctx, func(tx database.Executor) error {
		return Import(ctx, tx, decoded, database.SQLite)
	}); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := target.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = 'p1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Home" {
		t.Fatalf("imported title = %q", title)
	}
	var users int
	if err := target.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 0 {
		t.Fatalf("auth users imported: %d", users)
	}
}

func TestPostgresContentExportImportRoundTrip(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	db, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	sourceSchema := "portable_src_" + suffix
	targetSchema := "portable_dst_" + suffix
	for _, schema := range []string{sourceSchema, targetSchema} {
		if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
			t.Fatal(err)
		}
		for _, spec := range specs {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.%s AS TABLE public.%s WITH NO DATA", schema, spec.name, spec.name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DROP SCHEMA "+sourceSchema+" CASCADE")
		_, _ = db.ExecContext(ctx, "DROP SCHEMA "+targetSchema+" CASCADE")
	}()
	sourceTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sourceTx.ExecContext(ctx, "SET LOCAL search_path TO "+sourceSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := sourceTx.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('pg-p1','Postgres',2,'en','pg','body','published',2,1,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sourceTx.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('pg-r1','pg-p1',2,'en','pg','Postgres','body','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	document, err := Export(ctx, sourceTx)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourceTx.Rollback(); err != nil {
		t.Fatal(err)
	}
	targetTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := targetTx.ExecContext(ctx, "SET LOCAL search_path TO "+targetSchema); err != nil {
		t.Fatal(err)
	}
	if err := Import(ctx, targetTx, document, database.Postgres); err != nil {
		t.Fatal(err)
	}
	if err := targetTx.Commit(); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := db.QueryRowContext(ctx, "SELECT title FROM "+targetSchema+".pages WHERE id = 'pg-p1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Postgres" {
		t.Fatalf("imported PostgreSQL title = %q", title)
	}
}

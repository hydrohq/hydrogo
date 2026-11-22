package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/migrations"
)

func TestPortableNDJSONExportAndImportCLI(t *testing.T) {
	ctx := context.Background()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(dir, "src.sqlite")
	dstPath := filepath.Join(dir, "dst.sqlite")
	ndjsonPath := filepath.Join(dir, "export.ndjson")

	srcDB, err := database.Open(ctx, database.SQLite, srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, srcDB, database.SQLite, migrations.FS); err != nil {
		srcDB.Close()
		t.Fatal(err)
	}
	if _, err := srcDB.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('p1','CLI Page',1,'en','cli-page','body','published',1,1,'2026-01-01','2026-01-01')`); err != nil {
		srcDB.Close()
		t.Fatal(err)
	}
	srcDB.Close()

	dstDB, err := database.Open(ctx, database.SQLite, dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, dstDB, database.SQLite, migrations.FS); err != nil {
		dstDB.Close()
		t.Fatal(err)
	}
	dstDB.Close()

	var stdout, stderr bytes.Buffer
	err = run([]string{
		"portable", "export",
		"--database-dialect", "sqlite",
		"--database-dsn", srcPath,
		"--format", "ndjson",
		"--scope", "all",
		"--path", ndjsonPath,
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("portable export ndjson failed: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "portable-exported") || !strings.Contains(stdout.String(), "format=ndjson") {
		t.Fatalf("unexpected export output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"portable", "import",
		"--format", "ndjson",
		"--dry-run",
		"--path", ndjsonPath,
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("portable import dry-run failed: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "portable-import-valid") {
		t.Fatalf("unexpected dry-run output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"portable", "import",
		"--database-dialect", "sqlite",
		"--database-dsn", dstPath,
		"--format", "ndjson",
		"--confirm",
		"--path", ndjsonPath,
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("portable import confirm failed: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "portable-imported") {
		t.Fatalf("unexpected import output: %s", stdout.String())
	}

	verifyDB, err := database.Open(ctx, database.SQLite, dstPath)
	if err != nil {
		t.Fatal(err)
	}
	defer verifyDB.Close()
	var title string
	if err := verifyDB.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = 'p1'").Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "CLI Page" {
		t.Fatalf("imported title = %q, want %q", title, "CLI Page")
	}
}

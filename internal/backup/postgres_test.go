package backup

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestBoundedPostgresCommandOutput(t *testing.T) {
	var output boundedCommandOutput
	output.remaining = maxPostgresCommandOutput
	input := bytes.Repeat([]byte("x"), maxPostgresCommandOutput+1024)
	if written, err := output.Write(input); err != nil || written != len(input) {
		t.Fatalf("bounded output write = %d/%v", written, err)
	}
	if output.buffer.Len() != maxPostgresCommandOutput {
		t.Fatalf("bounded output size = %d, want %d", output.buffer.Len(), maxPostgresCommandOutput)
	}
	if !output.truncated || !bytes.HasSuffix([]byte(output.String()), []byte("[postgres command output truncated]")) {
		t.Fatal("bounded output did not record truncation")
	}
}

func TestRestorePostgresToStagingRejectsEmptyPaths(t *testing.T) {
	controls := RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}
	if err := RestorePostgresToStaging(context.Background(), PostgresTools{}, "", " ", "", Manifest{}, controls); err == nil {
		t.Fatal("postgres restore accepted empty source and destination")
	}
}

func TestPostgresDumpRestoreContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	tools, err := LocatePostgresTools()
	if err != nil {
		t.Skip(err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// pg_dump copies every schema of a database, so the source must be a database
	// of its own: another contract's schema on the shared one (with forced row
	// security) would fail the dump.
	adminDSN, err := postgresAdminDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := database.Open(ctx, database.Postgres, adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	sourceDB := "hydro_dump_source_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+sourceDB); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+sourceDB+" WITH (FORCE)")
	}()
	dsn = replacePostgresDatabase(t, dsn, sourceDB)
	source, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := database.ApplyMigrations(ctx, source, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}
	// The dump contract exercises snapshot/restore mechanics separately from
	// the RLS isolation contract. In this disposable database the table owner
	// temporarily removes FORCE so pg_dump can read a complete snapshot;
	// production uses the explicitly provisioned privileged backup role.
	rows, err := source.QueryContext(ctx, `SELECT quote_ident(relname) FROM pg_class WHERE relnamespace = 'public'::regnamespace AND relkind = 'r' AND relrowsecurity AND relforcerowsecurity`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatal(err)
		}
		if _, err := source.ExecContext(ctx, "ALTER TABLE "+table+" NO FORCE ROW LEVEL SECURITY"); err != nil {
			t.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	marker := "pg-restore-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := source.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ($1,$2,1,'en',$3,'restore-body','draft',1,0,NOW(),NOW())`, marker, "restore marker", marker); err != nil {
		t.Fatal(err)
	}
	mediaDir := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(mediaDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "hero.webp"), []byte("media-restore-marker"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshotDir := filepath.Join(t.TempDir(), "snapshot")
	manifest, err := SnapshotPostgres(ctx, tools, dsn, mediaDir, snapshotDir)
	if err != nil {
		t.Fatal(err)
	}
	targetDB := "hydro_restore_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+targetDB); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+targetDB+" WITH (FORCE)")
	}()
	targetURL := replacePostgresDatabase(t, dsn, targetDB)
	staging := filepath.Join(t.TempDir(), "staging")
	controls := RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}
	if err := RestorePostgresToStaging(ctx, tools, targetURL, snapshotDir, staging, manifest, controls); err != nil {
		t.Fatal(err)
	}
	target, err := database.Open(ctx, database.Postgres, targetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	var title string
	if err := target.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = $1", marker).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "restore marker" {
		t.Fatalf("restored title = %q", title)
	}
	media, err := os.ReadFile(filepath.Join(staging, "media", "hero.webp"))
	if err != nil {
		t.Fatal(err)
	}
	if string(media) != "media-restore-marker" {
		t.Fatalf("restored media = %q", media)
	}
	if err := ApplyRestoreControls(ctx, target, database.Postgres, controls); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staging, "RESTORE_QUARANTINED.json")); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(t.TempDir(), "active")
	rollback := filepath.Join(t.TempDir(), "rollback")
	if err := os.MkdirAll(active, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(active, "old.css"), []byte("old-release"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := ActivateStagingMedia(staging, active, rollback); err != nil {
		t.Fatal(err)
	}
	if restored, err := os.ReadFile(filepath.Join(active, "media", "hero.webp")); err != nil {
		t.Fatal(err)
	} else if string(restored) != "media-restore-marker" {
		t.Fatalf("activated media = %q", restored)
	}
	if retained, err := os.ReadFile(filepath.Join(rollback, "old.css")); err != nil {
		t.Fatal(err)
	} else if string(retained) != "old-release" {
		t.Fatalf("retained asset = %q", retained)
	}
}

func TestEscapePGPassField(t *testing.T) {
	got, err := escapePGPassField(`pa:ss\\word`)
	if err != nil {
		t.Fatal(err)
	}
	if got != `pa\:ss\\\\word` {
		t.Fatalf("escaped pgpass field = %q", got)
	}
	if _, err := escapePGPassField("line\nbreak"); err == nil {
		t.Fatal("newline accepted in pgpass field")
	}
}

func replacePostgresDatabase(t *testing.T, dsn, databaseName string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + databaseName
	return parsed.String()
}

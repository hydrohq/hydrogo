package backup

import (
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
	source, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if err := database.ApplyMigrations(ctx, source, database.Postgres, migrations.FS); err != nil {
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
	adminDSN, err := postgresAdminDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := database.Open(ctx, database.Postgres, adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	targetDB := "hydro_restore_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+targetDB); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	defer func() {
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE "+targetDB)
		_ = admin.Close()
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

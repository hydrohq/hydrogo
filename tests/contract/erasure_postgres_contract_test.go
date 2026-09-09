package contract_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/erasure"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestPostgresSQLAdapterReplayIsIdempotentForPagesAndMedia(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}

	ctx := context.Background()
	admin, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "erasure_contract_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	defer func() {
		_, _ = admin.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE")
		_ = admin.Close()
	}()

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()

	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(ctx, db, database.Postgres, migrations.FS); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('pg-erase-page','Title',1,'en','erase','body','draft',1,0,NOW(),NOW())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('pg-erase-rev','pg-erase-page',1,'en','erase','Title','body',NOW())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES ('pg-erase-media','erase/key','x.png','image/png',1,0,0,'private','ready',NOW(),NOW())`); err != nil {
		t.Fatal(err)
	}

	log, err := erasure.New(filepath.Join(t.TempDir(), "erasure.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range []erasure.Intent{
		{ID: "erase-pg-page-op", EditionID: "site", TargetID: "pg-erase-page", Kind: "erase.page"},
		{ID: "erase-pg-media-op", EditionID: "site", TargetID: "pg-erase-media", Kind: "erase.media"},
	} {
		if _, err := log.Append(intent); err != nil {
			t.Fatal(err)
		}
	}

	adapter := erasure.SQLAdapter{Dialect: database.Postgres}
	checkpoint, err := log.Replay(ctx, 0, func(ctx context.Context, intent erasure.Intent) error {
		return database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return adapter.Apply(ctx, tx, intent) })
	})
	if err != nil || checkpoint.Sequence != 2 {
		t.Fatalf("first postgres adapter replay = %#v/%v", checkpoint, err)
	}

	// Replay again to confirm idempotency
	if _, err := log.Replay(ctx, 0, func(ctx context.Context, intent erasure.Intent) error {
		return database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return adapter.Apply(ctx, tx, intent) })
	}); err != nil {
		t.Fatalf("idempotent postgres replay: %v", err)
	}

	var pages, revisions, media int
	for table, target := range map[string]*int{"pages": &pages, "page_revisions": &revisions, "media_assets": &media} {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if pages != 0 || revisions != 0 || media != 0 {
		t.Fatalf("erased rows in postgres = pages:%d revisions:%d media:%d", pages, revisions, media)
	}
}

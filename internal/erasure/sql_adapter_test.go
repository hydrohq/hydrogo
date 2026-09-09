package erasure

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLAdapterRejectsTypedNilExecutor(t *testing.T) {
	var db *sql.DB
	err := (SQLAdapter{Dialect: database.SQLite}).Apply(context.Background(), db, Intent{EditionID: "site", TargetID: "asset", Kind: "erase.media"})
	if err == nil {
		t.Fatal("typed nil erasure executor was accepted")
	}
}

func TestSQLiteSQLAdapterReplayIsIdempotentForPagesAndMedia(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "erasure.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES ('erase-page','Title',1,'en','erase','body','draft',1,0,'2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO page_revisions (id,page_id,revision,locale,slug,title,body,created_at) VALUES ('erase-revision','erase-page',1,'en','erase','Title','body','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO media_assets (id,storage_key,original_filename,mime_type,size_bytes,width,height,visibility,status,created_at,updated_at) VALUES ('erase-media','erase/key','x.png','image/png',1,0,0,'private','ready','2026-01-01','2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	log, err := New(filepath.Join(t.TempDir(), "erasure.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, intent := range []Intent{{ID: "erase-page-op", EditionID: "site", TargetID: "erase-page", Kind: "erase.page"}, {ID: "erase-media-op", EditionID: "site", TargetID: "erase-media", Kind: "erase.media"}} {
		if _, err := log.Append(intent); err != nil {
			t.Fatal(err)
		}
	}
	adapter := SQLAdapter{Dialect: database.SQLite}
	checkpoint, err := log.Replay(ctx, 0, func(ctx context.Context, intent Intent) error {
		return database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return adapter.Apply(ctx, tx, intent) })
	})
	if err != nil || checkpoint.Sequence != 2 {
		t.Fatalf("first adapter replay = %#v/%v", checkpoint, err)
	}
	if _, err := log.Replay(ctx, 0, func(ctx context.Context, intent Intent) error {
		return database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return adapter.Apply(ctx, tx, intent) })
	}); err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	var pages, revisions, media int
	for table, target := range map[string]*int{"pages": &pages, "page_revisions": &revisions, "media_assets": &media} {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if pages != 0 || revisions != 0 || media != 0 {
		t.Fatalf("erased rows = pages:%d revisions:%d media:%d", pages, revisions, media)
	}
	if err := adapter.Apply(ctx, db, Intent{EditionID: "site", TargetID: "x", Kind: "erase.unknown"}); err == nil || errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("unsupported kind error = %v", err)
	}
}

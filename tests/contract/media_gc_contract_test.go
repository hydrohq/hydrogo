package contract_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

type gcBlobStore struct{ deleted []string }

func (*gcBlobStore) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (*gcBlobStore) Get(context.Context, string) (io.ReadCloser, error)          { return nil, nil }
func (b *gcBlobStore) Delete(_ context.Context, key string) error {
	b.deleted = append(b.deleted, key)
	return nil
}

func TestSQLiteMediaGarbageCollectionContract(t *testing.T) {
	runMediaGarbageContract(t, database.SQLite, filepath.Join(t.TempDir(), "media-gc.sqlite"))
}

func TestPostgresMediaGarbageCollectionContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMediaGarbageContract(t, database.Postgres, dsn)
}

func runMediaGarbageContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	marker := fmt.Sprintf("gc-contract-%d", time.Now().UTC().UnixNano())
	store := mediastore.Store{Dialect: dialect}
	asset := media.Asset{ID: marker, StorageKey: "sha256/" + marker + ".png", OriginalFilename: marker + ".png", MimeType: "image/png", SizeBytes: 42}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return store.CreatePending(ctx, tx, asset) }); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkReady(ctx, db, asset.ID, media.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	query := `UPDATE media_assets SET created_at = ? WHERE id = ?`
	if dialect == database.Postgres {
		query = `UPDATE media_assets SET created_at = $1 WHERE id = $2`
	}
	if _, err := db.ExecContext(ctx, query, time.Now().UTC().Add(-10*24*time.Hour).Format(time.RFC3339Nano), asset.ID); err != nil {
		t.Fatal(err)
	}
	storeRepo := mediastore.Store{Dialect: dialect}
	blobs := &gcBlobStore{}
	collector := media.GarbageCollector{DB: db, Repository: storeRepo, References: mediastore.MediaReferences{Dialect: dialect, Pages: true}, BlobStore: blobs, Retention: 7 * 24 * time.Hour}
	plan, err := collector.Plan(ctx)
	if err != nil || len(plan) != 1 || plan[0].ID != asset.ID {
		t.Fatalf("initial plan=%+v err=%v", plan, err)
	}
	pageID := "gc-page-" + marker
	pageInsert := `INSERT INTO pages (id,title,locale,slug,body) VALUES (?,?,?,?,?)`
	if dialect == database.Postgres {
		pageInsert = `INSERT INTO pages (id,title,locale,slug,body) VALUES ($1,$2,$3,$4,$5)`
	}
	if _, err := db.ExecContext(ctx, pageInsert, pageID, "GC reference", "en", pageID, `{"media_asset_id":"`+asset.ID+`"}`); err != nil {
		t.Fatal(err)
	}
	plan, err = collector.Plan(ctx)
	if err != nil || len(plan) != 0 {
		t.Fatalf("referenced plan=%+v err=%v", plan, err)
	}
	pageDelete := `DELETE FROM pages WHERE id = ?`
	if dialect == database.Postgres {
		pageDelete = `DELETE FROM pages WHERE id = $1`
	}
	if _, err := db.ExecContext(ctx, pageDelete, pageID); err != nil {
		t.Fatal(err)
	}
	deleted, err := collector.Delete(ctx, []string{asset.ID})
	if err != nil || deleted != 1 || len(blobs.deleted) != 1 {
		t.Fatalf("deleted=%d blobs=%v err=%v", deleted, blobs.deleted, err)
	}
	if _, err := store.Get(ctx, db, asset.ID); err == nil {
		t.Fatal("deleted asset still has a metadata record")
	}
}

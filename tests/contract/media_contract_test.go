package contract_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteMediaAssetLifecycle(t *testing.T) {
	runMediaAssetLifecycle(t, database.SQLite, filepath.Join(t.TempDir(), "media.sqlite"))
}

func TestPostgresMediaAssetLifecycle(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMediaAssetLifecycle(t, database.Postgres, dsn)
}

func runMediaAssetLifecycle(t *testing.T, dialect database.Dialect, dsn string) {
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatal(err)
	}
	resetContractData(t, db, dialect)
	store := mediastore.Store{Dialect: dialect}
	asset := media.Asset{ID: "asset-1", StorageKey: "sha256/asset-1.png", OriginalFilename: "logo.png", MimeType: "image/png", SizeBytes: 42, Width: 10, Height: 10}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		if err := store.CreatePending(ctx, tx, asset); err != nil {
			return err
		}
		return store.MarkReady(ctx, tx, asset.ID, media.VisibilityPublic)
	}); err != nil {
		t.Fatal(err)
	}
	var status, visibility string
	if err := db.QueryRowContext(ctx, "SELECT status, visibility FROM media_assets WHERE id = 'asset-1'").Scan(&status, &visibility); err != nil {
		t.Fatal(err)
	}
	if status != "ready" || visibility != media.VisibilityPublic {
		t.Fatalf("asset state = %s/%s, want ready/public", status, visibility)
	}
}

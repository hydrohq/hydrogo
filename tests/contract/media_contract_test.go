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
		return store.CreatePending(ctx, tx, asset)
	}); err != nil {
		t.Fatal(err)
	}
	folder, err := media.NewFolder("Editorial", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateFolder(ctx, db, folder); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := store.SetMetadata(ctx, db, asset.ID, media.AssetMetadata{Focal: media.FocalPoint{X: .5, Y: .5}, FolderID: folder.ID}); err != nil {
		t.Fatalf("assign pending upload folder: %v", err)
	}
	if err := store.MarkReady(ctx, db, asset.ID, media.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	var status, visibility string
	if err := db.QueryRowContext(ctx, "SELECT status, visibility FROM media_assets WHERE id = 'asset-1'").Scan(&status, &visibility); err != nil {
		t.Fatal(err)
	}
	if status != "ready" || visibility != media.VisibilityPublic {
		t.Fatalf("asset state = %s/%s, want ready/public", status, visibility)
	}
	if err := store.SetMetadata(ctx, db, asset.ID, media.AssetMetadata{Focal: media.FocalPoint{X: .25, Y: .75}, FolderID: folder.ID, Tags: []string{"hero", "header"}}); err != nil {
		t.Fatalf("move/tag asset: %v", err)
	}
	stored, err := store.Get(ctx, db, asset.ID)
	if err != nil || stored.FolderID != folder.ID || stored.FocalX != .25 || len(stored.Tags) != 2 {
		t.Fatalf("metadata = %+v, err=%v", stored, err)
	}
	settings := media.AdminSettings{HotlinkProtection: true, AllowedRefererHosts: []string{"cdn.example.test"}, WatermarkEnabled: true, WatermarkText: "© HydroGo", WatermarkPosition: "bottom-right", WatermarkOpacity: .5, WatermarkMargin: 12, WatermarkLogoID: "logo-asset", WatermarkFolders: []string{folder.ID}}
	if err := store.SaveAdminSettings(ctx, db, settings); err != nil {
		t.Fatalf("save media settings: %v", err)
	}
	gotSettings, found, err := store.LoadAdminSettings(ctx, db)
	if err != nil || !found || !gotSettings.HotlinkProtection || gotSettings.WatermarkText != settings.WatermarkText || gotSettings.WatermarkLogoID != settings.WatermarkLogoID || len(gotSettings.WatermarkFolders) != 1 {
		t.Fatalf("media settings = %+v found=%v err=%v", gotSettings, found, err)
	}
}

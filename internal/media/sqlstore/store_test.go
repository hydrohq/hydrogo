package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestScopedMediaMetadataCannotCrossTenant(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:media-tenant-scope?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	alpha, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := tenant.NewScope("tenant-b", "beta")
	if err != nil {
		t.Fatal(err)
	}
	alphaCtx := tenant.WithScope(ctx, alpha)
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	asset := media.Asset{ID: "asset-a", StorageKey: "asset-a.bin", OriginalFilename: "a.bin", MimeType: "application/octet-stream", SizeBytes: 4}
	if err := database.NewTransactor(db).Within(alphaCtx, func(tx database.Executor) error {
		return store.CreatePending(alphaCtx, tx, asset)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(tenant.WithScope(ctx, beta), db, asset.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant media read = %v", err)
	}
	if err := store.MarkReady(tenant.WithScope(ctx, beta), db, asset.ID, media.VisibilityPublic); !errors.Is(err, media.ErrAssetState) {
		t.Fatalf("cross-tenant media transition = %v", err)
	}
	if err := store.MarkReady(alphaCtx, db, asset.ID, media.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	folder, err := media.NewFolder("Editorial", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CreateFolder(alphaCtx, db, folder); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateFolder(tenant.WithScope(ctx, beta), db, media.Folder{ID: "wrong-tenant-child", Name: "Child", ParentID: folder.ID}); !errors.Is(err, media.ErrInvalidFolder) {
		t.Fatalf("cross-tenant folder parent = %v", err)
	}
	if err := store.SetMetadata(alphaCtx, db, asset.ID, media.AssetMetadata{Focal: media.FocalPoint{X: .25, Y: .75}, FolderID: folder.ID, Tags: []string{"hero", "header"}}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || got.ID != asset.ID || got.Visibility != media.VisibilityPublic || got.FocalX != .25 || got.FolderID != folder.ID || len(got.Tags) != 2 {
		t.Fatalf("scoped media read = %#v err=%v", got, err)
	}
	folders, err := store.ListFolders(alphaCtx, db)
	if err != nil || len(folders) != 1 || folders[0].ID != folder.ID {
		t.Fatalf("folder list = %#v err=%v", folders, err)
	}
}

func TestMediaAdminSettingsAreTenantScoped(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:media-admin-settings?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	alpha, _ := tenant.NewScope("settings-a", "alpha")
	beta, _ := tenant.NewScope("settings-b", "beta")
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	want := media.AdminSettings{HotlinkProtection: true, AllowedRefererHosts: []string{"assets.example"}, WatermarkEnabled: true, WatermarkText: "© Alpha", WatermarkOpacity: .45, WatermarkLogoID: "logo-asset", WatermarkFolders: []string{"folder-a"}}
	if err := store.SaveAdminSettings(tenant.WithScope(ctx, alpha), db, want); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.LoadAdminSettings(tenant.WithScope(ctx, alpha), db)
	if err != nil || !found || got.WatermarkText != want.WatermarkText || !got.HotlinkProtection || got.WatermarkLogoID != "logo-asset" || len(got.WatermarkFolders) != 1 || got.WatermarkFolders[0] != "folder-a" {
		t.Fatalf("alpha settings = %+v, found=%v err=%v", got, found, err)
	}
	if _, found, err := store.LoadAdminSettings(tenant.WithScope(ctx, beta), db); err != nil || found {
		t.Fatalf("beta settings found=%v err=%v", found, err)
	}
}

func TestPendingUploadCanBeAssignedToWatermarkedFolderBeforeReady(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:media-pending-folder?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	folder, _ := media.NewFolder("Photos", "")
	if err := store.CreateFolder(ctx, db, folder); err != nil {
		t.Fatal(err)
	}
	asset := media.Asset{ID: "pending-image", StorageKey: "sha256/master.png", OriginalFilename: "master.png", MimeType: "image/png", SizeBytes: 64}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return store.CreatePending(ctx, tx, asset) }); err != nil {
		t.Fatal(err)
	}
	if err := store.SetMetadata(ctx, db, asset.ID, media.AssetMetadata{Focal: media.FocalPoint{X: .5, Y: .5}, FolderID: folder.ID}); err != nil {
		t.Fatalf("set pending upload folder: %v", err)
	}
	if err := store.MarkReady(ctx, db, asset.ID, media.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, db, asset.ID)
	if err != nil || got.FolderID != folder.ID || got.StorageKey != asset.StorageKey {
		t.Fatalf("ready upload = %+v err=%v", got, err)
	}
}

func TestMediaStoreFailClosedWithoutTenantScope(t *testing.T) {
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.CreatePending(context.Background(), nil, media.Asset{ID: "id"}); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing media scope = %v", err)
	}
}

func TestScopedMediaVariantCannotCrossTenant(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:media-variant-tenant-scope?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	beta, _ := tenant.NewScope("tenant-b", "beta")
	alphaCtx := tenant.WithScope(ctx, alpha)
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	asset := media.Asset{ID: "asset-v", StorageKey: "asset-v.bin", OriginalFilename: "v.bin", MimeType: "image/png", SizeBytes: 4}
	if err := database.NewTransactor(db).Within(alphaCtx, func(tx database.Executor) error { return store.CreatePending(alphaCtx, tx, asset) }); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateProcessing(alphaCtx, db, media.Variant{ID: "variant-a", MediaID: asset.ID, Kind: "thumb", MimeType: "image/png"}); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkResult(alphaCtx, db, "variant-a", "ready", 4, "thumb.png"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetReady(tenant.WithScope(ctx, beta), db, asset.ID, "thumb"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-tenant variant read = %v", err)
	}
	if got, err := store.GetReady(alphaCtx, db, asset.ID, "thumb"); err != nil || got.ID != "variant-a" || got.StorageKey != "tenants/tenant-a/thumb.png" {
		t.Fatalf("scoped variant read = %#v err=%v", got, err)
	}
}

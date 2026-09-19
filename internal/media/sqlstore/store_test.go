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
	got, err := store.Get(alphaCtx, db, asset.ID)
	if err != nil || got.ID != asset.ID || got.Visibility != media.VisibilityPublic {
		t.Fatalf("scoped media read = %#v err=%v", got, err)
	}
}

func TestMediaStoreFailClosedWithoutTenantScope(t *testing.T) {
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	if err := store.CreatePending(context.Background(), nil, media.Asset{ID: "id"}); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing media scope = %v", err)
	}
}

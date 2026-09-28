package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestGarbageRepositoryClaimsAndListsDerivedObjects(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "media-garbage.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE media_assets(id TEXT PRIMARY KEY,storage_key TEXT,original_filename TEXT,size_bytes INTEGER,created_at TEXT,updated_at TEXT,status TEXT,tenant_id TEXT)`,
		`CREATE TABLE media_variants(id TEXT,media_id TEXT,storage_key TEXT,tenant_id TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	created := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339Nano)
	_, err = db.Exec(`INSERT INTO media_assets VALUES('old','master.png','old.png',200,?,?,'ready','')`, created, created)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO media_variants VALUES('v','old','thumb.png','')`)
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Dialect: database.SQLite}
	items, err := store.ListGarbageCandidates(ctx, db, time.Now().UTC().Add(-7*24*time.Hour), 20)
	if err != nil || len(items) != 1 || items[0].ID != "old" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if claimed, err := store.ClaimGarbageAsset(ctx, db, "old", items[0].CreatedAt); err != nil || !claimed {
		t.Fatalf("claimed=%v err=%v", claimed, err)
	}
	keys, err := store.GarbageObjectKeys(ctx, db, "old")
	if err != nil || len(keys) != 2 {
		t.Fatalf("keys=%v err=%v", keys, err)
	}
	if err := store.DeleteGarbageAsset(ctx, db, "old"); err != nil {
		t.Fatal(err)
	}
}

func TestMediaReferenceScannerIncludesAllConfiguredSourcesAndTenantScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "media-refs.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE pages(body TEXT,tenant_id TEXT)`,
		`CREATE TABLE page_revisions(body TEXT,blocks TEXT,tenant_id TEXT)`,
		`CREATE TABLE news_items(body TEXT,cover_media_id TEXT,tenant_id TEXT)`,
		`CREATE TABLE site_settings(value TEXT,tenant_id TEXT)`,
		`CREATE TABLE form_inbox(payload TEXT,tenant_id TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = db.Exec(`INSERT INTO pages VALUES(?,?)`, `{"media_asset_id":"pageasset0001"}`, "alpha")
	_, _ = db.Exec(`INSERT INTO page_revisions VALUES(?,?,?)`, "", `[{"before_image_asset_id":"blockasset0001"}]`, "alpha")
	_, _ = db.Exec(`INSERT INTO news_items VALUES(?,?,?)`, "", "newsasset0001", "beta")
	_, _ = db.Exec(`INSERT INTO site_settings VALUES(?,?)`, `{"logo":"/media/siteasset0001"}`, "alpha")
	_, _ = db.Exec(`INSERT INTO form_inbox VALUES(?,?)`, `{"media_asset_id":"formasset0001"}`, "alpha")
	scope, _ := tenant.NewScope("alpha", "alpha")
	scanner := MediaReferences{Dialect: database.SQLite, RequireTenantScope: true, Pages: true, News: true, Site: true, Forms: true}
	ids, err := scanner.ReferencedMediaIDs(tenant.WithScope(ctx, scope), db)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pageasset0001", "blockasset0001", "siteasset0001", "formasset0001"} {
		if _, ok := ids[id]; !ok {
			t.Errorf("reference %q is missing: %v", id, ids)
		}
	}
	if _, ok := ids["newsasset0001"]; ok {
		t.Fatalf("cross-tenant reference leaked: %v", ids)
	}
}

package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestAPITokenStoreLifecycle(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "tokens.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE api_tokens (id TEXT PRIMARY KEY, user_id TEXT NOT NULL, tenant_id TEXT NOT NULL DEFAULT '', name TEXT NOT NULL, kind TEXT NOT NULL, token_hash TEXT NOT NULL UNIQUE, token_prefix TEXT NOT NULL, scopes TEXT NOT NULL DEFAULT '', expires_at TEXT, last_used_at TEXT, created_at TEXT NOT NULL, revoked_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	store := NewAPITokenStore(db, database.SQLite)
	ctx := context.Background()
	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)

	personal := &auth.APIToken{ID: "t1", UserID: "u1", Name: "cli", Kind: auth.APITokenPersonal, Prefix: "abcdefgh", Scopes: []string{auth.ScopeAdmin}, TokenHash: "hash-1", ExpiresAt: &expires}
	key := &auth.APIToken{ID: "k1", UserID: "u1", Name: "site", Kind: auth.APITokenKey, Prefix: "ijklmnop", Scopes: []string{auth.ScopeRead}, TokenHash: "hash-2"}
	other := &auth.APIToken{ID: "t2", UserID: "u2", Name: "other", Kind: auth.APITokenPersonal, Prefix: "qrstuvwx", Scopes: []string{auth.ScopeAdmin}, TokenHash: "hash-3"}
	for _, token := range []*auth.APIToken{personal, key, other} {
		if err := store.Create(ctx, token); err != nil {
			t.Fatal(err)
		}
	}

	found, err := store.FindByHash(ctx, "hash-1")
	if err != nil || found.ID != "t1" || found.Kind != auth.APITokenPersonal || len(found.Scopes) != 1 || found.Scopes[0] != auth.ScopeAdmin || found.ExpiresAt == nil || !found.ExpiresAt.Equal(expires) || found.CreatedAt.IsZero() {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	if _, err := store.FindByHash(ctx, "missing"); !errors.Is(err, auth.ErrAPITokenNotFound) {
		t.Fatalf("missing hash err = %v", err)
	}

	mine, err := store.ListForUser(ctx, "u1")
	if err != nil || len(mine) != 1 || mine[0].ID != "t1" {
		t.Fatalf("ListForUser=%+v err=%v", mine, err)
	}
	keys, err := store.ListAPIKeys(ctx)
	if err != nil || len(keys) != 1 || keys[0].ID != "k1" {
		t.Fatalf("ListAPIKeys=%+v err=%v", keys, err)
	}

	used := time.Now().UTC().Truncate(time.Second)
	if err := store.TouchLastUsed(ctx, "t1", used); err != nil {
		t.Fatal(err)
	}
	if found, _ = store.FindByHash(ctx, "hash-1"); found.LastUsedAt == nil || !found.LastUsedAt.Equal(used) {
		t.Fatalf("last_used_at = %v", found.LastUsedAt)
	}

	if err := store.Revoke(ctx, "t1", "u2"); !errors.Is(err, auth.ErrAPITokenNotFound) {
		t.Fatalf("cross-user revoke err = %v", err)
	}
	if err := store.Revoke(ctx, "t1", "u1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(ctx, "t1", "u1"); !errors.Is(err, auth.ErrAPITokenNotFound) {
		t.Fatalf("double revoke err = %v", err)
	}
	if found, _ = store.FindByHash(ctx, "hash-1"); found.RevokedAt == nil {
		t.Fatal("revoked_at not set")
	}
	if mine, _ = store.ListForUser(ctx, "u1"); len(mine) != 0 {
		t.Fatalf("revoked token still listed: %+v", mine)
	}
	if err := store.Revoke(ctx, "k1", ""); err != nil {
		t.Fatalf("owner-less revoke err = %v", err)
	}
}

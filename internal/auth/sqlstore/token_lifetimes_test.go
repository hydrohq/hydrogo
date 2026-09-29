package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestTokenLifetimePolicyPersistsValidatedChoices(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "token-lifetimes.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE auth_token_lifetimes(singleton_id INTEGER PRIMARY KEY,password_reset_minutes INTEGER NOT NULL,invitation_hours INTEGER NOT NULL,updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	store := TokenLifetimeStore{DB: db, Dialect: database.SQLite}
	if _, ok, err := store.Read(ctx); err != nil || ok {
		t.Fatalf("initial policy: ok=%v err=%v", ok, err)
	}
	want := auth.TokenLifetimes{PasswordReset: 24 * time.Hour, Invitation: 7 * 24 * time.Hour}
	if err := store.Write(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Read(ctx)
	if err != nil || !ok || got != want {
		t.Fatalf("policy=%+v ok=%v err=%v", got, ok, err)
	}
	if err := store.Write(ctx, auth.TokenLifetimes{PasswordReset: 2 * time.Hour, Invitation: 24 * time.Hour}); err == nil {
		t.Fatal("unsupported lifetime accepted")
	}
}

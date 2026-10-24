package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/pquerna/otp/totp"
)

// A TOTP code observed over a shoulder or through a phishing page must not
// work a second time inside its 30-second validity window.
func TestATOTPCodeCannotBeReplayed(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "totp.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := NewUserStore(db, database.SQLite)
	if err := store.Create(ctx, &auth.User{ID: "u-mfa", Email: "mfa@example.test", PasswordHash: "hash", Role: auth.RoleAdmin, Active: true}); err != nil {
		t.Fatal(err)
	}
	mfa := auth.MFAService{Store: store, MasterKey: []byte("0123456789abcdef0123456789abcdef")}
	enrollment, err := mfa.Enroll(ctx, db, "u-mfa", "mfa@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := mfa.Enable(ctx, db, "u-mfa"); err != nil {
		t.Fatal(err)
	}
	code, err := totp.GenerateCode(enrollment.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := mfa.Verify(ctx, db, "u-mfa", code); err != nil {
		t.Fatalf("first use of a fresh code: %v", err)
	}
	if err := mfa.Verify(ctx, db, "u-mfa", code); !errors.Is(err, auth.ErrMFAInvalidCode) {
		t.Fatalf("replaying the same code: %v, want ErrMFAInvalidCode", err)
	}
	stale, err := totp.GenerateCode(enrollment.Secret, time.Now().Add(-30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if stale != code {
		if err := mfa.Verify(ctx, db, "u-mfa", stale); !errors.Is(err, auth.ErrMFAInvalidCode) {
			t.Fatalf("an older step after a newer one was used: %v, want ErrMFAInvalidCode", err)
		}
	}
}

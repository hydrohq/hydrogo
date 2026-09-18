package members_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

func TestMemberMobileTokenRotationRevokesReusedFamily(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "member-mobile.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `INSERT INTO members (id,email,password_hash,active,created_at,updated_at) VALUES (?,?,?,?,?,?)`, "member-mobile", "reader@example.test", "unused", 1, now, now); err != nil {
		t.Fatal(err)
	}
	service := members.MobileAuthService{DB: db, Dialect: database.SQLite}
	pair, err := service.Issue(ctx, "member-mobile", "reader-ios", "Reader phone", "iOS")
	if err != nil {
		t.Fatal(err)
	}
	if memberID, _, err := service.ValidateAccess(ctx, pair.AccessToken); err != nil || memberID != "member-mobile" {
		t.Fatalf("validate=%q/%v", memberID, err)
	}
	rotated, err := service.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(ctx, pair.RefreshToken); err == nil {
		t.Fatal("reused refresh token unexpectedly succeeded")
	}
	if _, _, err := service.ValidateAccess(ctx, rotated.AccessToken); err == nil {
		t.Fatal("family reuse left rotated access token valid")
	}
}

package members_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
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
	activePair, err := service.Issue(ctx, "member-mobile", "reader-android", "Reader tablet", "android")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE members SET active = 0 WHERE id = ?`, "member-mobile"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.ValidateAccess(ctx, activePair.AccessToken); err == nil {
		t.Fatal("disabled member retained a valid access token")
	}
}

func TestMemberMobileTenantScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "member-mobile-tenant.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `INSERT INTO members (id,email,password_hash,active,created_at,updated_at,tenant_id) VALUES (?,?,?,?,?,?,?)`, "member-a", "reader-a@example.test", "unused", 1, now, now, "tenant-a"); err != nil {
		t.Fatal(err)
	}
	service := members.MobileAuthService{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	ctxA := tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-a", Slug: "a"})
	ctxB := tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-b", Slug: "b"})
	pair, err := service.Issue(ctxA, "member-a", "reader-ios", "Reader phone", "iOS")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.ValidateAccess(ctxB, pair.AccessToken); !errors.Is(err, members.ErrMobileTokenInvalid) {
		t.Fatalf("cross-tenant access = %v", err)
	}
	if _, _, err := service.ValidateAccess(context.Background(), pair.AccessToken); !errors.Is(err, members.ErrMobileTokenInvalid) {
		t.Fatalf("missing scope access = %v", err)
	}
	if devices, err := service.ListDevices(ctxB, "member-a", ""); err != nil || len(devices) != 0 {
		t.Fatalf("cross-tenant devices = %#v, %v", devices, err)
	}
}

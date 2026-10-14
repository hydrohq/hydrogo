package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
)

func setupMemberOAuthDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "member-oauth.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	schema := `
	CREATE TABLE members (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1,
		email_verified_at TEXT,
		last_login_at TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE member_oauth_identities (
		provider TEXT NOT NULL,
		subject TEXT NOT NULL,
		member_id TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		PRIMARY KEY (provider, subject, tenant_id)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMemberOAuthResolveRejectsUnverifiedOrIncompleteIdentity(t *testing.T) {
	db := setupMemberOAuthDB(t)
	store := sqlstore.OAuthStore{DB: db, Dialect: database.SQLite}
	ctx := context.Background()
	if _, err := store.ResolveOAuth(ctx, auth.OAuthIdentity{Provider: "google", Subject: "sub", Email: "a@example.test", EmailVerified: false}); !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("unverified email error = %v", err)
	}
	if _, err := store.ResolveOAuth(ctx, auth.OAuthIdentity{Provider: "google", Subject: "", Email: "a@example.test", EmailVerified: true}); !errors.Is(err, members.ErrInvalidCredentials) {
		t.Fatalf("missing subject error = %v", err)
	}
}

func TestMemberOAuthResolveProvisionsNewMemberOnFirstSignIn(t *testing.T) {
	db := setupMemberOAuthDB(t)
	store := sqlstore.OAuthStore{DB: db, Dialect: database.SQLite}
	ctx := context.Background()
	identity := auth.OAuthIdentity{Provider: "google", Subject: "google-sub-1", Email: "new-member@example.test", EmailVerified: true}

	member, err := store.ResolveOAuth(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if !member.Active || member.PasswordHash != "" || member.EmailVerifiedAt == nil {
		t.Fatalf("provisioned member = %+v", member)
	}
	if member.Email != "new-member@example.test" {
		t.Fatalf("provisioned member email = %q", member.Email)
	}

	// Second sign-in with the same identity resolves to the same member
	// without provisioning a duplicate.
	again, err := store.ResolveOAuth(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != member.ID {
		t.Fatalf("second resolve created a different member: %+v vs %+v", again, member)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM members").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("member count = %d, want 1", count)
	}
}

func TestMemberOAuthResolveLinksExistingMemberByEmail(t *testing.T) {
	db := setupMemberOAuthDB(t)
	memberStore := sqlstore.Store{DB: db, Dialect: database.SQLite}
	ctx := context.Background()
	existing := &members.Member{ID: "existing-member", Email: "existing@example.test", PasswordHash: "hash", Active: true}
	existing.CreatedAt, existing.UpdatedAt = existing.CreatedAt.UTC(), existing.UpdatedAt.UTC()
	if err := memberStore.Create(ctx, existing); err != nil {
		t.Fatal(err)
	}

	store := sqlstore.OAuthStore{DB: db, Dialect: database.SQLite}
	identity := auth.OAuthIdentity{Provider: "github", Subject: "github-sub-1", Email: "existing@example.test", EmailVerified: true}
	member, err := store.ResolveOAuth(ctx, identity)
	if err != nil {
		t.Fatal(err)
	}
	if member.ID != "existing-member" {
		t.Fatalf("linked member = %+v, want existing-member", member)
	}
	if member.EmailVerifiedAt == nil {
		t.Fatal("linking a verified social identity should mark the member's email verified")
	}
	var linkCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM member_oauth_identities WHERE member_id = ?", "existing-member").Scan(&linkCount); err != nil {
		t.Fatal(err)
	}
	if linkCount != 1 {
		t.Fatalf("linked identity count = %d, want 1", linkCount)
	}
}

func TestMemberOAuthResolveRejectsInactiveMember(t *testing.T) {
	db := setupMemberOAuthDB(t)
	memberStore := sqlstore.Store{DB: db, Dialect: database.SQLite}
	ctx := context.Background()
	inactive := &members.Member{ID: "inactive-member", Email: "inactive@example.test", PasswordHash: "hash", Active: false}
	if err := memberStore.Create(ctx, inactive); err != nil {
		t.Fatal(err)
	}
	store := sqlstore.OAuthStore{DB: db, Dialect: database.SQLite}
	identity := auth.OAuthIdentity{Provider: "github", Subject: "github-sub-2", Email: "inactive@example.test", EmailVerified: true}
	if _, err := store.ResolveOAuth(ctx, identity); !errors.Is(err, members.ErrMemberInactive) {
		t.Fatalf("inactive member error = %v", err)
	}
}

func TestMemberOAuthResolveTenantScoped(t *testing.T) {
	// Member email uniqueness is global, not per-tenant (see the members
	// table's UNIQUE(email), unchanged by the tenant-scope migration), so
	// this uses distinct emails per tenant to isolate what the OAuth
	// resolver itself scopes: a provider identity linked in one tenant must
	// not resolve, or leak its member, into another tenant.
	db := setupMemberOAuthDB(t)
	store := sqlstore.OAuthStore{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	ctxA := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-a", Slug: "a"})
	ctxB := tenant.WithScope(context.Background(), tenant.Scope{TenantID: "tenant-b", Slug: "b"})

	memberA, err := store.ResolveOAuth(ctxA, auth.OAuthIdentity{Provider: "google", Subject: "google-sub-3", Email: "tenant-a-member@example.test", EmailVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	memberB, err := store.ResolveOAuth(ctxB, auth.OAuthIdentity{Provider: "google", Subject: "google-sub-3", Email: "tenant-b-member@example.test", EmailVerified: true})
	if err != nil {
		t.Fatal(err)
	}
	if memberA.ID == memberB.ID {
		t.Fatal("cross-tenant OAuth resolve returned the same member")
	}
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM members").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("member count across tenants = %d, want 2", total)
	}
}

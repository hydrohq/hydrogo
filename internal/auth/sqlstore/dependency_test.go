package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestAuthStoresRejectMissingDatabaseAndTransactions(t *testing.T) {
	ctx := context.Background()
	if _, err := NewUserStore(nil, database.SQLite).GetByID(ctx, "user"); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing user database error = %v", err)
	}
	if _, err := NewSessionStore(nil, database.SQLite).GetByTokenHash(ctx, "token"); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing session database error = %v", err)
	}

	db := &sql.DB{}
	store := NewUserStore(db, database.SQLite)
	if err := store.CreateUserTx(ctx, nil, &auth.User{}); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing transaction error = %v", err)
	}
	if err := store.PutFactor(ctx, nil, "user", nil, false); !errors.Is(err, database.ErrDatabaseRequired) {
		t.Fatalf("missing MFA transaction error = %v", err)
	}
}

func TestAuthIdentityTenantScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "auth-tenant.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY,email TEXT NOT NULL,password_hash TEXT NOT NULL,role TEXT NOT NULL,active INTEGER NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL,tenant_id TEXT NOT NULL)`,
		`CREATE TABLE sessions (id TEXT PRIMARY KEY,user_id TEXT NOT NULL,token_hash TEXT NOT NULL,expires_at TEXT NOT NULL,created_at TEXT NOT NULL,last_seen_at TEXT NOT NULL,user_agent TEXT NOT NULL,ip_address TEXT NOT NULL,tenant_id TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	users := NewUserStore(db, database.SQLite)
	users.RequireTenantScope = true
	sessions := NewSessionStore(db, database.SQLite)
	sessions.RequireTenantScope = true
	ctxA := tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-a", Slug: "a"})
	ctxB := tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-b", Slug: "b"})
	if err := users.Create(ctxA, &auth.User{ID: "u-a", Email: "admin@example.test", PasswordHash: "hash", Role: auth.RoleAdmin, Active: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := users.GetByEmail(ctxB, "admin@example.test"); !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("cross-tenant user = %v", err)
	}
	if _, err := users.GetByID(ctx, "u-a"); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing user scope = %v", err)
	}
	sess := &auth.Session{ID: "s-a", UserID: "u-a", TokenHash: "hash-a", ExpiresAt: time.Now().UTC().Add(time.Hour)}
	if err := sessions.Create(ctxA, sess); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.GetByTokenHash(ctxB, "hash-a"); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("cross-tenant session = %v", err)
	}
	if err := sessions.Create(ctxA, &auth.Session{ID: "s-a-other", UserID: "u-a", TokenHash: "hash-b", ExpiresAt: time.Now().UTC().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.DeleteOtherByUserID(ctxA, "u-a", "hash-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.GetByTokenHash(ctxA, "hash-a"); err != nil {
		t.Fatalf("current session was revoked: %v", err)
	}
	if _, err := sessions.GetByTokenHash(ctxA, "hash-b"); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("other session still active: %v", err)
	}
}

func TestAuthSecurityTenantScope(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "auth-security-tenant.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE auth_invites (id TEXT PRIMARY KEY,email TEXT NOT NULL,role TEXT NOT NULL,token_hash TEXT NOT NULL,expires_at TEXT NOT NULL,consumed_at TEXT,created_at TEXT NOT NULL,tenant_id TEXT NOT NULL)`,
		`CREATE TABLE auth_recovery_tokens (id TEXT PRIMARY KEY,user_id TEXT NOT NULL,purpose TEXT NOT NULL,token_hash TEXT NOT NULL,expires_at TEXT NOT NULL,consumed_at TEXT,created_at TEXT NOT NULL,tenant_id TEXT NOT NULL)`,
		`CREATE TABLE user_mfa (user_id TEXT PRIMARY KEY,secret_ciphertext BLOB NOT NULL,enabled INTEGER NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL,tenant_id TEXT NOT NULL)`,
		`CREATE TABLE mfa_recovery_codes (id TEXT PRIMARY KEY,user_id TEXT NOT NULL,code_hash TEXT NOT NULL,used_at TEXT,tenant_id TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := NewUserStore(db, database.SQLite)
	store.RequireTenantScope = true
	ctxA := tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-a", Slug: "a"})
	ctxB := tenant.WithScope(ctx, tenant.Scope{TenantID: "tenant-b", Slug: "b"})
	now := time.Now().UTC()
	if err := store.CreateInvite(ctxA, db, "invite-a", "a@example.test", auth.RoleEditor, "invite-hash-a", now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ConsumeInvite(ctxB, db, "invite-hash-a", now); !errors.Is(err, auth.ErrInviteInvalid) {
		t.Fatalf("cross-tenant invite = %v", err)
	}
	if err := store.CreateRecoveryToken(ctxA, db, "recovery-a", "user-a", auth.PasswordResetPurpose, "recovery-hash-a", now.Add(time.Hour), now); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeRecoveryToken(ctxB, db, auth.PasswordResetPurpose, "recovery-hash-a", now); !errors.Is(err, auth.ErrRecoveryTokenInvalid) {
		t.Fatalf("cross-tenant recovery = %v", err)
	}
	if err := store.PutFactor(ctxA, db, "user-a", []byte("cipher"), true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.GetFactor(ctxB, db, "user-a"); !errors.Is(err, auth.ErrMFAUnavailable) {
		t.Fatalf("cross-tenant MFA = %v", err)
	}
}

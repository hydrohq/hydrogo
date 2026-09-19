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
}

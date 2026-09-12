package sqlstore_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
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
		updated_at TEXT NOT NULL
	);
	CREATE TABLE member_sessions (
		id TEXT PRIMARY KEY,
		member_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		ip_address TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return db
}

func TestSQLiteMemberAndSessionStore(t *testing.T) {
	db := setupTestDB(t)
	mStore := sqlstore.Store{DB: db, Dialect: database.SQLite}
	sStore := sqlstore.SessionStore{DB: db, Dialect: database.SQLite}

	ctx := context.Background()
	now := time.Now().UTC()

	// 1. Create Member
	m := &members.Member{
		ID:           "m-1",
		Email:        "test@example.com",
		PasswordHash: "hash123",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := mStore.Create(ctx, m); err != nil {
		t.Fatalf("mStore.Create failed: %v", err)
	}

	// 2. Get Member By ID and Email
	fetched, err := mStore.GetByID(ctx, "m-1")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Email != "test@example.com" || !fetched.Active {
		t.Fatalf("GetByID unexpected member: %+v", fetched)
	}

	byEmail, err := mStore.GetByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if byEmail.ID != "m-1" {
		t.Fatalf("GetByEmail unexpected id: %s", byEmail.ID)
	}

	// 3. Create Session
	expiresAt := now.Add(24 * time.Hour)
	session := &members.Session{
		ID:         "sess-1",
		MemberID:   "m-1",
		TokenHash:  "token-hash-123",
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
		LastSeenAt: now,
		UserAgent:  "test-agent",
		IPAddress:  "127.0.0.1",
	}
	if err := sStore.Create(ctx, session); err != nil {
		t.Fatalf("sStore.Create failed: %v", err)
	}

	// 4. Get Session By Token Hash
	fetchedSess, err := sStore.GetByTokenHash(ctx, "token-hash-123")
	if err != nil {
		t.Fatalf("GetByTokenHash failed: %v", err)
	}
	if fetchedSess.MemberID != "m-1" || fetchedSess.ID != "sess-1" {
		t.Fatalf("GetByTokenHash unexpected session: %+v", fetchedSess)
	}
	if fetchedSess.ExpiresAt.IsZero() || fetchedSess.CreatedAt.IsZero() {
		t.Fatalf("GetByTokenHash zero timestamp: %+v", fetchedSess)
	}

	// 5. Service.ValidateSession
	svc := members.Service{Members: mStore, Sessions: sStore}
	_, _, err = svc.ValidateSession(ctx, "raw-token") // wrong raw token
	if err == nil {
		t.Fatal("expected error for wrong raw token")
	}

	// Now create a real session via Service
	rawToken, createdSess, err := svc.CreateSession(ctx, "m-1", "agent", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("svc.CreateSession failed: %v", err)
	}
	valSess, valMem, err := svc.ValidateSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("svc.ValidateSession failed: %v", err)
	}
	if valSess.ID != createdSess.ID || valMem.ID != "m-1" {
		t.Fatalf("svc.ValidateSession mismatch: sess=%+v mem=%+v", valSess, valMem)
	}
}

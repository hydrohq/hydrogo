package sqlstore_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
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
		updated_at TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE member_sessions (
		id TEXT PRIMARY KEY,
		member_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE member_email_verification_tokens (
		id TEXT PRIMARY KEY,
		member_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL,
		consumed_at TEXT,
		created_at TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT ''
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return db
}

func TestMemberEmailVerificationTokenIsExpiringAndOneUse(t *testing.T) {
	db := setupTestDB(t)
	store := sqlstore.Store{DB: db, Dialect: database.SQLite}
	ctx := context.Background()
	now := time.Now().UTC()
	if err := store.Create(ctx, &members.Member{ID: "verify-member", Email: "verify@example.test", PasswordHash: "hash", Active: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	service := members.EmailVerificationService{Tokens: store}
	tx := database.NewTenantTransactor(db, database.SQLite, false)
	var token string
	if err := tx.Within(ctx, func(exec database.Executor) error {
		var err error
		token, err = service.Issue(ctx, exec, &members.Member{ID: "verify-member", Email: "verify@example.test", Active: true}, time.Hour)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(token) != 64 {
		t.Fatalf("issued token length = %d", len(token))
	}
	if err := tx.Within(ctx, func(exec database.Executor) error { return service.Verify(ctx, exec, token) }); err != nil {
		t.Fatal(err)
	}
	member, err := store.GetByID(ctx, "verify-member")
	if err != nil || member.EmailVerifiedAt == nil {
		t.Fatalf("verification state = %#v, err=%v", member, err)
	}
	if err := tx.Within(ctx, func(exec database.Executor) error { return service.Verify(ctx, exec, token) }); !errors.Is(err, members.ErrEmailVerificationInvalid) {
		t.Fatalf("replay verification error = %v", err)
	}
}

func TestMemberEmailVerificationTokenCannotCrossTenant(t *testing.T) {
	db := setupTestDB(t)
	store := sqlstore.Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	alpha, _ := tenant.NewScope("alpha", "alpha")
	beta, _ := tenant.NewScope("beta", "beta")
	ctxA, ctxB := tenant.WithScope(context.Background(), alpha), tenant.WithScope(context.Background(), beta)
	now := time.Now().UTC()
	if err := store.Create(ctxA, &members.Member{ID: "tenant-member", Email: "tenant@example.test", PasswordHash: "hash", Active: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	service := members.EmailVerificationService{Tokens: store}
	tx := database.NewTenantTransactor(db, database.SQLite, true)
	var token string
	if err := tx.Within(ctxA, func(exec database.Executor) error {
		var err error
		token, err = service.Issue(ctxA, exec, &members.Member{ID: "tenant-member", Email: "tenant@example.test", Active: true}, time.Hour)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Within(ctxB, func(exec database.Executor) error { return service.Verify(ctxB, exec, token) }); !errors.Is(err, members.ErrEmailVerificationInvalid) {
		t.Fatalf("cross-tenant verify error = %v", err)
	}
	if err := tx.Within(ctxA, func(exec database.Executor) error { return service.Verify(ctxA, exec, token) }); err != nil {
		t.Fatalf("owner-tenant verify error = %v", err)
	}
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

func TestTenantScopedMemberStoreFailsClosed(t *testing.T) {
	db := setupTestDB(t)
	store := sqlstore.Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	alpha, _ := tenant.NewScope("tenant-a", "alpha")
	now := time.Now().UTC()
	member := &members.Member{ID: "scoped-a", Email: "a@example.com", PasswordHash: "hash", Active: true, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(tenant.WithScope(context.Background(), alpha), member); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetByID(context.Background(), member.ID); !errors.Is(err, tenant.ErrMissingScope) {
		t.Fatalf("missing member scope error = %v", err)
	}
	if _, err := store.GetByID(tenant.WithScope(context.Background(), alpha), member.ID); err != nil {
		t.Fatalf("scoped member lookup = %v", err)
	}
}

func TestSQLiteMemberCreateMapsUniqueEmailViolation(t *testing.T) {
	db := setupTestDB(t)
	store := sqlstore.Store{DB: db, Dialect: database.SQLite}
	now := time.Now().UTC()
	member := &members.Member{ID: "m-1", Email: "duplicate@example.com", PasswordHash: "hash", Active: true, CreatedAt: now, UpdatedAt: now}
	if err := store.Create(context.Background(), member); err != nil {
		t.Fatal(err)
	}
	duplicate := *member
	duplicate.ID = "m-2"
	if err := store.Create(context.Background(), &duplicate); !errors.Is(err, members.ErrMemberExists) {
		t.Fatalf("expected ErrMemberExists, got %v", err)
	}
}

func TestSessionGetByIDFindsOldestBeyondListLimitAndScopesTenant(t *testing.T) {
	db := setupTestDB(t)
	store := sqlstore.SessionStore{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	alpha, _ := tenant.NewScope("alpha", "alpha")
	beta, _ := tenant.NewScope("beta", "beta")
	ctxA := tenant.WithScope(context.Background(), alpha)
	ctxB := tenant.WithScope(context.Background(), beta)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	oldest := "session-000"
	for i := 0; i < 150; i++ {
		id := fmt.Sprintf("session-%03d", i)
		session := &members.Session{ID: id, MemberID: "member-a", TokenHash: "hash-" + id, ExpiresAt: base.Add(24 * time.Hour), CreatedAt: base.Add(time.Duration(i) * time.Minute), LastSeenAt: base.Add(time.Duration(i) * time.Minute), UserAgent: "test", IPAddress: "127.0.0.1"}
		if err := store.Create(ctxA, session); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.GetByID(ctxA, "member-a", oldest)
	if err != nil || got == nil || got.ID != oldest {
		t.Fatalf("oldest lookup = %#v, err=%v", got, err)
	}
	service := members.Service{Sessions: store}
	if err := service.RevokeSessionByID(ctxA, "member-a", oldest); err != nil {
		t.Fatalf("oldest revoke = %v", err)
	}
	if _, err := store.GetByID(ctxA, "member-a", oldest); !errors.Is(err, members.ErrMemberNotFound) {
		t.Fatalf("deleted lookup err=%v", err)
	}
	if _, err := store.GetByID(ctxB, "member-a", "session-001"); !errors.Is(err, members.ErrMemberNotFound) {
		t.Fatalf("cross-tenant lookup err=%v", err)
	}
}

func TestSessionStoreDeleteExpiredRemovesOnlyPastSessions(t *testing.T) {
	db := setupTestDB(t)
	mStore := sqlstore.Store{DB: db, Dialect: database.SQLite}
	sStore := sqlstore.SessionStore{DB: db, Dialect: database.SQLite}
	ctx := context.Background()
	now := time.Now().UTC()
	if err := mStore.Create(ctx, &members.Member{ID: "m-exp", Email: "exp@example.test", PasswordHash: "hash", Active: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	expired := &members.Session{ID: "s-expired", MemberID: "m-exp", TokenHash: "hash-expired", ExpiresAt: now.Add(-time.Hour), CreatedAt: now.Add(-2 * time.Hour), LastSeenAt: now.Add(-2 * time.Hour)}
	live := &members.Session{ID: "s-live", MemberID: "m-exp", TokenHash: "hash-live", ExpiresAt: now.Add(time.Hour), CreatedAt: now, LastSeenAt: now}
	if err := sStore.Create(ctx, expired); err != nil {
		t.Fatal(err)
	}
	if err := sStore.Create(ctx, live); err != nil {
		t.Fatal(err)
	}
	deleted, err := sStore.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := sStore.GetByTokenHash(ctx, "hash-expired"); err == nil {
		t.Fatal("the expired session must be gone")
	}
	if _, err := sStore.GetByTokenHash(ctx, "hash-live"); err != nil {
		t.Fatalf("the live session must survive: %v", err)
	}
}

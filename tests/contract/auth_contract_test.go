package contract_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteAuthContract(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "auth_contract.sqlite")
	runAuthContract(t, database.SQLite, dsn)
}

func TestPostgresAuthContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runAuthContract(t, database.Postgres, dsn)
}

func runAuthContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dialect, err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}

	userStore := authstore.NewUserStore(db, dialect)
	sessionStore := authstore.NewSessionStore(db, dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.TestArgon2Params)

	// 1. Create User
	u, err := authSvc.CreateUser(ctx, "Admin@HydroGo.io", "StrongSecretPassword123!", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if u.Email != "admin@hydrogo.io" {
		t.Fatalf("expected normalized lowercase email, got: %s", u.Email)
	}

	// 2. Duplicate email rejected
	if _, err := authSvc.CreateUser(ctx, "admin@hydrogo.io", "pass", auth.RoleEditor); err != auth.ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got: %v", err)
	}

	// 3. Authenticate with correct password
	authedUser, err := authSvc.Authenticate(ctx, "ADMIN@hydrogo.io", "StrongSecretPassword123!")
	if err != nil {
		t.Fatalf("auth with valid password failed: %v", err)
	}
	if authedUser.ID != u.ID {
		t.Fatalf("authed user ID mismatch: got %s, want %s", authedUser.ID, u.ID)
	}

	// 4. Authenticate with invalid password
	_, err = authSvc.Authenticate(ctx, "admin@hydrogo.io", "wrong-password")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for wrong password, got: %v", err)
	}

	// 5. Create Session
	rawToken, sess, err := authSvc.CreateSession(ctx, u.ID, "TestAgent/1.0", "127.0.0.1", 1*time.Hour)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if rawToken == "" || sess.TokenHash == "" {
		t.Fatalf("expected non-empty rawToken and tokenHash")
	}

	// 6. Validate Session
	valSess, valUser, err := authSvc.ValidateSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("validate session failed: %v", err)
	}
	if valSess.ID != sess.ID || valUser.ID != u.ID {
		t.Fatalf("session or user mismatch")
	}

	// 7. Revoke Session
	if err := authSvc.RevokeSession(ctx, rawToken); err != nil {
		t.Fatalf("revoke session failed: %v", err)
	}

	// 8. Validate revoked session fails
	if _, _, err := authSvc.ValidateSession(ctx, rawToken); err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound for revoked session, got: %v", err)
	}
}

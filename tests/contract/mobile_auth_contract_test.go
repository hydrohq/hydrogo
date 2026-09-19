package contract_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteMobileAuthContract(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "mobile-contract.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	runMobileAuthContract(t, db, database.SQLite)
}

func TestPostgresMobileAuthContract(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMobileAuthContract(t, openIsolatedPostgres(t, context.Background()), database.Postgres)
}

func runMobileAuthContract(t *testing.T, db *sql.DB, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}
	userID := insertMobileContractUser(t, db, dialect)
	mobile := auth.NewMobileAuthService(db, dialect)

	pair, err := mobile.IssueTokenPair(ctx, userID, "contract-device", "Contract phone", "iOS")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if gotUserID, gotDeviceID, err := mobile.ValidateAccessToken(ctx, pair.AccessToken); err != nil || gotUserID != userID || gotDeviceID != "contract-device" {
		t.Fatalf("validate issued access token = %q, %q, %v", gotUserID, gotDeviceID, err)
	}

	rotated, err := mobile.RefreshTokenPair(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := mobile.RefreshTokenPair(ctx, pair.RefreshToken); !errors.Is(err, auth.ErrTokenReused) {
		t.Fatalf("reused refresh error = %v, want ErrTokenReused", err)
	}
	if _, _, err := mobile.ValidateAccessToken(ctx, rotated.AccessToken); !errors.Is(err, auth.ErrDeviceRevoked) {
		t.Fatalf("family reuse must revoke rotated access token, got %v", err)
	}

	pair, err = mobile.IssueTokenPair(ctx, userID, "device-to-revoke", "Contract phone", "Android")
	if err != nil {
		t.Fatalf("issue revocable session: %v", err)
	}
	if err := mobile.RevokeDevice(ctx, userID, "device-to-revoke"); err != nil {
		t.Fatalf("revoke device: %v", err)
	}
	if _, _, err := mobile.ValidateAccessToken(ctx, pair.AccessToken); !errors.Is(err, auth.ErrDeviceRevoked) {
		t.Fatalf("revoked device access token = %v, want ErrDeviceRevoked", err)
	}

	if dialect == database.Postgres {
		testConcurrentPostgresRefresh(t, mobile, userID)
	}

	pair, err = mobile.IssueTokenPair(ctx, userID, "disabled-user-device", "Contract phone", "iOS")
	if err != nil {
		t.Fatalf("issue disabled-user session: %v", err)
	}
	if dialect == database.Postgres {
		_, err = db.ExecContext(ctx, `UPDATE users SET active = $1 WHERE id = $2`, false, userID)
	} else {
		_, err = db.ExecContext(ctx, `UPDATE users SET active = ? WHERE id = ?`, 0, userID)
	}
	if err != nil {
		t.Fatalf("disable mobile user: %v", err)
	}
	if _, err := mobile.RefreshTokenPair(ctx, pair.RefreshToken); !errors.Is(err, auth.ErrUserInactive) {
		t.Fatalf("disabled user's refresh error = %v, want ErrUserInactive", err)
	}
	if _, _, err := mobile.ValidateAccessToken(ctx, pair.AccessToken); !errors.Is(err, auth.ErrDeviceRevoked) {
		t.Fatalf("disabled user's access token error = %v, want ErrDeviceRevoked", err)
	}

	testMultiTenantMobileAuthIsolation(t, db, dialect)
}

func testMultiTenantMobileAuthIsolation(t *testing.T, db *sql.DB, dialect database.Dialect) {
	ctx := context.Background()
	scopeAlpha, _ := tenant.NewScope("tenant-alpha", "org-alpha")
	scopeBeta, _ := tenant.NewScope("tenant-beta", "org-beta")
	ctxAlpha := tenant.WithScope(ctx, scopeAlpha)
	ctxBeta := tenant.WithScope(ctx, scopeBeta)

	mobile := auth.NewMobileAuthService(db, dialect)
	mobile.RequireTenantScope = true

	userAlphaID := "user-alpha-mobile-contract"
	now := time.Now().UTC()
	if dialect == database.Postgres {
		_, err := db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, role, active, tenant_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, userAlphaID, "alpha@example.test", "hash", "editor", true, "tenant-alpha", now, now)
		if err != nil {
			t.Fatalf("insert alpha user: %v", err)
		}
	} else {
		_, err := db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, role, active, tenant_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, userAlphaID, "alpha@example.test", "hash", "editor", 1, "tenant-alpha", now.Format(time.RFC3339), now.Format(time.RFC3339))
		if err != nil {
			t.Fatalf("insert alpha user sqlite: %v", err)
		}
	}

	pair, err := mobile.IssueTokenPair(ctxAlpha, userAlphaID, "alpha-phone", "iPhone 16", "iOS")
	if err != nil {
		t.Fatalf("issue alpha token pair: %v", err)
	}

	uid, devID, err := mobile.ValidateAccessToken(ctxAlpha, pair.AccessToken)
	if err != nil || uid != userAlphaID || devID != "alpha-phone" {
		t.Fatalf("validate token under tenant alpha: got uid=%s, devID=%s, err=%v", uid, devID, err)
	}

	_, _, err = mobile.ValidateAccessToken(ctxBeta, pair.AccessToken)
	if err == nil {
		t.Fatalf("expected cross-tenant validate access token under tenant beta to fail, but succeeded")
	}

	_, err = mobile.RefreshTokenPair(ctxBeta, pair.RefreshToken)
	if err == nil {
		t.Fatalf("expected cross-tenant refresh under tenant beta to fail, but succeeded")
	}

	_, _, err = mobile.ValidateAccessToken(context.Background(), pair.AccessToken)
	if err == nil {
		t.Fatalf("expected missing tenant context to fail closed")
	}
}

func insertMobileContractUser(t *testing.T, db *sql.DB, dialect database.Dialect) string {
	t.Helper()
	const userID = "mobile-contract-user"
	now := time.Now().UTC()
	if dialect == database.Postgres {
		_, err := db.Exec(`INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`, userID, "mobile-contract@example.test", "not-used", "editor", true, now, now)
		if err != nil {
			t.Fatalf("insert postgres user: %v", err)
		}
		return userID
	}
	_, err := db.Exec(`INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, "mobile-contract@example.test", "not-used", "editor", 1, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert sqlite user: %v", err)
	}
	return userID
}

func testConcurrentPostgresRefresh(t *testing.T, mobile *auth.MobileAuthService, userID string) {
	t.Helper()
	pair, err := mobile.IssueTokenPair(context.Background(), userID, "concurrent-device", "Contract phone", "iOS")
	if err != nil {
		t.Fatalf("issue concurrent session: %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, refreshErr := mobile.RefreshTokenPair(context.Background(), pair.RefreshToken)
			results <- refreshErr
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for refreshErr := range results {
		if refreshErr == nil {
			successes++
			continue
		}
		if !errors.Is(refreshErr, auth.ErrTokenReused) {
			t.Fatalf("concurrent refresh error = %v, want ErrTokenReused", refreshErr)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent refresh successes = %d, want exactly 1", successes)
	}
}

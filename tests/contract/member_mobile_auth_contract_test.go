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

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/members"
)

func TestSQLiteMemberMobileAuthContract(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "member-mobile-contract.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	runMemberMobileAuthContract(t, db, database.SQLite)
}

func TestPostgresMemberMobileAuthContract(t *testing.T) {
	if os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runMemberMobileAuthContract(t, openIsolatedPostgres(t, context.Background()), database.Postgres)
}

func runMemberMobileAuthContract(t *testing.T, db *sql.DB, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}
	const memberID = "member-mobile-contract"
	now := time.Now().UTC()
	if dialect == database.Postgres {
		if _, err := db.ExecContext(ctx, `INSERT INTO members (id,email,password_hash,active,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6)`, memberID, "member-contract@example.test", "not-used", true, now, now); err != nil {
			t.Fatalf("insert postgres member: %v", err)
		}
	} else if _, err := db.ExecContext(ctx, `INSERT INTO members (id,email,password_hash,active,created_at,updated_at) VALUES (?,?,?,?,?,?)`, memberID, "member-contract@example.test", "not-used", 1, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert sqlite member: %v", err)
	}
	mobile := members.MobileAuthService{DB: db, Dialect: dialect}
	pair, err := mobile.Issue(ctx, memberID, "contract-device", "Contract phone", "iOS")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if gotMemberID, gotDeviceID, err := mobile.ValidateAccess(ctx, pair.AccessToken); err != nil || gotMemberID != memberID || gotDeviceID != "contract-device" {
		t.Fatalf("validate issued access token = %q, %q, %v", gotMemberID, gotDeviceID, err)
	}
	rotated, err := mobile.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := mobile.Refresh(ctx, pair.RefreshToken); !errors.Is(err, members.ErrMobileTokenInvalid) {
		t.Fatalf("reused refresh error = %v, want ErrMobileTokenInvalid", err)
	}
	if _, _, err := mobile.ValidateAccess(ctx, rotated.AccessToken); !errors.Is(err, members.ErrMobileTokenInvalid) {
		t.Fatalf("family reuse must revoke rotated access token, got %v", err)
	}

	if dialect == database.Postgres {
		testConcurrentMemberRefresh(t, &mobile, memberID)
	}
	activePair, err := mobile.Issue(ctx, memberID, "disabled-device", "Contract phone", "Android")
	if err != nil {
		t.Fatalf("issue disabled session: %v", err)
	}
	if dialect == database.Postgres {
		_, err = db.ExecContext(ctx, `UPDATE members SET active = $1 WHERE id = $2`, false, memberID)
	} else {
		_, err = db.ExecContext(ctx, `UPDATE members SET active = ? WHERE id = ?`, 0, memberID)
	}
	if err != nil {
		t.Fatalf("disable member: %v", err)
	}
	if _, _, err := mobile.ValidateAccess(ctx, activePair.AccessToken); !errors.Is(err, members.ErrMobileTokenInvalid) {
		t.Fatalf("disabled member access error = %v, want ErrMobileTokenInvalid", err)
	}

	testMultiTenantMemberMobileAuthIsolation(t, db, dialect)
}

func testMultiTenantMemberMobileAuthIsolation(t *testing.T, db *sql.DB, dialect database.Dialect) {
	ctx := context.Background()
	scopeAlpha, _ := tenant.NewScope("tenant-alpha", "org-alpha")
	scopeBeta, _ := tenant.NewScope("tenant-beta", "org-beta")
	ctxAlpha := tenant.WithScope(ctx, scopeAlpha)
	ctxBeta := tenant.WithScope(ctx, scopeBeta)

	mobile := members.MobileAuthService{DB: db, Dialect: dialect, RequireTenantScope: true}

	memberAlphaID := "member-alpha-mobile-contract"
	now := time.Now().UTC()
	if dialect == database.Postgres {
		_, err := db.ExecContext(ctx, `INSERT INTO members (id, email, password_hash, active, tenant_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, memberAlphaID, "member-alpha@example.test", "hash", true, "tenant-alpha", now, now)
		if err != nil {
			t.Fatalf("insert alpha member: %v", err)
		}
	} else {
		_, err := db.ExecContext(ctx, `INSERT INTO members (id, email, password_hash, active, tenant_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, memberAlphaID, "member-alpha@example.test", "hash", 1, "tenant-alpha", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatalf("insert alpha member sqlite: %v", err)
		}
	}

	pair, err := mobile.Issue(ctxAlpha, memberAlphaID, "alpha-phone", "iPhone 16", "iOS")
	if err != nil {
		t.Fatalf("issue alpha member token: %v", err)
	}

	mid, devID, err := mobile.ValidateAccess(ctxAlpha, pair.AccessToken)
	if err != nil || mid != memberAlphaID || devID != "alpha-phone" {
		t.Fatalf("validate member token under tenant alpha: got mid=%s, devID=%s, err=%v", mid, devID, err)
	}

	_, _, err = mobile.ValidateAccess(ctxBeta, pair.AccessToken)
	if err == nil {
		t.Fatalf("expected cross-tenant validate member token under tenant beta to fail, but succeeded")
	}

	_, err = mobile.Refresh(ctxBeta, pair.RefreshToken)
	if err == nil {
		t.Fatalf("expected cross-tenant member refresh under tenant beta to fail, but succeeded")
	}

	devices, err := mobile.ListDevices(ctxBeta, memberAlphaID, "alpha-phone")
	if err != nil || len(devices) != 0 {
		t.Fatalf("expected 0 devices for alpha member under tenant beta, got %d, err=%v", len(devices), err)
	}
}

func testConcurrentMemberRefresh(t *testing.T, mobile *members.MobileAuthService, memberID string) {
	t.Helper()
	pair, err := mobile.Issue(context.Background(), memberID, "concurrent-device", "Contract phone", "iOS")
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
			_, refreshErr := mobile.Refresh(context.Background(), pair.RefreshToken)
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
		if !errors.Is(refreshErr, members.ErrMobileTokenInvalid) {
			t.Fatalf("concurrent refresh error = %v, want ErrMobileTokenInvalid", refreshErr)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent refresh successes = %d, want exactly 1", successes)
	}
}

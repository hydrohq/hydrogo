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

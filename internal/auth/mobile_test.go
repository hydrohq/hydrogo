package auth_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func setupTestDB(t *testing.T) (*sql.DB, database.Dialect) {
	t.Helper()
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "mobile_test.sqlite")
	db, err := database.Open(ctx, database.SQLite, dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}

	// Insert test user
	nowStr := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, role, active, created_at, updated_at)
		VALUES ('usr-123', 'mobile@example.com', 'hash', 'editor', 1, ?, ?)`, nowStr, nowStr)
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}

	return db, database.SQLite
}

func TestMobileAuthService_IssueAndRotate(t *testing.T) {
	db, dialect := setupTestDB(t)
	svc := auth.NewMobileAuthService(db, dialect)
	ctx := context.Background()

	// 1. Issue initial token pair
	tokens, err := svc.IssueTokenPair(ctx, "usr-123", "device-ios-1", "iPhone 16 Pro", "iOS")
	if err != nil {
		t.Fatalf("failed to issue tokens: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected non-empty tokens, got: %+v", tokens)
	}

	// 2. Validate Access Token
	uid, devID, err := svc.ValidateAccessToken(ctx, tokens.AccessToken)
	if err != nil {
		t.Fatalf("failed to validate access token: %v", err)
	}
	if uid != "usr-123" || devID != "device-ios-1" {
		t.Fatalf("unexpected validated data: uid=%s, devID=%s", uid, devID)
	}

	// 3. Normal Refresh Token Rotation
	rotatedTokens, err := svc.RefreshTokenPair(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("failed to rotate refresh token: %v", err)
	}
	if rotatedTokens.AccessToken == "" || rotatedTokens.RefreshToken == "" {
		t.Fatalf("expected non-empty rotated tokens: %+v", rotatedTokens)
	}

	// 4. Token Reuse Attack Detection: Presenting the OLD refresh token again must fail and revoke family
	_, err = svc.RefreshTokenPair(ctx, tokens.RefreshToken)
	if err != auth.ErrTokenReused {
		t.Fatalf("expected ErrTokenReused, got: %v", err)
	}

	// 5. Subsequent attempts with the newer rotated token must now also fail because family was revoked!
	_, err = svc.RefreshTokenPair(ctx, rotatedTokens.RefreshToken)
	if err != auth.ErrInvalidToken && err != auth.ErrTokenReused {
		t.Fatalf("expected family revocation rejection, got: %v", err)
	}
}

func TestMobileAuthService_DeviceRevocation(t *testing.T) {
	db, dialect := setupTestDB(t)
	svc := auth.NewMobileAuthService(db, dialect)
	ctx := context.Background()

	tokens1, err := svc.IssueTokenPair(ctx, "usr-123", "device-iphone", "iPhone 15", "iOS")
	if err != nil {
		t.Fatalf("failed to issue token 1: %v", err)
	}
	tokens2, err := svc.IssueTokenPair(ctx, "usr-123", "device-android", "Pixel 9", "Android")
	if err != nil {
		t.Fatalf("failed to issue token 2: %v", err)
	}

	// Revoke device-iphone only
	if err := svc.RevokeDevice(ctx, "usr-123", "device-iphone"); err != nil {
		t.Fatalf("failed to revoke device: %v", err)
	}

	// iPhone token should now be rejected
	_, _, err = svc.ValidateAccessToken(ctx, tokens1.AccessToken)
	if err != auth.ErrDeviceRevoked {
		t.Fatalf("expected ErrDeviceRevoked, got: %v", err)
	}

	// Android token should still be valid
	uid, devID, err := svc.ValidateAccessToken(ctx, tokens2.AccessToken)
	if err != nil || devID != "device-android" {
		t.Fatalf("expected android device to remain valid, got uid=%s, devID=%s, err=%v", uid, devID, err)
	}

	// Revoke all devices
	if err := svc.RevokeAllDevices(ctx, "usr-123"); err != nil {
		t.Fatalf("failed to revoke all devices: %v", err)
	}

	// Android token should now also be rejected
	_, _, err = svc.ValidateAccessToken(ctx, tokens2.AccessToken)
	if err != auth.ErrDeviceRevoked {
		t.Fatalf("expected ErrDeviceRevoked after all-devices revocation, got: %v", err)
	}
}

func TestMobileAuthService_RejectsExpiredAccessToken(t *testing.T) {
	db, dialect := setupTestDB(t)
	svc := auth.NewMobileAuthService(db, dialect)
	ctx := context.Background()

	tokens, err := svc.IssueTokenPair(ctx, "usr-123", "device-expiring", "Test device", "iOS")
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}
	_, err = db.ExecContext(ctx, `UPDATE mobile_device_sessions SET access_expires_at = ? WHERE access_token_hash = ?`, time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), tokenHashForTest(tokens.AccessToken))
	if err != nil {
		t.Fatalf("expire access token: %v", err)
	}
	if _, _, err := svc.ValidateAccessToken(ctx, tokens.AccessToken); err != auth.ErrInvalidToken {
		t.Fatalf("ValidateAccessToken error = %v, want ErrInvalidToken", err)
	}
}

func tokenHashForTest(raw string) string {
	// The service deliberately stores only SHA-256 token digests.
	return fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
}

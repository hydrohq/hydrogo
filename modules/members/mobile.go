package members

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrMobileTokenInvalid = errors.New("invalid or expired member mobile token")

type MobileAuthService struct {
	DB      *sql.DB
	Dialect database.Dialect
}

func (s MobileAuthService) Issue(ctx context.Context, memberID, deviceID, deviceName, platform string) (*auth.TokenPair, error) {
	if s.DB == nil || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(platform) == "" {
		return nil, ErrMobileTokenInvalid
	}
	access, accessHash, err := auth.GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	refresh, refreshHash, err := auth.GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	id, _, err := auth.GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	family, _, err := auth.GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	accessUntil := now.Add(auth.AccessTokenTTL)
	refreshUntil := now.Add(auth.RefreshTokenTTL)
	q := `INSERT INTO member_mobile_device_sessions (id,member_id,device_id,device_name,platform,family_id,refresh_token_hash,access_token_hash,access_expires_at,refresh_expires_at,created_at,last_used_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
	args := []any{id, memberID, deviceID, deviceName, platform, family, refreshHash, accessHash, accessUntil.Format(time.RFC3339Nano), refreshUntil.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}
	if s.Dialect == database.Postgres {
		q = `INSERT INTO member_mobile_device_sessions (id,member_id,device_id,device_name,platform,family_id,refresh_token_hash,access_token_hash,access_expires_at,refresh_expires_at,created_at,last_used_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
		args = []any{id, memberID, deviceID, deviceName, platform, family, refreshHash, accessHash, accessUntil, refreshUntil, now, now}
	}
	if _, err = s.DB.ExecContext(ctx, q, args...); err != nil {
		return nil, fmt.Errorf("create member mobile session: %w", err)
	}
	return &auth.TokenPair{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresInSeconds: int64(auth.AccessTokenTTL.Seconds()), ExpiresAt: accessUntil}, nil
}

func (s MobileAuthService) ValidateAccess(ctx context.Context, raw string) (string, string, error) {
	q := `SELECT member_id,device_id,revoked_at,access_expires_at FROM member_mobile_device_sessions WHERE access_token_hash = ?`
	if s.Dialect == database.Postgres {
		q = `SELECT member_id,device_id,revoked_at,access_expires_at FROM member_mobile_device_sessions WHERE access_token_hash = $1`
	}
	var memberID, deviceID string
	var revoked, expires any
	if err := s.DB.QueryRowContext(ctx, q, auth.HashSessionToken(raw)).Scan(&memberID, &deviceID, &revoked, &expires); err != nil {
		return "", "", ErrMobileTokenInvalid
	}
	if mobileTime(revoked).IsZero() && time.Now().UTC().Before(mobileTime(expires)) {
		return memberID, deviceID, nil
	}
	return "", "", ErrMobileTokenInvalid
}

func (s MobileAuthService) RevokeAll(ctx context.Context, memberID string) error {
	return s.revoke(ctx, "member_id", memberID)
}
func (s MobileAuthService) RevokeDevice(ctx context.Context, memberID, deviceID string) error {
	return s.revoke(ctx, "member_id = "+s.ph(1)+" AND device_id", memberID, deviceID)
}
func (s MobileAuthService) ph(n int) string {
	if s.Dialect == database.Postgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
func (s MobileAuthService) revoke(ctx context.Context, field string, values ...string) error {
	q := `UPDATE member_mobile_device_sessions SET revoked_at = ` + s.ph(1) + ` WHERE ` + field + ` = ` + s.ph(2) + ` AND revoked_at IS NULL`
	args := []any{time.Now().UTC().Format(time.RFC3339Nano), values[0]}
	if s.Dialect == database.Postgres {
		args = []any{time.Now().UTC(), values[0]}
	}
	if len(values) == 2 {
		q = `UPDATE member_mobile_device_sessions SET revoked_at = ` + s.ph(1) + ` WHERE member_id = ` + s.ph(2) + ` AND device_id = ` + s.ph(3) + ` AND revoked_at IS NULL`
		args = append(args, values[1])
	}
	_, err := s.DB.ExecContext(ctx, q, args...)
	return err
}
func mobileTime(v any) time.Time {
	switch x := v.(type) {
	case time.Time:
		return x.UTC()
	case string:
		t, _ := time.Parse(time.RFC3339Nano, x)
		return t
	case []byte:
		return mobileTime(string(x))
	default:
		return time.Time{}
	}
}

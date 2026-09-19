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
	if s.DB == nil || strings.TrimSpace(raw) == "" {
		return "", "", ErrMobileTokenInvalid
	}
	q := `SELECT s.member_id,s.device_id,s.revoked_at,s.access_expires_at,m.active FROM member_mobile_device_sessions s JOIN members m ON m.id=s.member_id WHERE s.access_token_hash = ?`
	if s.Dialect == database.Postgres {
		q = `SELECT s.member_id,s.device_id,s.revoked_at,s.access_expires_at,m.active FROM member_mobile_device_sessions s JOIN members m ON m.id=s.member_id WHERE s.access_token_hash = $1`
	}
	var memberID, deviceID string
	var revoked, expires, active any
	if err := s.DB.QueryRowContext(ctx, q, auth.HashSessionToken(raw)).Scan(&memberID, &deviceID, &revoked, &expires, &active); err != nil {
		return "", "", ErrMobileTokenInvalid
	}
	if mobileTime(revoked).IsZero() && time.Now().UTC().Before(mobileTime(expires)) && mobileActive(active) {
		return memberID, deviceID, nil
	}
	return "", "", ErrMobileTokenInvalid
}

// Refresh rotates a single-use member refresh token. A reused token revokes its
// whole family so a stolen older token cannot keep a member signed in.
func (s MobileAuthService) Refresh(ctx context.Context, raw string) (*auth.TokenPair, error) {
	if s.DB == nil || strings.TrimSpace(raw) == "" {
		return nil, ErrMobileTokenInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := `SELECT s.id,s.member_id,s.device_id,s.device_name,s.platform,s.family_id,s.revoked_at,s.refresh_expires_at,m.active FROM member_mobile_device_sessions s JOIN members m ON m.id=s.member_id WHERE s.refresh_token_hash=?`
	if s.Dialect == database.Postgres {
		q += ` FOR UPDATE`
		q = strings.Replace(q, "?", "$1", 1)
	}
	var id, memberID, deviceID, deviceName, platform, family string
	var revoked, expires, active any
	if err := tx.QueryRowContext(ctx, q, auth.HashSessionToken(raw)).Scan(&id, &memberID, &deviceID, &deviceName, &platform, &family, &revoked, &expires, &active); err != nil {
		return nil, ErrMobileTokenInvalid
	}
	now := time.Now().UTC()
	if !mobileTime(revoked).IsZero() || !now.Before(mobileTime(expires)) || !mobileActive(active) {
		_, _ = tx.ExecContext(ctx, s.revokeFamilyQuery(), s.revokeFamilyArgs(now, family)...)
		_ = tx.Commit()
		return nil, ErrMobileTokenInvalid
	}
	update := `UPDATE member_mobile_device_sessions SET revoked_at=? WHERE id=? AND revoked_at IS NULL`
	args := []any{now.Format(time.RFC3339Nano), id}
	if s.Dialect == database.Postgres {
		update = `UPDATE member_mobile_device_sessions SET revoked_at=$1 WHERE id=$2 AND revoked_at IS NULL`
		args = []any{now, id}
	}
	result, err := tx.ExecContext(ctx, update, args...)
	if err != nil {
		return nil, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
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
	newID, _, err := auth.GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	accessUntil := now.Add(auth.AccessTokenTTL)
	refreshUntil := now.Add(auth.RefreshTokenTTL)
	insert := `INSERT INTO member_mobile_device_sessions (id,member_id,device_id,device_name,platform,family_id,refresh_token_hash,access_token_hash,access_expires_at,refresh_expires_at,created_at,last_used_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`
	values := []any{newID, memberID, deviceID, deviceName, platform, family, refreshHash, accessHash, accessUntil.Format(time.RFC3339Nano), refreshUntil.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}
	if s.Dialect == database.Postgres {
		insert = `INSERT INTO member_mobile_device_sessions (id,member_id,device_id,device_name,platform,family_id,refresh_token_hash,access_token_hash,access_expires_at,refresh_expires_at,created_at,last_used_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
		values = []any{newID, memberID, deviceID, deviceName, platform, family, refreshHash, accessHash, accessUntil, refreshUntil, now, now}
	}
	if _, err = tx.ExecContext(ctx, insert, values...); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &auth.TokenPair{AccessToken: access, RefreshToken: refresh, TokenType: "Bearer", ExpiresInSeconds: int64(auth.AccessTokenTTL.Seconds()), ExpiresAt: accessUntil}, nil
}
func (s MobileAuthService) revokeFamilyQuery() string {
	if s.Dialect == database.Postgres {
		return `UPDATE member_mobile_device_sessions SET revoked_at=$1 WHERE family_id=$2 AND revoked_at IS NULL`
	}
	return `UPDATE member_mobile_device_sessions SET revoked_at=? WHERE family_id=? AND revoked_at IS NULL`
}
func (s MobileAuthService) revokeFamilyArgs(now time.Time, family string) []any {
	if s.Dialect == database.Postgres {
		return []any{now, family}
	}
	return []any{now.Format(time.RFC3339Nano), family}
}
func mobileActive(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case int:
		return x != 0
	case []byte:
		return mobileActive(string(x))
	case string:
		return x == "1" || strings.EqualFold(x, "true")
	}
	return false
}

func (s MobileAuthService) RevokeAll(ctx context.Context, memberID string) error {
	if s.DB == nil || strings.TrimSpace(memberID) == "" {
		return ErrMobileTokenInvalid
	}
	return s.revoke(ctx, "member_id", memberID)
}
func (s MobileAuthService) RevokeDevice(ctx context.Context, memberID, deviceID string) error {
	if s.DB == nil || strings.TrimSpace(memberID) == "" || strings.TrimSpace(deviceID) == "" {
		return ErrMobileTokenInvalid
	}
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

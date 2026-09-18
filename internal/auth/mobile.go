package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var (
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrTokenReused      = errors.New("security alert: refresh token reused, session family revoked")
	ErrDeviceRevoked    = errors.New("device session has been revoked")
	ErrDeviceIDRequired = errors.New("device_id is required")
	ErrPlatformRequired = errors.New("platform is required")
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 30 * 24 * time.Hour
)

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	ExpiresInSeconds int64     `json:"expires_in"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type MobileDeviceSession struct {
	ID               string
	UserID           string
	DeviceID         string
	DeviceName       string
	Platform         string
	FamilyID         string
	RefreshTokenHash string
	AccessTokenHash  string
	RevokedAt        *time.Time
	ExpiresAt        time.Time
	CreatedAt        time.Time
	LastUsedAt       time.Time
}

type DeviceSessionSummary struct {
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name"`
	Platform   string    `json:"platform"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	IsCurrent  bool      `json:"is_current"`
}

type MobileAuthService struct {
	db      *sql.DB
	dialect database.Dialect
}

func NewMobileAuthService(db *sql.DB, dialect database.Dialect) *MobileAuthService {
	return &MobileAuthService{db: db, dialect: dialect}
}

func generateSecureToken(prefix string, byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(rawToken)))
	return hex.EncodeToString(sum[:])
}

// IssueTokenPair creates a new mobile device session and returns a fresh access/refresh token pair.
func (s *MobileAuthService) IssueTokenPair(ctx context.Context, userID, deviceID, deviceName, platform string) (*TokenPair, error) {
	if strings.TrimSpace(deviceID) == "" {
		return nil, ErrDeviceIDRequired
	}
	if strings.TrimSpace(platform) == "" {
		return nil, ErrPlatformRequired
	}

	rawAccessToken, err := generateSecureToken("mat", 32)
	if err != nil {
		return nil, err
	}
	rawRefreshToken, err := generateSecureToken("mrt", 48)
	if err != nil {
		return nil, err
	}

	sessionID, err := generateSecureToken("mds", 16)
	if err != nil {
		return nil, err
	}
	familyID, err := generateSecureToken("fam", 16)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	accessExpiry := now.Add(AccessTokenTTL)
	refreshExpiry := now.Add(RefreshTokenTTL)

	accessHash := hashToken(rawAccessToken)
	refreshHash := hashToken(rawRefreshToken)

	var query string
	if s.dialect == database.Postgres {
		query = `INSERT INTO mobile_device_sessions
			(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, expires_at, created_at, last_used_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	} else {
		query = `INSERT INTO mobile_device_sessions
			(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, expires_at, created_at, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	nowStr := now.Format(time.RFC3339)
	expiryStr := refreshExpiry.Format(time.RFC3339)

	var dbExpires any = expiryStr
	var dbCreated any = nowStr
	var dbLastUsed any = nowStr
	if s.dialect == database.Postgres {
		dbExpires = refreshExpiry
		dbCreated = now
		dbLastUsed = now
	}

	_, err = s.db.ExecContext(ctx, query, sessionID, userID, deviceID, deviceName, platform, familyID, refreshHash, accessHash, dbExpires, dbCreated, dbLastUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to create mobile session: %w", err)
	}

	return &TokenPair{
		AccessToken:      rawAccessToken,
		RefreshToken:     rawRefreshToken,
		TokenType:        "Bearer",
		ExpiresInSeconds: int64(AccessTokenTTL.Seconds()),
		ExpiresAt:        accessExpiry,
	}, nil
}

func parseMobileTime(v any) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val.UTC()
	case string:
		t, err := time.Parse(time.RFC3339Nano, val)
		if err == nil {
			return t.UTC()
		}
		t, err = time.Parse(time.RFC3339, val)
		if err == nil {
			return t.UTC()
		}
		return time.Time{}
	case []byte:
		return parseMobileTime(string(val))
	default:
		return time.Time{}
	}
}

// RefreshTokenPair implements single-use Refresh Token Rotation with Token Family reuse detection.
func (s *MobileAuthService) RefreshTokenPair(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	refreshHash := hashToken(rawRefreshToken)

	var (
		sessionID    string
		userID       string
		deviceID     string
		deviceName   string
		platform     string
		familyID     string
		revokedAtVal any
		expiresAtVal any
	)

	var query string
	if s.dialect == database.Postgres {
		query = `SELECT id, user_id, device_id, device_name, platform, family_id, revoked_at, expires_at
			FROM mobile_device_sessions WHERE refresh_token_hash = $1`
	} else {
		query = `SELECT id, user_id, device_id, device_name, platform, family_id, revoked_at, expires_at
			FROM mobile_device_sessions WHERE refresh_token_hash = ?`
	}

	row := s.db.QueryRowContext(ctx, query, refreshHash)
	if err := row.Scan(&sessionID, &userID, &deviceID, &deviceName, &platform, &familyID, &revokedAtVal, &expiresAtVal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	now := time.Now().UTC()
	revokedAt := parseMobileTime(revokedAtVal)
	expiresAt := parseMobileTime(expiresAtVal)

	// Token Family Reuse Detection:
	// If a revoked/already-used refresh token is presented again, someone might have intercepted it!
	// Invalidate the ENTIRE token family immediately.
	if !revokedAt.IsZero() {
		_ = s.RevokeFamily(ctx, familyID)
		return nil, ErrTokenReused
	}

	if now.After(expiresAt) {
		return nil, ErrInvalidToken
	}

	// Invalidate the current session token (single-use rotation)
	var revokeQuery string
	if s.dialect == database.Postgres {
		revokeQuery = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE id = $2`
		_, _ = s.db.ExecContext(ctx, revokeQuery, now, sessionID)
	} else {
		revokeQuery = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE id = ?`
		_, _ = s.db.ExecContext(ctx, revokeQuery, now.Format(time.RFC3339), sessionID)
	}

	// Issue replacement token pair within the same Family
	rawAccessToken, err := generateSecureToken("mat", 32)
	if err != nil {
		return nil, err
	}
	rawRefreshToken, err = generateSecureToken("mrt", 48)
	if err != nil {
		return nil, err
	}

	newSessionID, err := generateSecureToken("mds", 16)
	if err != nil {
		return nil, err
	}

	accessExpiry := now.Add(AccessTokenTTL)
	refreshExpiry := now.Add(RefreshTokenTTL)

	newAccessHash := hashToken(rawAccessToken)
	newRefreshHash := hashToken(rawRefreshToken)

	var insertQuery string
	if s.dialect == database.Postgres {
		insertQuery = `INSERT INTO mobile_device_sessions
			(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, expires_at, created_at, last_used_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	} else {
		insertQuery = `INSERT INTO mobile_device_sessions
			(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, expires_at, created_at, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	nowStr := now.Format(time.RFC3339)
	expiryStr := refreshExpiry.Format(time.RFC3339)

	var dbExpires any = expiryStr
	var dbCreated any = nowStr
	var dbLastUsed any = nowStr
	if s.dialect == database.Postgres {
		dbExpires = refreshExpiry
		dbCreated = now
		dbLastUsed = now
	}

	_, err = s.db.ExecContext(ctx, insertQuery, newSessionID, userID, deviceID, deviceName, platform, familyID, newRefreshHash, newAccessHash, dbExpires, dbCreated, dbLastUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate mobile session: %w", err)
	}

	return &TokenPair{
		AccessToken:      rawAccessToken,
		RefreshToken:     rawRefreshToken,
		TokenType:        "Bearer",
		ExpiresInSeconds: int64(AccessTokenTTL.Seconds()),
		ExpiresAt:        accessExpiry,
	}, nil
}

// ValidateAccessToken checks the validity of a bearer access token.
func (s *MobileAuthService) ValidateAccessToken(ctx context.Context, rawAccessToken string) (userID, deviceID string, err error) {
	accessHash := hashToken(rawAccessToken)

	var (
		revokedAtVal any
		expiresAtVal any
	)

	var query string
	if s.dialect == database.Postgres {
		query = `SELECT user_id, device_id, revoked_at, expires_at FROM mobile_device_sessions WHERE access_token_hash = $1`
	} else {
		query = `SELECT user_id, device_id, revoked_at, expires_at FROM mobile_device_sessions WHERE access_token_hash = ?`
	}

	row := s.db.QueryRowContext(ctx, query, accessHash)
	if err := row.Scan(&userID, &deviceID, &revokedAtVal, &expiresAtVal); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrInvalidToken
		}
		return "", "", err
	}

	revokedAt := parseMobileTime(revokedAtVal)
	expiresAt := parseMobileTime(expiresAtVal)

	if !revokedAt.IsZero() {
		return "", "", ErrDeviceRevoked
	}

	if time.Now().UTC().After(expiresAt) {
		return "", "", ErrInvalidToken
	}

	return userID, deviceID, nil
}

// RevokeFamily invalidates all sessions belonging to a specific family ID.
func (s *MobileAuthService) RevokeFamily(ctx context.Context, familyID string) error {
	now := time.Now().UTC()
	var query string
	if s.dialect == database.Postgres {
		query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`
		_, err := s.db.ExecContext(ctx, query, now, familyID)
		return err
	}
	query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE family_id = ? AND revoked_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, now.Format(time.RFC3339), familyID)
	return err
}

// RevokeDevice invalidates all active sessions for a specific user device.
func (s *MobileAuthService) RevokeDevice(ctx context.Context, userID, deviceID string) error {
	now := time.Now().UTC()
	var query string
	if s.dialect == database.Postgres {
		query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE user_id = $2 AND device_id = $3 AND revoked_at IS NULL`
		_, err := s.db.ExecContext(ctx, query, now, userID, deviceID)
		return err
	}
	query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE user_id = ? AND device_id = ? AND revoked_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, now.Format(time.RFC3339), userID, deviceID)
	return err
}

// RevokeAllDevices logs out the user from all mobile devices.
func (s *MobileAuthService) RevokeAllDevices(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	var query string
	if s.dialect == database.Postgres {
		query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`
		_, err := s.db.ExecContext(ctx, query, now, userID)
		return err
	}
	query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, now.Format(time.RFC3339), userID)
	return err
}

// ListDeviceSessions returns non-revoked device sessions for a user.
func (s *MobileAuthService) ListDeviceSessions(ctx context.Context, userID, currentDeviceID string) ([]DeviceSessionSummary, error) {
	var query string
	if s.dialect == database.Postgres {
		query = `SELECT device_id, device_name, platform, created_at, last_used_at
			FROM mobile_device_sessions WHERE user_id = $1 AND revoked_at IS NULL
			ORDER BY last_used_at DESC`
	} else {
		query = `SELECT device_id, device_name, platform, created_at, last_used_at
			FROM mobile_device_sessions WHERE user_id = ? AND revoked_at IS NULL
			ORDER BY last_used_at DESC`
	}

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []DeviceSessionSummary
	seenDevices := make(map[string]bool)

	for rows.Next() {
		var (
			dID           string
			dName         string
			platform      string
			createdAtVal  any
			lastUsedAtVal any
		)
		if err := rows.Scan(&dID, &dName, &platform, &createdAtVal, &lastUsedAtVal); err != nil {
			return nil, err
		}
		if seenDevices[dID] {
			continue
		}
		seenDevices[dID] = true
		summaries = append(summaries, DeviceSessionSummary{
			DeviceID:   dID,
			DeviceName: dName,
			Platform:   platform,
			CreatedAt:  parseMobileTime(createdAtVal),
			LastUsedAt: parseMobileTime(lastUsedAtVal),
			IsCurrent:  dID == currentDeviceID,
		})
	}

	return summaries, nil
}

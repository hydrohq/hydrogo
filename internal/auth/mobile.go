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
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
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
	db                 *sql.DB
	dialect            database.Dialect
	RequireTenantScope bool
}

func (s *MobileAuthService) tenant(ctx context.Context) (string, error) {
	if !s.RequireTenantScope {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func (s *MobileAuthService) within(ctx context.Context, fn func(database.Executor) error) error {
	if s.db == nil {
		return database.ErrDatabaseRequired
	}
	return database.NewTenantTransactor(s.db, s.dialect, s.RequireTenantScope).Within(ctx, fn)
}

func (s *MobileAuthService) withinQuery(ctx context.Context, fn func(database.QueryExecutor) error) error {
	if s.db == nil {
		return database.ErrDatabaseRequired
	}
	return database.NewTenantTransactor(s.db, s.dialect, s.RequireTenantScope).WithinQuery(ctx, fn)
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
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, ErrInvalidToken
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
			(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, access_expires_at, refresh_expires_at, created_at, last_used_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	} else {
		query = `INSERT INTO mobile_device_sessions
			(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, access_expires_at, refresh_expires_at, created_at, last_used_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	}

	nowStr := now.Format(time.RFC3339)
	accessExpiryStr := accessExpiry.Format(time.RFC3339)
	refreshExpiryStr := refreshExpiry.Format(time.RFC3339)

	var dbAccessExpires any = accessExpiryStr
	var dbRefreshExpires any = refreshExpiryStr
	var dbCreated any = nowStr
	var dbLastUsed any = nowStr
	if s.dialect == database.Postgres {
		dbAccessExpires = accessExpiry
		dbRefreshExpires = refreshExpiry
		dbCreated = now
		dbLastUsed = now
	}

	err = s.within(ctx, func(exec database.Executor) error {
		if s.RequireTenantScope {
			var exists int
			userCheck := `SELECT 1 FROM users WHERE id = ? AND tenant_id = ?`
			if s.dialect == database.Postgres {
				userCheck = `SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2`
			}
			if err := exec.QueryRowContext(ctx, userCheck, userID, tenantID).Scan(&exists); err != nil {
				return ErrInvalidToken
			}
		}
		_, execErr := exec.ExecContext(ctx, query, sessionID, userID, deviceID, deviceName, platform, familyID, refreshHash, accessHash, dbAccessExpires, dbRefreshExpires, dbCreated, dbLastUsed)
		return execErr
	})
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			return nil, ErrInvalidToken
		}
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

func parseMobileBool(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case int64:
		return value != 0
	case int:
		return value != 0
	case string:
		return value == "1" || strings.EqualFold(value, "true")
	case []byte:
		return parseMobileBool(string(value))
	default:
		return false
	}
}

// RefreshTokenPair implements single-use Refresh Token Rotation with Token Family reuse detection.
func (s *MobileAuthService) RefreshTokenPair(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, ErrInvalidToken
	}
	refreshHash := hashToken(rawRefreshToken)

	var (
		sessionID           string
		userID              string
		deviceID            string
		deviceName          string
		platform            string
		familyID            string
		revokedAtVal        any
		refreshExpiresAtVal any
		userActiveVal       any
	)

	var query string
	args := []any{refreshHash}
	if s.RequireTenantScope {
		args = append(args, tenantID)
	}
	if s.dialect == database.Postgres {
		query = `SELECT m.id, m.user_id, m.device_id, m.device_name, m.platform, m.family_id, m.revoked_at, m.refresh_expires_at, u.active
			FROM mobile_device_sessions m JOIN users u ON u.id = m.user_id WHERE m.refresh_token_hash = $1`
		if s.RequireTenantScope {
			query += ` AND u.tenant_id = $2`
		}
		query += ` FOR UPDATE`
	} else {
		query = `SELECT m.id, m.user_id, m.device_id, m.device_name, m.platform, m.family_id, m.revoked_at, m.refresh_expires_at, u.active
			FROM mobile_device_sessions m JOIN users u ON u.id = m.user_id WHERE m.refresh_token_hash = ?`
		if s.RequireTenantScope {
			query += ` AND u.tenant_id = ?`
		}
	}

	var (
		resultPair  *TokenPair
		securityErr error
	)
	err = s.within(ctx, func(tx database.Executor) error {
		row := tx.QueryRowContext(ctx, query, args...)
		if err := row.Scan(&sessionID, &userID, &deviceID, &deviceName, &platform, &familyID, &revokedAtVal, &refreshExpiresAtVal, &userActiveVal); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidToken
			}
			return err
		}

		now := time.Now().UTC()
		revokedAt := parseMobileTime(revokedAtVal)
		refreshExpiresAt := parseMobileTime(refreshExpiresAtVal)

		// Token Family Reuse Detection:
		// If a revoked/already-used refresh token is presented again, someone might have intercepted it!
		// Invalidate the ENTIRE token family immediately.
		if !revokedAt.IsZero() {
			if err := revokeMobileFamily(ctx, tx, s.dialect, familyID, now); err != nil {
				return err
			}
			securityErr = ErrTokenReused
			return nil
		}

		if !now.Before(refreshExpiresAt) {
			return ErrInvalidToken
		}
		if !parseMobileBool(userActiveVal) {
			if err := revokeMobileFamily(ctx, tx, s.dialect, familyID, now); err != nil {
				return err
			}
			securityErr = ErrUserInactive
			return nil
		}

		// Invalidate the current session token (single-use rotation)
		var revokeQuery string
		if s.dialect == database.Postgres {
			revokeQuery = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE id = $2`
			result, updateErr := tx.ExecContext(ctx, revokeQuery+` AND revoked_at IS NULL`, now, sessionID)
			if updateErr != nil {
				return updateErr
			}
			changed, updateErr := result.RowsAffected()
			if updateErr != nil || changed != 1 {
				if updateErr != nil {
					return updateErr
				}
				return ErrTokenReused
			}
		} else {
			revokeQuery = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE id = ?`
			result, updateErr := tx.ExecContext(ctx, revokeQuery+` AND revoked_at IS NULL`, now.Format(time.RFC3339), sessionID)
			if updateErr != nil {
				return updateErr
			}
			changed, updateErr := result.RowsAffected()
			if updateErr != nil || changed != 1 {
				if updateErr != nil {
					return updateErr
				}
				return ErrTokenReused
			}
		}

		// Issue replacement token pair within the same Family
		rawAccessToken, err := generateSecureToken("mat", 32)
		if err != nil {
			return err
		}
		rawRefreshToken, err := generateSecureToken("mrt", 48)
		if err != nil {
			return err
		}

		newSessionID, err := generateSecureToken("mds", 16)
		if err != nil {
			return err
		}

		accessExpiry := now.Add(AccessTokenTTL)
		refreshExpiry := now.Add(RefreshTokenTTL)

		newAccessHash := hashToken(rawAccessToken)
		newRefreshHash := hashToken(rawRefreshToken)

		var insertQuery string
		if s.dialect == database.Postgres {
			insertQuery = `INSERT INTO mobile_device_sessions
				(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, access_expires_at, refresh_expires_at, created_at, last_used_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
		} else {
			insertQuery = `INSERT INTO mobile_device_sessions
				(id, user_id, device_id, device_name, platform, family_id, refresh_token_hash, access_token_hash, access_expires_at, refresh_expires_at, created_at, last_used_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		}

		nowStr := now.Format(time.RFC3339)
		accessExpiryStr := accessExpiry.Format(time.RFC3339)
		refreshExpiryStr := refreshExpiry.Format(time.RFC3339)

		var dbAccessExpires any = accessExpiryStr
		var dbRefreshExpires any = refreshExpiryStr
		var dbCreated any = nowStr
		var dbLastUsed any = nowStr
		if s.dialect == database.Postgres {
			dbAccessExpires = accessExpiry
			dbRefreshExpires = refreshExpiry
			dbCreated = now
			dbLastUsed = now
		}

		_, err = tx.ExecContext(ctx, insertQuery, newSessionID, userID, deviceID, deviceName, platform, familyID, newRefreshHash, newAccessHash, dbAccessExpires, dbRefreshExpires, dbCreated, dbLastUsed)
		if err != nil {
			return fmt.Errorf("failed to rotate mobile session: %w", err)
		}

		resultPair = &TokenPair{
			AccessToken:      rawAccessToken,
			RefreshToken:     rawRefreshToken,
			TokenType:        "Bearer",
			ExpiresInSeconds: int64(AccessTokenTTL.Seconds()),
			ExpiresAt:        accessExpiry,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if securityErr != nil {
		return nil, securityErr
	}
	return resultPair, nil
}

// ValidateAccessToken checks the validity of a bearer access token.
func (s *MobileAuthService) ValidateAccessToken(ctx context.Context, rawAccessToken string) (userID, deviceID string, err error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return "", "", ErrInvalidToken
	}
	accessHash := hashToken(rawAccessToken)

	var (
		revokedAtVal       any
		accessExpiresAtVal any
	)

	var query string
	args := []any{accessHash}
	if s.RequireTenantScope {
		args = append(args, tenantID)
	}
	if s.dialect == database.Postgres {
		query = `SELECT m.user_id, m.device_id, m.revoked_at, m.access_expires_at FROM mobile_device_sessions m JOIN users u ON u.id = m.user_id WHERE m.access_token_hash = $1`
		if s.RequireTenantScope {
			query += ` AND u.tenant_id = $2`
		}
	} else {
		query = `SELECT m.user_id, m.device_id, m.revoked_at, m.access_expires_at FROM mobile_device_sessions m JOIN users u ON u.id = m.user_id WHERE m.access_token_hash = ?`
		if s.RequireTenantScope {
			query += ` AND u.tenant_id = ?`
		}
	}

	err = s.within(ctx, func(exec database.Executor) error {
		row := exec.QueryRowContext(ctx, query, args...)
		if scanErr := row.Scan(&userID, &deviceID, &revokedAtVal, &accessExpiresAtVal); scanErr != nil {
			if errors.Is(scanErr, sql.ErrNoRows) {
				return ErrInvalidToken
			}
			return scanErr
		}
		return nil
	})
	if err != nil {
		return "", "", err
	}

	revokedAt := parseMobileTime(revokedAtVal)
	accessExpiresAt := parseMobileTime(accessExpiresAtVal)

	if !revokedAt.IsZero() {
		return "", "", ErrDeviceRevoked
	}

	if !time.Now().UTC().Before(accessExpiresAt) {
		return "", "", ErrInvalidToken
	}

	return userID, deviceID, nil
}

func revokeMobileFamily(ctx context.Context, tx database.Executor, dialect database.Dialect, familyID string, now time.Time) error {
	query := `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`
	args := []any{now, familyID}
	if dialect == database.SQLite {
		query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE family_id = ? AND revoked_at IS NULL`
		args[0] = now.Format(time.RFC3339)
	}
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

// RevokeFamily invalidates all sessions belonging to a specific family ID.
func (s *MobileAuthService) RevokeFamily(ctx context.Context, familyID string) error {
	now := time.Now().UTC()
	var query string
	var args []any
	if s.dialect == database.Postgres {
		query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE family_id = $2 AND revoked_at IS NULL`
		args = []any{now, familyID}
	} else {
		query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE family_id = ? AND revoked_at IS NULL`
		args = []any{now.Format(time.RFC3339), familyID}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, err := exec.ExecContext(ctx, query, args...)
		return err
	})
}

// RevokeDevice invalidates all active sessions for a specific user device.
func (s *MobileAuthService) RevokeDevice(ctx context.Context, userID, deviceID string) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return ErrInvalidToken
	}
	now := time.Now().UTC()
	var query string
	var args []any
	if s.dialect == database.Postgres {
		query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE user_id = $2 AND device_id = $3 AND revoked_at IS NULL`
		args = []any{now, userID, deviceID}
		if s.RequireTenantScope {
			query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE user_id IN (SELECT id FROM users WHERE id = $2 AND tenant_id = $3) AND device_id = $4 AND revoked_at IS NULL`
			args = []any{now, userID, tenantID, deviceID}
		}
	} else {
		query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE user_id = ? AND device_id = ? AND revoked_at IS NULL`
		args = []any{now.Format(time.RFC3339), userID, deviceID}
		if s.RequireTenantScope {
			query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE user_id IN (SELECT id FROM users WHERE id = ? AND tenant_id = ?) AND device_id = ? AND revoked_at IS NULL`
			args = []any{now.Format(time.RFC3339), userID, tenantID, deviceID}
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, err := exec.ExecContext(ctx, query, args...)
		return err
	})
}

// RevokeAllDevices logs out the user from all mobile devices.
func (s *MobileAuthService) RevokeAllDevices(ctx context.Context, userID string) error {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return ErrInvalidToken
	}
	now := time.Now().UTC()
	var query string
	var args []any
	if s.dialect == database.Postgres {
		query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`
		args = []any{now, userID}
		if s.RequireTenantScope {
			query = `UPDATE mobile_device_sessions SET revoked_at = $1 WHERE user_id IN (SELECT id FROM users WHERE id = $2 AND tenant_id = $3) AND revoked_at IS NULL`
			args = []any{now, userID, tenantID}
		}
	} else {
		query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`
		args = []any{now.Format(time.RFC3339), userID}
		if s.RequireTenantScope {
			query = `UPDATE mobile_device_sessions SET revoked_at = ? WHERE user_id IN (SELECT id FROM users WHERE id = ? AND tenant_id = ?) AND revoked_at IS NULL`
			args = []any{now.Format(time.RFC3339), userID, tenantID}
		}
	}
	return s.within(ctx, func(exec database.Executor) error {
		_, err := exec.ExecContext(ctx, query, args...)
		return err
	})
}

// ListDeviceSessions returns non-revoked device sessions for a user.
func (s *MobileAuthService) ListDeviceSessions(ctx context.Context, userID, currentDeviceID string) ([]DeviceSessionSummary, error) {
	tenantID, err := s.tenant(ctx)
	if err != nil {
		return nil, ErrInvalidToken
	}
	var query string
	args := []any{userID}
	if s.RequireTenantScope {
		args = append(args, tenantID)
	}
	if s.dialect == database.Postgres {
		query = `SELECT m.device_id, m.device_name, m.platform, m.created_at, m.last_used_at
			FROM mobile_device_sessions m JOIN users u ON u.id = m.user_id WHERE m.user_id = $1 AND m.revoked_at IS NULL`
		if s.RequireTenantScope {
			query += ` AND u.tenant_id = $2`
		}
		query += ` ORDER BY m.last_used_at DESC`
	} else {
		query = `SELECT m.device_id, m.device_name, m.platform, m.created_at, m.last_used_at
			FROM mobile_device_sessions m JOIN users u ON u.id = m.user_id WHERE m.user_id = ? AND m.revoked_at IS NULL`
		if s.RequireTenantScope {
			query += ` AND u.tenant_id = ?`
		}
		query += ` ORDER BY m.last_used_at DESC`
	}

	var summaries []DeviceSessionSummary
	err = s.withinQuery(ctx, func(exec database.QueryExecutor) error {
		rows, queryErr := exec.QueryContext(ctx, query, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

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
				return err
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
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return summaries, nil
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
)

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
}

func GenerateSessionToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("failed to generate random session token: %w", err)
	}
	rawToken = hex.EncodeToString(b)
	digest := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(digest[:])
	return rawToken, tokenHash, nil
}

func HashSessionToken(rawToken string) string {
	digest := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(digest[:])
}

type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
	Touch(ctx context.Context, tokenHash string, lastSeen time.Time) error
}

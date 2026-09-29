package members

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrEmailVerificationInvalid = errors.New("member email verification link is invalid or expired")

type VerificationToken struct {
	ID, MemberID, Hash   string
	ExpiresAt, CreatedAt time.Time
}

type EmailVerificationTokens interface {
	CreateEmailVerificationToken(context.Context, database.Executor, VerificationToken) error
	ConsumeEmailVerificationToken(context.Context, database.Executor, string, time.Time) error
}

type EmailVerificationService struct{ Tokens EmailVerificationTokens }

func (s EmailVerificationService) Issue(ctx context.Context, tx database.Executor, member *Member, ttl time.Duration) (string, error) {
	if s.Tokens == nil || database.RequireExecutor(tx) != nil || member == nil || strings.TrimSpace(member.ID) == "" || !auth.ValidateEmail(member.Email) || !member.Active || member.EmailVerifiedAt != nil {
		return "", ErrEmailVerificationInvalid
	}
	if ttl <= 0 || ttl > 7*24*time.Hour {
		ttl = 24 * time.Hour
	}
	secret := make([]byte, 32)
	id := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	if _, err := rand.Read(id); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(secret)
	digest := sha256.Sum256([]byte(raw))
	now := time.Now().UTC()
	err := s.Tokens.CreateEmailVerificationToken(ctx, tx, VerificationToken{ID: hex.EncodeToString(id), MemberID: member.ID, Hash: hex.EncodeToString(digest[:]), CreatedAt: now, ExpiresAt: now.Add(ttl)})
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (s EmailVerificationService) Verify(ctx context.Context, tx database.Executor, raw string) error {
	if s.Tokens == nil || database.RequireExecutor(tx) != nil || len(raw) != 64 {
		return ErrEmailVerificationInvalid
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return ErrEmailVerificationInvalid
	}
	digest := sha256.Sum256([]byte(raw))
	if err := s.Tokens.ConsumeEmailVerificationToken(ctx, tx, hex.EncodeToString(digest[:]), time.Now().UTC()); err != nil {
		return ErrEmailVerificationInvalid
	}
	return nil
}

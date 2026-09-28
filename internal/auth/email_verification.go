package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

const EmailVerificationPurpose = "email_verification"

var ErrEmailVerificationInvalid = errors.New("email verification link is invalid or expired")

type VerifiedEmailUserRepository interface {
	LockEmailVerificationUserTx(context.Context, database.Executor, string) (string, *time.Time, error)
	SetEmailVerifiedTx(context.Context, database.Executor, string, time.Time) error
}
type EmailVerificationTokenLookup interface {
	FindEmailVerificationOwnerTx(context.Context, database.Executor, string, time.Time) (string, error)
}
type EmailVerificationService struct {
	Users  VerifiedEmailUserRepository
	Tokens RecoveryStore
}

func (s EmailVerificationService) Issue(ctx context.Context, tx database.Executor, user *User, ttl time.Duration) (string, string, error) {
	if guards.IsNil(s.Users) || guards.IsNil(s.Tokens) || database.RequireExecutor(tx) != nil {
		return "", "", ErrRecoveryDependencyRequired
	}
	if user == nil || !user.Active || strings.TrimSpace(user.ID) == "" || !ValidateEmail(user.Email) {
		return "", "", ErrEmailVerificationInvalid
	}
	currentEmail, verifiedAt, err := s.Users.LockEmailVerificationUserTx(ctx, tx, user.ID)
	if err != nil || NormalizeEmail(currentEmail) != NormalizeEmail(user.Email) {
		return "", "", ErrEmailVerificationInvalid
	}
	if verifiedAt != nil {
		return currentEmail, "", nil
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		ttl = 2 * time.Hour
	}
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", "", err
	}
	raw := hex.EncodeToString(rawBytes)
	digest := sha256.Sum256([]byte(raw))
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	if err := s.Tokens.CreateRecoveryToken(ctx, tx, hex.EncodeToString(idBytes), user.ID, EmailVerificationPurpose, hex.EncodeToString(digest[:]), now.Add(ttl), now); err != nil {
		return "", "", err
	}
	return user.Email, raw, nil
}
func (s EmailVerificationService) Verify(ctx context.Context, tx database.Executor, raw string) error {
	if guards.IsNil(s.Users) || guards.IsNil(s.Tokens) || database.RequireExecutor(tx) != nil || strings.TrimSpace(raw) == "" {
		return ErrEmailVerificationInvalid
	}
	digest := sha256.Sum256([]byte(raw))
	lookup, ok := s.Tokens.(EmailVerificationTokenLookup)
	if !ok {
		return ErrEmailVerificationInvalid
	}
	tokenHash := hex.EncodeToString(digest[:])
	userID, err := lookup.FindEmailVerificationOwnerTx(ctx, tx, tokenHash, time.Now().UTC())
	if err != nil {
		return ErrEmailVerificationInvalid
	}
	if _, _, err := s.Users.LockEmailVerificationUserTx(ctx, tx, userID); err != nil {
		return ErrEmailVerificationInvalid
	}
	consumedUserID, err := s.Tokens.ConsumeRecoveryToken(ctx, tx, EmailVerificationPurpose, tokenHash, time.Now().UTC())
	if err != nil || consumedUserID != userID {
		return ErrEmailVerificationInvalid
	}
	if err := s.Users.SetEmailVerifiedTx(ctx, tx, userID, time.Now().UTC()); err != nil {
		return ErrEmailVerificationInvalid
	}
	return nil
}

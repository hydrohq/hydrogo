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

const PasswordResetPurpose = "password_reset"

var (
	ErrRecoveryTokenInvalid       = errors.New("invalid or expired recovery token")
	ErrRecoveryDependencyRequired = errors.New("password recovery dependency is required")
)

type RecoveryStore interface {
	CreateRecoveryToken(context.Context, database.Executor, string, string, string, string, time.Time, time.Time) error
	ConsumeRecoveryToken(context.Context, database.Executor, string, string, time.Time) (string, error)
}

type TransactionalSessionRevoker interface {
	DeleteByUserIDTx(context.Context, database.Executor, string) error
}

type PasswordRecovery struct {
	Users       UserRepository
	Tokens      RecoveryStore
	Sessions    SessionRepository
	ArgonParams Argon2Params
}

func (r PasswordRecovery) Issue(ctx context.Context, tx database.Executor, email string, ttl time.Duration) (string, error) {
	if guards.IsNil(r.Users) || guards.IsNil(r.Tokens) {
		return "", ErrRecoveryDependencyRequired
	}
	user, err := r.Users.GetByEmail(ctx, NormalizeEmail(email))
	if err != nil {
		return "", err
	}
	if !user.Active {
		return "", ErrUserInactive
	}
	if ttl <= 0 || ttl > time.Hour {
		ttl = 30 * time.Minute
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(bytes)
	digest := sha256.Sum256([]byte(raw))
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := r.Tokens.CreateRecoveryToken(ctx, tx, hex.EncodeToString(idBytes), user.ID, PasswordResetPurpose, hex.EncodeToString(digest[:]), now.Add(ttl), now); err != nil {
		return "", err
	}
	return raw, nil
}

func (r PasswordRecovery) Reset(ctx context.Context, tx database.Executor, rawToken, newPassword string) error {
	if guards.IsNil(r.Users) || guards.IsNil(r.Tokens) || guards.IsNil(r.Sessions) {
		return ErrRecoveryDependencyRequired
	}
	if strings.TrimSpace(rawToken) == "" || newPassword == "" {
		return ErrRecoveryTokenInvalid
	}
	digest := sha256.Sum256([]byte(rawToken))
	userID, err := r.Tokens.ConsumeRecoveryToken(ctx, tx, PasswordResetPurpose, hex.EncodeToString(digest[:]), time.Now().UTC())
	if err != nil {
		return ErrRecoveryTokenInvalid
	}
	hash, err := HashPassword(newPassword, r.ArgonParams)
	if err != nil {
		return err
	}
	if err := r.Users.UpdatePassword(ctx, tx, userID, hash); err != nil {
		return err
	}
	if revoker, ok := r.Sessions.(TransactionalSessionRevoker); ok {
		return revoker.DeleteByUserIDTx(ctx, tx, userID)
	}
	return r.Sessions.DeleteByUserID(ctx, userID)
}

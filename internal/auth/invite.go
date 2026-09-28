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

const InvitePurpose = "invite"

var (
	ErrInviteInvalid            = errors.New("invalid or expired invitation")
	ErrInviteDependencyRequired = errors.New("invitation dependency is required")
)

type InviteStore interface {
	CreateInvite(context.Context, database.Executor, string, string, Role, string, time.Time, time.Time) error
	ConsumeInvite(context.Context, database.Executor, string, time.Time) (string, Role, error)
}

type TransactionalUserCreator interface {
	CreateUserTx(context.Context, database.Executor, *User) error
}

type InvitationService struct {
	Users       UserRepository
	Invites     InviteStore
	ArgonParams Argon2Params
}

func (s InvitationService) Issue(ctx context.Context, db database.Executor, email string, role Role, ttl time.Duration) (string, error) {
	if guards.IsNil(s.Users) || guards.IsNil(s.Invites) {
		return "", ErrInviteDependencyRequired
	}
	if err := database.RequireExecutor(db); err != nil {
		return "", err
	}
	email = NormalizeEmail(email)
	if !ValidateEmail(email) || (role != RoleEditor && role != RoleViewer) {
		return "", ErrInviteInvalid
	}
	if user, err := s.Users.GetByEmail(ctx, email); err == nil && user != nil {
		return "", ErrUserAlreadyExists
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return "", err
	}
	if ttl <= 0 || ttl > 7*24*time.Hour {
		ttl = 24 * time.Hour
	}
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(rawBytes)
	digest := sha256.Sum256([]byte(raw))
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := s.Invites.CreateInvite(ctx, db, hex.EncodeToString(idBytes), email, role, hex.EncodeToString(digest[:]), now.Add(ttl), now); err != nil {
		return "", err
	}
	return raw, nil
}

func (s InvitationService) Accept(ctx context.Context, tx database.Executor, rawToken, password string) (*User, error) {
	if guards.IsNil(s.Users) || guards.IsNil(s.Invites) {
		return nil, ErrInviteDependencyRequired
	}
	if err := database.RequireExecutor(tx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(rawToken) == "" || password == "" {
		return nil, ErrInviteInvalid
	}
	digest := sha256.Sum256([]byte(rawToken))
	email, role, err := s.Invites.ConsumeInvite(ctx, tx, hex.EncodeToString(digest[:]), time.Now().UTC())
	if err != nil {
		return nil, ErrInviteInvalid
	}
	hash, err := HashPassword(password, s.ArgonParams)
	if err != nil {
		return nil, err
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &User{ID: hex.EncodeToString(idBytes), Email: email, PasswordHash: hash, Role: role, Active: true, CreatedAt: now, UpdatedAt: now, EmailVerifiedAt: &now}
	creator, ok := s.Users.(TransactionalUserCreator)
	if !ok {
		return nil, errors.New("transactional user creation is unavailable")
	}
	if err := creator.CreateUserTx(ctx, tx, user); err != nil {
		return nil, err
	}
	return user, nil
}

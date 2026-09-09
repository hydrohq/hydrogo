package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

const (
	SessionCookieName = "hydro_session"
	DefaultSessionTTL = 24 * time.Hour * 7
	DefaultIdleTTL    = 24 * time.Hour
)

var (
	ErrInvalidCredentials        = errors.New("invalid email or password")
	ErrUserInactive              = errors.New("user account is inactive")
	ErrUserRepositoryRequired    = errors.New("user repository is required")
	ErrSessionRepositoryRequired = errors.New("session repository is required")
)

type contextKey string

const (
	userCtxKey    contextKey = "hydro_user"
	sessionCtxKey contextKey = "hydro_session"
)

type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	argonParams Argon2Params
	dummyHash   string
	idleTTL     time.Duration
}

func NewAuthService(users UserRepository, sessions SessionRepository, params Argon2Params) *AuthService {
	dummyParams := params
	if err := validateArgon2Params(params); err != nil {
		// Keep authentication's timing defense usable even if an embedding
		// application passes an invalid tuning profile. Keep the original
		// profile so password creation still rejects the bad configuration.
		dummyParams = DefaultArgon2Params
	}
	// Keep the unknown-user verification at the same Argon2 cost as real users.
	dummyHash, _ := HashPassword("hydrogo-dummy-password", dummyParams)
	return &AuthService{
		userRepo:    users,
		sessionRepo: sessions,
		argonParams: params,
		dummyHash:   dummyHash,
		idleTTL:     DefaultIdleTTL,
	}
}

func (s *AuthService) CreateUser(ctx context.Context, email, password string, role Role) (*User, error) {
	if s == nil || guards.IsNil(s.userRepo) {
		return nil, ErrUserRepositoryRequired
	}
	normEmail := NormalizeEmail(email)
	if existing, _ := s.userRepo.GetByEmail(ctx, normEmail); existing != nil {
		return nil, ErrUserAlreadyExists
	}

	hash, err := HashPassword(password, s.argonParams)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(idBytes)

	now := time.Now().UTC()
	u := &User{
		ID:           id,
		Email:        normEmail,
		PasswordHash: hash,
		Role:         role,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AuthService) Authenticate(ctx context.Context, email, password string) (*User, error) {
	if s == nil || guards.IsNil(s.userRepo) {
		return nil, ErrUserRepositoryRequired
	}
	normEmail := NormalizeEmail(email)
	u, err := s.userRepo.GetByEmail(ctx, normEmail)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Fake verify to mitigate timing attacks
			_, _ = VerifyPassword(password, s.dummyHash)
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !u.Active {
		return nil, ErrUserInactive
	}

	valid, err := VerifyPassword(password, u.PasswordHash)
	if err != nil || !valid {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

func (s *AuthService) CreateSession(ctx context.Context, userID, userAgent, ipAddress string, ttl time.Duration) (string, *Session, error) {
	if s == nil || guards.IsNil(s.sessionRepo) {
		return "", nil, ErrSessionRepositoryRequired
	}
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}

	rawToken, tokenHash, err := GenerateSessionToken()
	if err != nil {
		return "", nil, err
	}

	sessIDBytes := make([]byte, 16)
	if _, err := rand.Read(sessIDBytes); err != nil {
		return "", nil, err
	}
	sessID := hex.EncodeToString(sessIDBytes)

	now := time.Now().UTC()
	sess := &Session{
		ID:         sessID,
		UserID:     userID,
		TokenHash:  tokenHash,
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
		LastSeenAt: now,
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
	}

	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return "", nil, err
	}

	return rawToken, sess, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, rawToken string) (*Session, *User, error) {
	if s == nil || guards.IsNil(s.sessionRepo) {
		return nil, nil, ErrSessionRepositoryRequired
	}
	if guards.IsNil(s.userRepo) {
		return nil, nil, ErrUserRepositoryRequired
	}
	tokenHash := HashSessionToken(rawToken)
	sess, err := s.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	if sess == nil || !sess.ExpiresAt.After(now) {
		_ = s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, nil, ErrSessionExpired
	}

	u, err := s.userRepo.GetByID(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if !u.Active {
		return nil, nil, ErrUserInactive
	}
	if s.idleTTL > 0 && now.Sub(sess.LastSeenAt) > s.idleTTL {
		_ = s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, nil, ErrSessionExpired
	}

	// Update last seen periodically (fire and forget / non-blocking)
	if now.Sub(sess.LastSeenAt) > 5*time.Minute {
		_ = s.sessionRepo.Touch(ctx, tokenHash, now)
	}

	return sess, u, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, rawToken string) error {
	if s == nil || guards.IsNil(s.sessionRepo) {
		return ErrSessionRepositoryRequired
	}
	tokenHash := HashSessionToken(rawToken)
	return s.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
}

func (s *AuthService) RevokeAllUserSessions(ctx context.Context, userID string) error {
	if s == nil || guards.IsNil(s.sessionRepo) {
		return ErrSessionRepositoryRequired
	}
	return s.sessionRepo.DeleteByUserID(ctx, userID)
}

func (s *AuthService) SetUserActive(ctx context.Context, userID string, active bool) error {
	if s == nil || guards.IsNil(s.userRepo) {
		return ErrUserRepositoryRequired
	}
	if !active && guards.IsNil(s.sessionRepo) {
		return ErrSessionRepositoryRequired
	}
	if err := s.userRepo.SetActive(ctx, userID, active); err != nil {
		return err
	}
	if !active {
		return s.RevokeAllUserSessions(ctx, userID)
	}
	return nil
}

func SetSessionCookie(w http.ResponseWriter, rawToken string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionCtxKey, s)
}

func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userCtxKey).(*User)
	return u
}

func SessionFromContext(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionCtxKey).(*Session)
	return s
}

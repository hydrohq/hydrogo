package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

const (
	SessionCookieName = "hydro_session"
	DefaultSessionTTL = 24 * time.Hour * 7
	DefaultIdleTTL    = 24 * time.Hour
)

var (
	ErrInvalidCredentials           = errors.New("invalid email or password")
	ErrUserInactive                 = errors.New("user account is inactive")
	ErrUserRepositoryRequired       = errors.New("user repository is required")
	ErrSessionRepositoryRequired    = errors.New("session repository is required")
	ErrMobileSessionRevokerRequired = errors.New("mobile session revoker is required")
	ErrCurrentPasswordRequired      = errors.New("current password is required")
	ErrPasswordMismatch             = errors.New("new passwords do not match")
	ErrLastAdmin                    = errors.New("cannot remove the last active administrator")
)

type contextKey string

const (
	userCtxKey    contextKey = "hydro_user"
	sessionCtxKey contextKey = "hydro_session"
)

type AuthService struct {
	userRepo          UserRepository
	sessionRepo       SessionRepository
	mobileSessions    MobileSessionRevoker
	argonParams       Argon2Params
	dummyHash         string
	idleTTL           time.Duration
	permissions       PermissionStore
	permissionCatalog *PermissionCatalog
}

func (s *AuthService) SetPermissionStore(store PermissionStore) {
	if s != nil {
		s.permissions = store
	}
}

// SetPermissionCatalog installs the active Edition's explicit permission
// allowlist. Call during composition before serving requests.
func (s *AuthService) SetPermissionCatalog(catalog *PermissionCatalog) error {
	if s == nil || catalog == nil {
		return errors.New("permission catalog is required")
	}
	s.permissionCatalog = catalog
	return nil
}

// AvailablePermissions returns the deterministic names exposed to the
// authenticated Admin Studio permission editor.
func (s *AuthService) AvailablePermissions() []string {
	if s != nil && s.permissionCatalog != nil {
		return s.permissionCatalog.Names()
	}
	catalog, _ := NewPermissionCatalog()
	return catalog.Names()
}

// Can checks an actor against this service's active Edition permission
// catalog. Custom Edition policies should prefer this over checking names
// from untrusted request input.
func (s *AuthService) Can(user *User, permission string) bool {
	if s == nil {
		return false
	}
	catalog := s.permissionCatalog
	if catalog == nil {
		catalog, _ = NewPermissionCatalog()
	}
	return catalog.Allows(user, permission)
}

// MobileSessionRevoker keeps bearer credentials on the same disable lifecycle
// as browser sessions. The implementation deliberately exposes no token data.
type MobileSessionRevoker interface {
	RevokeAllDevices(ctx context.Context, userID string) error
}

func NewAuthService(users UserRepository, sessions SessionRepository, mobileSessions MobileSessionRevoker, params Argon2Params) *AuthService {
	permissionCatalog, _ := NewPermissionCatalog()
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
		userRepo:          users,
		sessionRepo:       sessions,
		mobileSessions:    mobileSessions,
		argonParams:       params,
		dummyHash:         dummyHash,
		idleTTL:           DefaultIdleTTL,
		permissionCatalog: permissionCatalog,
	}
}

func (s *AuthService) CreateUser(ctx context.Context, email, password string, role Role) (*User, error) {
	if s == nil || guards.IsNil(s.userRepo) {
		return nil, ErrUserRepositoryRequired
	}
	normEmail := NormalizeEmail(email)
	if !ValidateEmail(normEmail) {
		return nil, errors.New("invalid email address")
	}
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
	if s.permissions != nil {
		overrides, err := s.permissions.LoadPermissionOverrides(ctx, u.ID)
		if err != nil {
			return nil, nil, err
		}
		activeOverrides := make(map[string]bool, len(overrides))
		for permission, allowed := range overrides {
			if s.permissionKnown(permission) {
				activeOverrides[permission] = allowed
			}
		}
		u.PermissionOverrides = activeOverrides
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

type selectiveSessionRevoker interface {
	DeleteOtherByUserID(context.Context, string, string) error
}

type sessionLister interface {
	ListByUserID(context.Context, string, time.Time) ([]Session, error)
}

func (s *AuthService) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	if s == nil || guards.IsNil(s.sessionRepo) {
		return nil, ErrSessionRepositoryRequired
	}
	lister, ok := s.sessionRepo.(sessionLister)
	if !ok {
		return nil, errors.New("session repository cannot list sessions")
	}
	return lister.ListByUserID(ctx, userID, time.Now().UTC())
}

func (s *AuthService) RevokeOtherUserSessions(ctx context.Context, userID, currentTokenHash string) error {
	if s == nil || guards.IsNil(s.sessionRepo) {
		return ErrSessionRepositoryRequired
	}
	revoker, ok := s.sessionRepo.(selectiveSessionRevoker)
	if !ok {
		return errors.New("session repository cannot retain the current session")
	}
	return revoker.DeleteOtherByUserID(ctx, userID, currentTokenHash)
}

// ChangeCredentials verifies the current password before changing either
// account credential. The caller must revoke the browser session afterwards;
// this service method keeps hashing and persistence in one place.
func (s *AuthService) ChangeCredentials(ctx context.Context, userID, currentPassword, email, newPassword string, tx database.Executor) error {
	if s == nil || guards.IsNil(s.userRepo) {
		return ErrUserRepositoryRequired
	}
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	valid, err := VerifyPassword(currentPassword, u.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidCredentials
	}
	if email == "" && newPassword == "" {
		return ErrCurrentPasswordRequired
	}
	if email != "" {
		email = NormalizeEmail(email)
		if !ValidateEmail(email) {
			return errors.New("invalid email address")
		}
		if existing, lookupErr := s.userRepo.GetByEmail(ctx, email); lookupErr == nil && existing.ID != userID {
			return ErrUserAlreadyExists
		}
		if err := s.userRepo.UpdateEmail(ctx, tx, userID, email); err != nil {
			return err
		}
	}
	if newPassword != "" {
		hash, err := HashPassword(newPassword, s.argonParams)
		if err != nil {
			return err
		}
		if err := s.userRepo.UpdatePassword(ctx, tx, userID, hash); err != nil {
			return err
		}
	}
	return nil
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
		if err := s.RevokeAllUserSessions(ctx, userID); err != nil {
			return err
		}
		if s.mobileSessions == nil {
			return ErrMobileSessionRevokerRequired
		}
		return s.mobileSessions.RevokeAllDevices(ctx, userID)
	}
	return nil
}

func (s *AuthService) UpdateUserRole(ctx context.Context, userID string, role Role) error {
	if s == nil || guards.IsNil(s.userRepo) {
		return ErrUserRepositoryRequired
	}
	repo, ok := s.userRepo.(UserAdminRepository)
	if !ok {
		return ErrUserRepositoryRequired
	}
	users, err := repo.List(ctx)
	if err != nil {
		return err
	}
	activeAdmins := 0
	targetWasAdmin := false
	for _, user := range users {
		if user.Active && user.Role == RoleAdmin {
			activeAdmins++
		}
		if user.ID == userID && user.Role == RoleAdmin {
			targetWasAdmin = true
		}
	}
	if targetWasAdmin && role != RoleAdmin && activeAdmins <= 1 {
		return ErrLastAdmin
	}
	if err := repo.UpdateRole(ctx, userID, role); err != nil {
		return err
	}
	return s.RevokeAllUserSessions(ctx, userID)
}

func (s *AuthService) ReplaceUserPermissions(ctx context.Context, userID string, overrides map[string]bool) error {
	if s == nil || s.permissions == nil {
		return ErrUserRepositoryRequired
	}
	for permission := range overrides {
		if !s.permissionKnown(permission) {
			return errors.New("invalid permission")
		}
	}
	if err := s.permissions.ReplacePermissionOverrides(ctx, userID, overrides); err != nil {
		return err
	}
	return s.RevokeAllUserSessions(ctx, userID)
}

func (s *AuthService) permissionKnown(permission string) bool {
	if s != nil && s.permissionCatalog != nil {
		return s.permissionCatalog.Contains(permission)
	}
	for _, known := range KnownPermissions {
		if permission == known {
			return true
		}
	}
	return false
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
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userCtxKey, u)
}

func WithSession(ctx context.Context, s *Session) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionCtxKey, s)
}

func UserFromContext(ctx context.Context) *User {
	if ctx == nil {
		return nil
	}
	u, _ := ctx.Value(userCtxKey).(*User)
	return u
}

func SessionFromContext(ctx context.Context) *Session {
	if ctx == nil {
		return nil
	}
	s, _ := ctx.Value(sessionCtxKey).(*Session)
	return s
}

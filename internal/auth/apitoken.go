package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

// APITokenPrefix opens every plaintext token so clients and middleware can
// tell a HydroGo credential from other bearer schemes.
const APITokenPrefix = "hyd_"

const apiTokenPrefixLength = 8

type APITokenKind string

const (
	APITokenPersonal APITokenKind = "personal"
	APITokenKey      APITokenKind = "api_key"
)

// Token scopes. A personal token always carries ScopeAdmin and acts as its
// owner; API keys are read-only and act at most as a viewer (content and
// commerce reads), whatever their owner can do. ScopeRead and ScopePreview are
// currently equivalent: no handler distinguishes drafts by key scope yet.
const (
	ScopeAdmin   = "admin"
	ScopeRead    = "read"
	ScopePreview = "preview"
)

var (
	ErrAPITokenNotFound      = errors.New("api token not found")
	ErrAPITokenInvalid       = errors.New("api token is invalid")
	ErrAPITokenStoreRequired = errors.New("api token store is required")
	ErrAPITokenScope         = errors.New("api token scope is not allowed")
)

type APIToken struct {
	ID         string       `json:"id"`
	UserID     string       `json:"user_id"`
	Name       string       `json:"name"`
	Kind       APITokenKind `json:"kind"`
	Prefix     string       `json:"prefix"`
	Scopes     []string     `json:"scopes"`
	TokenHash  string       `json:"-"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty"`
	LastUsedAt *time.Time   `json:"last_used_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	RevokedAt  *time.Time   `json:"revoked_at,omitempty"`
}

// HasScope reports whether the token carries the named scope.
func (t APIToken) HasScope(scope string) bool {
	for _, candidate := range t.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

// APITokenStore persists hashed tokens. Implementations scope rows by tenant
// exactly like SessionRepository.
type APITokenStore interface {
	Create(ctx context.Context, token *APIToken) error
	ListForUser(ctx context.Context, userID string) ([]APIToken, error)
	ListAPIKeys(ctx context.Context) ([]APIToken, error)
	// Revoke marks the token revoked. A non-empty userID restricts the
	// operation to that owner; an empty one revokes regardless of owner.
	Revoke(ctx context.Context, id, userID string) error
	FindByHash(ctx context.Context, tokenHash string) (*APIToken, error)
	TouchLastUsed(ctx context.Context, id string, at time.Time) error
}

// GenerateAPIToken returns a plaintext token, its storage hash and prefix.
func GenerateAPIToken() (plaintext, tokenHash, prefix string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	plaintext = APITokenPrefix + secret
	return plaintext, HashAPIToken(plaintext), secret[:apiTokenPrefixLength], nil
}

func HashAPIToken(plaintext string) string {
	digest := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(digest[:])
}

// IsAPIToken reports whether a credential string looks like a HydroGo token.
func IsAPIToken(credential string) bool {
	return strings.HasPrefix(credential, APITokenPrefix) && len(credential) > len(APITokenPrefix)+apiTokenPrefixLength
}

// APITokenFromRequest extracts a HydroGo token from "Authorization: Bearer"
// or "X-API-Key". The second result is true when either header carried a
// HydroGo credential, valid or not, so callers can refuse to fall back to
// cookies.
func APITokenFromRequest(r *http.Request) (string, bool) {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key, IsAPIToken(key)
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && IsAPIToken(parts[1]) {
		return parts[1], true
	}
	return "", false
}

// SetAPITokenStore enables bearer-token authentication for the service.
func (s *AuthService) SetAPITokenStore(store APITokenStore) {
	if s != nil {
		s.apiTokens = store
	}
}

// IssueAPIToken creates a token for userID and returns the plaintext once.
// Personal tokens always carry ScopeAdmin; keys must be read or preview.
func (s *AuthService) IssueAPIToken(ctx context.Context, userID, name string, kind APITokenKind, scopes []string, expiresAt *time.Time) (string, *APIToken, error) {
	if s == nil || guards.IsNil(s.apiTokens) {
		return "", nil, ErrAPITokenStoreRequired
	}
	name = strings.TrimSpace(name)
	if userID == "" || name == "" || len(name) > 120 {
		return "", nil, ErrAPITokenInvalid
	}
	switch kind {
	case APITokenPersonal:
		scopes = []string{ScopeAdmin}
	case APITokenKey:
		if len(scopes) != 1 || (scopes[0] != ScopeRead && scopes[0] != ScopePreview) {
			return "", nil, ErrAPITokenScope
		}
	default:
		return "", nil, ErrAPITokenInvalid
	}
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		return "", nil, ErrAPITokenInvalid
	}
	plaintext, tokenHash, prefix, err := GenerateAPIToken()
	if err != nil {
		return "", nil, err
	}
	token := &APIToken{ID: newAPITokenID(), UserID: userID, Name: name, Kind: kind, Prefix: prefix, Scopes: scopes, TokenHash: tokenHash, ExpiresAt: expiresAt, CreatedAt: time.Now().UTC()}
	if err := s.apiTokens.Create(ctx, token); err != nil {
		return "", nil, err
	}
	return plaintext, token, nil
}

func (s *AuthService) ListAPITokens(ctx context.Context, userID string) ([]APIToken, error) {
	if s == nil || guards.IsNil(s.apiTokens) {
		return nil, ErrAPITokenStoreRequired
	}
	return s.apiTokens.ListForUser(ctx, userID)
}

func (s *AuthService) ListAPIKeys(ctx context.Context) ([]APIToken, error) {
	if s == nil || guards.IsNil(s.apiTokens) {
		return nil, ErrAPITokenStoreRequired
	}
	return s.apiTokens.ListAPIKeys(ctx)
}

// RevokeAPIToken revokes id; userID, when set, must own the token.
func (s *AuthService) RevokeAPIToken(ctx context.Context, id, userID string) error {
	if s == nil || guards.IsNil(s.apiTokens) {
		return ErrAPITokenStoreRequired
	}
	return s.apiTokens.Revoke(ctx, id, userID)
}

// ValidateAPIToken resolves a plaintext token to its active owner.
func (s *AuthService) ValidateAPIToken(ctx context.Context, plaintext string) (*User, *APIToken, error) {
	if s == nil || guards.IsNil(s.apiTokens) {
		return nil, nil, ErrAPITokenStoreRequired
	}
	if guards.IsNil(s.userRepo) {
		return nil, nil, ErrUserRepositoryRequired
	}
	if !IsAPIToken(plaintext) {
		return nil, nil, ErrAPITokenInvalid
	}
	token, err := s.apiTokens.FindByHash(ctx, HashAPIToken(plaintext))
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	if token == nil || token.RevokedAt != nil || (token.ExpiresAt != nil && !token.ExpiresAt.After(now)) {
		return nil, nil, ErrAPITokenInvalid
	}
	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil || !user.Active {
		return nil, nil, ErrUserInactive
	}
	if err := s.attachPermissionOverrides(ctx, user); err != nil {
		return nil, nil, err
	}
	if token.LastUsedAt == nil || now.Sub(*token.LastUsedAt) > 5*time.Minute {
		_ = s.apiTokens.TouchLastUsed(ctx, token.ID, now)
	}
	return user, token, nil
}

func newAPITokenID() string {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

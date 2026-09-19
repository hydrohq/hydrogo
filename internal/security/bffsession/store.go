// Package bffsession contains a bounded reference session adapter for a web
// BFF. It stores the Core session server-side and exposes only an opaque BFF
// cookie to the browser. Deployments with more than one BFF instance must
// replace Store with a shared implementation behind the same interfaces.
package bffsession

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidConfig   = errors.New("invalid bff session configuration")
	ErrUnauthenticated = errors.New("bff session is not authenticated")
	ErrCapacity        = errors.New("bff session capacity is exhausted")
	ErrInvalidCSRF     = errors.New("invalid bff csrf token")
)

type Config struct {
	CookieName string
	TTL        time.Duration
	MaxEntries int
	Secure     bool
	SameSite   http.SameSite
}

type record struct {
	coreCookie string
	csrfHash   [sha256.Size]byte
	expiresAt  time.Time
}

type Store struct {
	mu       sync.Mutex
	config   Config
	sessions map[string]record
	now      func() time.Time
}

// New validates the explicit cookie/security policy. Secure=false is allowed
// only for local HTTP fixtures; production callers should require Secure=true.
func New(config Config) (*Store, error) {
	if !validCookieName(config.CookieName) || config.TTL <= 0 || config.TTL > 30*24*time.Hour || config.MaxEntries <= 0 || config.MaxEntries > 1_000_000 {
		return nil, ErrInvalidConfig
	}
	if config.SameSite == http.SameSiteDefaultMode {
		config.SameSite = http.SameSiteLaxMode
	}
	return &Store{config: config, sessions: make(map[string]record, config.MaxEntries), now: time.Now}, nil
}

func validCookieName(name string) bool {
	if name == "" || strings.TrimSpace(name) != name {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~', ':':
		default:
			return false
		}
	}
	return true
}

// Create stores a server-side Core cookie and sends a BFF-only cookie to the
// browser. The returned CSRF token must be rendered into the BFF frontend's
// protected form/bootstrap response; it is never sent to Core automatically.
func (s *Store) Create(w http.ResponseWriter, coreCookie string) (string, error) {
	if s == nil || strings.TrimSpace(coreCookie) == "" {
		return "", ErrUnauthenticated
	}
	now := s.now()
	s.mu.Lock()
	s.pruneLocked(now)
	if len(s.sessions) >= s.config.MaxEntries {
		s.mu.Unlock()
		return "", ErrCapacity
	}
	sessionID, err := randomToken()
	if err != nil {
		s.mu.Unlock()
		return "", err
	}
	csrfToken, err := randomToken()
	if err != nil {
		s.mu.Unlock()
		return "", err
	}
	hash := sha256.Sum256([]byte(csrfToken))
	s.sessions[sessionID] = record{coreCookie: coreCookie, csrfHash: hash, expiresAt: now.Add(s.config.TTL)}
	s.mu.Unlock()
	if w != nil {
		http.SetCookie(w, &http.Cookie{Name: s.config.CookieName, Value: sessionID, Path: "/", HttpOnly: true, Secure: s.config.Secure, SameSite: s.config.SameSite, MaxAge: int(s.config.TTL / time.Second)})
	}
	return csrfToken, nil
}

func (s *Store) CoreCookie(_ context.Context, r *http.Request) (string, error) {
	record, ok := s.lookup(r)
	if !ok {
		return "", ErrUnauthenticated
	}
	return record.coreCookie, nil
}

// Validate implements the BFF CSRFValidator contract. The token is accepted
// from the explicit header only; query/form tokens are intentionally ignored.
func (s *Store) Validate(_ context.Context, r *http.Request) error {
	record, ok := s.lookup(r)
	if !ok {
		return ErrUnauthenticated
	}
	provided := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if provided == "" {
		return ErrInvalidCSRF
	}
	hash := sha256.Sum256([]byte(provided))
	if subtle.ConstantTimeCompare(hash[:], record.csrfHash[:]) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

func (s *Store) Revoke(w http.ResponseWriter, r *http.Request) {
	if s == nil || r == nil {
		return
	}
	if cookie, err := r.Cookie(s.config.CookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	if w != nil {
		http.SetCookie(w, &http.Cookie{Name: s.config.CookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.config.Secure, SameSite: s.config.SameSite, MaxAge: -1})
	}
}

func (s *Store) lookup(r *http.Request) (record, bool) {
	if s == nil || r == nil {
		return record{}, false
	}
	cookie, err := r.Cookie(s.config.CookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return record{}, false
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.sessions[cookie.Value]
	if !ok || !now.Before(item.expiresAt) {
		if ok {
			delete(s.sessions, cookie.Value)
		}
		return record{}, false
	}
	return item, true
}

func (s *Store) pruneLocked(now time.Time) {
	for key, item := range s.sessions {
		if !now.Before(item.expiresAt) {
			delete(s.sessions, key)
		}
	}
}

func randomToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

package bffsession

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidRedisConfig = errors.New("invalid redis bff session configuration")
	ErrCorruptSession     = errors.New("corrupt bff session")
)

type RedisConfig struct {
	CookieName string
	Prefix     string
	TTL        time.Duration
	Secure     bool
	SameSite   http.SameSite
}

// RedisStore is the shared-session implementation for a multi-instance BFF.
// Session values are encrypted before Redis storage; Redis availability or key
// decryption failure always fails closed. Capacity, eviction and persistence
// remain explicit Redis/Valkey operator settings rather than process memory.
type RedisStore struct {
	client redis.UniversalClient
	key    []byte
	config RedisConfig
}

var _ SessionStore = (*RedisStore)(nil)

type redisRecord struct {
	CoreCookie string            `json:"core_cookie"`
	CSRFHash   [sha256.Size]byte `json:"csrf_hash"`
	ExpiresAt  time.Time         `json:"expires_at"`
}

func NewRedisStore(client redis.UniversalClient, key []byte, config RedisConfig) (*RedisStore, error) {
	if client == nil || len(key) != 32 || !validCookieName(config.CookieName) || strings.TrimSpace(config.Prefix) == "" || config.TTL <= 0 || config.TTL > 30*24*time.Hour {
		return nil, ErrInvalidRedisConfig
	}
	if config.SameSite == http.SameSiteDefaultMode {
		config.SameSite = http.SameSiteLaxMode
	}
	copyKey := append([]byte(nil), key...)
	return &RedisStore{client: client, key: copyKey, config: config}, nil
}

func (s *RedisStore) Create(ctx context.Context, w http.ResponseWriter, coreCookie string) (string, error) {
	if s == nil || strings.TrimSpace(coreCookie) == "" {
		return "", ErrUnauthenticated
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sessionID, err := randomToken()
	if err != nil {
		return "", err
	}
	csrfToken, err := randomToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	payload, err := s.seal(sessionID, redisRecord{CoreCookie: coreCookie, CSRFHash: sha256.Sum256([]byte(csrfToken)), ExpiresAt: now.Add(s.config.TTL)})
	if err != nil {
		return "", err
	}
	operationCtx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()
	if err := s.client.Set(operationCtx, s.config.Prefix+sessionID, payload, s.config.TTL).Err(); err != nil {
		return "", err
	}
	if w != nil {
		http.SetCookie(w, &http.Cookie{Name: s.config.CookieName, Value: sessionID, Path: "/", HttpOnly: true, Secure: s.config.Secure, SameSite: s.config.SameSite, MaxAge: int(s.config.TTL / time.Second)})
	}
	return csrfToken, nil
}

func (s *RedisStore) CoreCookie(ctx context.Context, r *http.Request) (string, error) {
	record, sessionID, ok := s.lookup(ctx, r)
	if !ok {
		return "", ErrUnauthenticated
	}
	_ = sessionID
	return record.CoreCookie, nil
}

func (s *RedisStore) Validate(ctx context.Context, r *http.Request) error {
	record, _, ok := s.lookup(ctx, r)
	if !ok {
		return ErrUnauthenticated
	}
	provided := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if provided == "" {
		return ErrInvalidCSRF
	}
	hash := sha256.Sum256([]byte(provided))
	if subtle.ConstantTimeCompare(hash[:], record.CSRFHash[:]) != 1 {
		return ErrInvalidCSRF
	}
	return nil
}

func (s *RedisStore) Revoke(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if s == nil || r == nil {
		return
	}
	if cookie, err := r.Cookie(s.config.CookieName); err == nil {
		operationCtx, cancel := context.WithTimeout(ctxOrBackground(ctx), 250*time.Millisecond)
		_ = s.client.Del(operationCtx, s.config.Prefix+cookie.Value).Err()
		cancel()
	}
	if w != nil {
		http.SetCookie(w, &http.Cookie{Name: s.config.CookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.config.Secure, SameSite: s.config.SameSite, MaxAge: -1})
	}
}

func (s *RedisStore) lookup(ctx context.Context, r *http.Request) (redisRecord, string, bool) {
	if s == nil || r == nil {
		return redisRecord{}, "", false
	}
	cookie, err := r.Cookie(s.config.CookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return redisRecord{}, "", false
	}
	operationCtx, cancel := context.WithTimeout(ctxOrBackground(ctx), 250*time.Millisecond)
	defer cancel()
	payload, err := s.client.Get(operationCtx, s.config.Prefix+cookie.Value).Bytes()
	if err != nil {
		return redisRecord{}, "", false
	}
	record, err := s.open(cookie.Value, payload)
	if err != nil || !time.Now().Before(record.ExpiresAt) || strings.TrimSpace(record.CoreCookie) == "" {
		if err == nil {
			_ = s.client.Del(operationCtx, s.config.Prefix+cookie.Value).Err()
		}
		return redisRecord{}, "", false
	}
	return record, cookie.Value, true
}

func (s *RedisStore) seal(sessionID string, record redisRecord) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := aead.Seal(nil, nonce, plain, []byte(sessionID))
	return append(nonce, sealed...), nil
}

func (s *RedisStore) open(sessionID string, payload []byte) (redisRecord, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return redisRecord{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(payload) < aead.NonceSize() {
		return redisRecord{}, ErrCorruptSession
	}
	plain, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], []byte(sessionID))
	if err != nil {
		return redisRecord{}, ErrCorruptSession
	}
	var record redisRecord
	if err := json.Unmarshal(plain, &record); err != nil {
		return redisRecord{}, ErrCorruptSession
	}
	return record, nil
}

func ctxOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

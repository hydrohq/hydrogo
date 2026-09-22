//go:build !hydro_no_redis

package bffsession

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisStoreEncryptsAndBindsSessionPayload(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer client.Close()
	key := []byte("01234567890123456789012345678901")
	store, err := NewRedisStore(client, key, RedisConfig{CookieName: "bff_sid", Prefix: "hydro:bff:", TTL: time.Minute, SameSite: http.SameSiteLaxMode})
	if err != nil {
		t.Fatal(err)
	}
	record := redisRecord{CoreCookie: "hydro_session=secret", CSRFHash: sha256.Sum256([]byte("csrf")), ExpiresAt: time.Now().Add(time.Minute)}
	payload, err := store.seal("sid-1", record)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) == record.CoreCookie || len(payload) < 32 {
		t.Fatal("payload is not encrypted")
	}
	decoded, err := store.open("sid-1", payload)
	if err != nil || decoded.CoreCookie != record.CoreCookie || decoded.CSRFHash != record.CSRFHash {
		t.Fatalf("decoded = %#v, err = %v", decoded, err)
	}
	if _, err := store.open("other-session", payload); err == nil {
		t.Fatal("session-bound ciphertext opened with another id")
	}
}

func TestNewRedisStoreRejectsUnsafeConfig(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer client.Close()
	for _, tc := range []struct {
		name string
		key  []byte
		cfg  RedisConfig
	}{
		{"short key", []byte("short"), RedisConfig{CookieName: "bff_sid", Prefix: "hydro:", TTL: time.Minute}},
		{"empty prefix", []byte("01234567890123456789012345678901"), RedisConfig{CookieName: "bff_sid", TTL: time.Minute}},
		{"empty cookie", []byte("01234567890123456789012345678901"), RedisConfig{Prefix: "hydro:", TTL: time.Minute}},
		{"zero ttl", []byte("01234567890123456789012345678901"), RedisConfig{CookieName: "bff_sid", Prefix: "hydro:"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewRedisStore(client, tc.key, tc.cfg); err != ErrInvalidRedisConfig {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRedisStoreLifecycleWhenConfigured(t *testing.T) {
	rawURL := strings.TrimSpace(os.Getenv("HYDRO_TEST_REDIS_URL"))
	if rawURL == "" {
		t.Skip("HYDRO_TEST_REDIS_URL is not configured")
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(options)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("configured Redis unavailable: %v", err)
	}
	prefix := "hydro:test:bff:" + strings.ReplaceAll(time.Now().UTC().Format("150405.000"), ".", "") + ":"
	store, err := NewRedisStore(client, []byte("01234567890123456789012345678901"), RedisConfig{CookieName: "bff_sid", Prefix: prefix, TTL: time.Minute, Secure: true})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	csrf, err := store.Create(ctx, response, "hydro_session=redis-core")
	if err != nil {
		t.Fatal(err)
	}
	setCookie := response.Header().Get("Set-Cookie")
	cookiePair := setCookie[:strings.Index(setCookie, ";")]
	request := httptest.NewRequest(http.MethodPost, "https://frontend.example/api/v1/member", nil)
	request.Header.Set("Cookie", cookiePair)
	request.Header.Set("X-CSRF-Token", csrf)
	coreCookie, err := store.CoreCookie(ctx, request)
	if err != nil || coreCookie != "hydro_session=redis-core" {
		t.Fatalf("core cookie=%q err=%v", coreCookie, err)
	}
	if err := store.Validate(ctx, request); err != nil {
		t.Fatal(err)
	}
	store.Revoke(ctx, nil, request)
	if _, err := store.CoreCookie(ctx, request); err == nil {
		t.Fatal("revoked Redis session accepted")
	}
}

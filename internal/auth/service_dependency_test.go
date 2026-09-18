package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAuthServiceRejectsMissingRepositories(t *testing.T) {
	ctx := context.Background()
	service := NewAuthService(nil, nil, nil, TestArgon2Params)
	if _, err := service.Authenticate(ctx, "user@example.test", "password"); !errors.Is(err, ErrUserRepositoryRequired) {
		t.Fatalf("authenticate error = %v", err)
	}
	if _, _, err := service.CreateSession(ctx, "user", "", "", 0); !errors.Is(err, ErrSessionRepositoryRequired) {
		t.Fatalf("create session error = %v", err)
	}
	if err := service.RevokeSession(ctx, "token"); !errors.Is(err, ErrSessionRepositoryRequired) {
		t.Fatalf("revoke session error = %v", err)
	}
}

func TestAuthServiceUsesSafeDummyHashForInvalidParameters(t *testing.T) {
	service := NewAuthService(nil, nil, nil, Argon2Params{})
	if service.dummyHash == "" {
		t.Fatal("invalid Argon2 parameters left dummy hash empty")
	}
	if _, err := VerifyPassword("wrong", service.dummyHash); err != nil {
		t.Fatalf("safe dummy hash is invalid: %v", err)
	}
}

func TestEmergencyRevokeRemovesAllUserSessions(t *testing.T) {
	sessions := &httpSessionRepo{session: &Session{ID: "s1", UserID: "u1", TokenHash: HashSessionToken("token"), ExpiresAt: time.Now().UTC().Add(time.Hour)}}
	service := NewAuthService(nil, sessions, nil, TestArgon2Params)
	if err := service.RevokeAllUserSessions(context.Background(), "u1"); err != nil {
		t.Fatalf("revoke all sessions: %v", err)
	}
	if sessions.session != nil {
		t.Fatal("emergency revoke left a user session active")
	}
}

func TestCreatedSessionStoresOnlyTokenHash(t *testing.T) {
	sessions := &httpSessionRepo{}
	service := NewAuthService(nil, sessions, nil, TestArgon2Params)
	rawToken, session, err := service.CreateSession(context.Background(), "u1", "browser", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if rawToken == "" || session.TokenHash == "" || rawToken == session.TokenHash {
		t.Fatal("session persistence exposed or omitted the raw token")
	}
	if sessions.session.TokenHash != HashSessionToken(rawToken) {
		t.Fatal("stored session hash does not match token digest")
	}
}

func TestDisableUserFailsClosedWithoutMobileSessionRevoker(t *testing.T) {
	service := NewAuthService(httpUserRepo{user: &User{ID: "u1", Active: true}}, &httpSessionRepo{}, nil, TestArgon2Params)
	if err := service.SetUserActive(context.Background(), "u1", false); !errors.Is(err, ErrMobileSessionRevokerRequired) {
		t.Fatalf("disable without mobile session revoker error = %v, want ErrMobileSessionRevokerRequired", err)
	}
}

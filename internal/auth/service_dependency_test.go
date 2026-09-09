package auth

import (
	"context"
	"errors"
	"testing"
)

func TestAuthServiceRejectsMissingRepositories(t *testing.T) {
	ctx := context.Background()
	service := NewAuthService(nil, nil, TestArgon2Params)
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
	service := NewAuthService(nil, nil, Argon2Params{})
	if service.dummyHash == "" {
		t.Fatal("invalid Argon2 parameters left dummy hash empty")
	}
	if _, err := VerifyPassword("wrong", service.dummyHash); err != nil {
		t.Fatalf("safe dummy hash is invalid: %v", err)
	}
}

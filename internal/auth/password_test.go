package auth

import (
	"testing"
)

func TestArgon2PasswordHashingAndVerification(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password, TestArgon2Params)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if len(hash) == 0 {
		t.Fatalf("expected non-empty hash string")
	}

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("unexpected error verifying correct password: %v", err)
	}
	if !valid {
		t.Fatalf("expected password verification to succeed for matching password")
	}

	invalid, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("unexpected error verifying incorrect password: %v", err)
	}
	if invalid {
		t.Fatalf("expected password verification to fail for non-matching password")
	}
}

func TestVerifyPasswordInvalidFormat(t *testing.T) {
	_, err := VerifyPassword("password", "not-a-valid-argon2-hash")
	if err != ErrInvalidHash {
		t.Fatalf("expected ErrInvalidHash, got: %v", err)
	}
}

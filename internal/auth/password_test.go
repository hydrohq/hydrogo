package auth

import (
	"strings"
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

func TestValidatePasswordStrength(t *testing.T) {
	for _, weak := range []string{"short", "password123", "aaaaaaaaaaaaaaaa", "12345678", "AdminWelcome2026!"} {
		if err := ValidatePasswordStrength(weak); err != ErrWeakPassword {
			t.Errorf("ValidatePasswordStrength(%q) = %v, want ErrWeakPassword", weak, err)
		}
	}
	for _, strong := range []string{"Lime-Train7!Harbor", "correct horse battery staple"} {
		if err := ValidatePasswordStrength(strong); err != nil {
			t.Errorf("ValidatePasswordStrength(%q) = %v", strong, err)
		}
	}
}

func TestVerifyPasswordInvalidFormat(t *testing.T) {
	_, err := VerifyPassword("password", "not-a-valid-argon2-hash")
	if err != ErrInvalidHash {
		t.Fatalf("expected ErrInvalidHash, got: %v", err)
	}
}

func TestArgon2ParametersAreBounded(t *testing.T) {
	if _, err := HashPassword("password", Argon2Params{Memory: 512 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}); err != ErrInvalidArgonParams {
		t.Fatalf("oversized hash parameters error = %v, want %v", err, ErrInvalidArgonParams)
	}
	malicious := "$argon2id$v=19$m=999999,t=1,p=1$YWJjZGVmZ2hpamts$YWJjZGVmZ2hpamts"
	if _, err := VerifyPassword("password", malicious); err != ErrInvalidHash {
		t.Fatalf("malicious encoded parameters error = %v, want %v", err, ErrInvalidHash)
	}
	if _, err := VerifyPassword("password", strings.Repeat("x", maxEncodedPasswordHashBytes+1)); err != ErrInvalidHash {
		t.Fatalf("oversized encoded hash error = %v, want %v", err, ErrInvalidHash)
	}
}

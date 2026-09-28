package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("invalid password hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
	ErrInvalidArgonParams  = errors.New("invalid argon2 parameters")
	ErrWeakPassword        = errors.New("password does not meet strength requirements")
)

type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024, // 64 MB
	Iterations:  1,
	Parallelism: 4,
	SaltLength:  16,
	KeyLength:   32,
}

// TestArgon2Params provides faster hashing parameters for unit tests.
var TestArgon2Params = Argon2Params{
	Memory:      1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  8,
	KeyLength:   16,
}

const maxEncodedPasswordHashBytes = 4096

var commonPasswordFragments = []string{"password", "qwerty", "welcome", "admin", "secret", "letmein", "iloveyou", "monkey", "dragon", "football", "abc123"}

// ValidatePasswordStrength applies a dependency-free server-side estimate for
// newly chosen passwords. Long passphrases remain valid without character
// class rules; common words and low-entropy strings are rejected.
func ValidatePasswordStrength(password string) error {
	runes := []rune(password)
	if len(runes) < 8 || len(runes) > 1024 || !utf8.ValidString(password) {
		return ErrWeakPassword
	}
	lower := strings.ToLower(password)
	for _, fragment := range commonPasswordFragments {
		if strings.Contains(lower, fragment) {
			return ErrWeakPassword
		}
	}
	pool := 0
	classes := 0
	var hasLower, hasUpper, hasDigit, hasSymbol bool
	unique := make(map[rune]struct{}, len(runes))
	for _, r := range runes {
		unique[r] = struct{}{}
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsSpace(r):
			hasSymbol = true
		default:
			hasSymbol = true
		}
	}
	if hasLower {
		pool += 26
		classes++
	}
	if hasUpper {
		pool += 26
		classes++
	}
	if hasDigit {
		pool += 10
		classes++
	}
	if hasSymbol {
		pool += 32
		classes++
	}
	// A single repeated character or a single class with no meaningful variety
	// has substantially less entropy than length alone suggests.
	if len(unique) < 4 || classes == 0 {
		return ErrWeakPassword
	}
	entropy := float64(len(runes)) * math.Log2(float64(pool))
	if entropy < 40 {
		return ErrWeakPassword
	}
	return nil
}

func validateArgon2Params(params Argon2Params) error {
	if params.Memory < 8 || params.Memory > 256*1024 || params.Iterations < 1 || params.Iterations > 10 || params.Parallelism < 1 || params.Parallelism > 16 || params.Memory < uint32(params.Parallelism)*8 || params.SaltLength < 8 || params.SaltLength > 64 || params.KeyLength < 16 || params.KeyLength > 64 {
		return ErrInvalidArgonParams
	}
	return nil
}

func HashPassword(password string, params Argon2Params) (string, error) {
	if err := validateArgon2Params(params); err != nil {
		return "", err
	}
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate random salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	if len(encodedHash) == 0 || len(encodedHash) > maxEncodedPasswordHashBytes {
		return false, ErrInvalidHash
	}
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrInvalidHash
	}
	if fmt.Sprintf("v=%d", version) != parts[2] {
		return false, ErrInvalidHash
	}
	if version != argon2.Version {
		return false, ErrIncompatibleVersion
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, ErrInvalidHash
	}
	if fmt.Sprintf("m=%d,t=%d,p=%d", memory, iterations, parallelism) != parts[3] {
		return false, ErrInvalidHash
	}
	params := Argon2Params{Memory: memory, Iterations: iterations, Parallelism: parallelism, SaltLength: 0, KeyLength: 0}
	if err := validateArgon2Params(paramsWithUnknownLengths(params)); err != nil {
		return false, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHash
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidHash
	}

	keyLen := uint32(len(decodedHash))
	params.SaltLength = uint32(len(salt))
	params.KeyLength = keyLen
	if err := validateArgon2Params(params); err != nil {
		return false, ErrInvalidHash
	}
	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLen)

	if subtle.ConstantTimeCompare(decodedHash, computedHash) == 1 {
		return true, nil
	}

	return false, nil
}

// paramsWithUnknownLengths validates only the resource-bearing fields before
// decoding attacker-controlled salt and key material.
func paramsWithUnknownLengths(params Argon2Params) Argon2Params {
	params.SaltLength = 8
	params.KeyLength = 16
	return params
}

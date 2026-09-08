package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/pquerna/otp/totp"
)

var (
	ErrMFAInvalidCode = errors.New("invalid MFA code")
	ErrMFAUnavailable = errors.New("MFA is unavailable")
)

type MFAStore interface {
	PutFactor(context.Context, database.Executor, string, []byte, bool) error
	GetFactor(context.Context, database.Executor, string) ([]byte, bool, error)
	PutRecoveryCode(context.Context, database.Executor, string, string, string) error
	ConsumeRecoveryCode(context.Context, database.Executor, string, string, time.Time) error
}

type MFAService struct {
	Store     MFAStore
	MasterKey []byte
}

type MFAEnrollment struct {
	Secret        string
	Provisioning  string
	RecoveryCodes []string
}

func (s MFAService) Enroll(ctx context.Context, tx database.Executor, userID, accountName string) (MFAEnrollment, error) {
	if len(s.MasterKey) != 16 && len(s.MasterKey) != 24 && len(s.MasterKey) != 32 {
		return MFAEnrollment{}, ErrMFAUnavailable
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "HydroGo", AccountName: accountName})
	if err != nil {
		return MFAEnrollment{}, fmt.Errorf("generate TOTP factor: %w", err)
	}
	ciphertext, err := encryptMFA(s.MasterKey, []byte(key.Secret()))
	if err != nil {
		return MFAEnrollment{}, err
	}
	if err := s.Store.PutFactor(ctx, tx, userID, ciphertext, false); err != nil {
		return MFAEnrollment{}, err
	}
	codes, err := s.generateRecoveryCodes(ctx, tx, userID)
	if err != nil {
		return MFAEnrollment{}, err
	}
	return MFAEnrollment{Secret: key.Secret(), Provisioning: key.URL(), RecoveryCodes: codes}, nil
}

func (s MFAService) Enable(ctx context.Context, tx database.Executor, userID string) error {
	factor, _, err := s.Store.GetFactor(ctx, tx, userID)
	if err != nil {
		return err
	}
	return s.Store.PutFactor(ctx, tx, userID, factor, true)
}

func (s MFAService) Verify(ctx context.Context, db database.Executor, userID, code string) error {
	ciphertext, enabled, err := s.Store.GetFactor(ctx, db, userID)
	if err != nil {
		return err
	}
	if !enabled {
		return ErrMFAInvalidCode
	}
	secret, err := decryptMFA(s.MasterKey, ciphertext)
	if err != nil || !totp.Validate(strings.TrimSpace(code), string(secret)) {
		return ErrMFAInvalidCode
	}
	return nil
}

func (s MFAService) ConsumeRecoveryCode(ctx context.Context, tx database.Executor, userID, code string) error {
	digest := sha256.Sum256([]byte(strings.TrimSpace(strings.ToUpper(code))))
	return s.Store.ConsumeRecoveryCode(ctx, tx, userID, hex.EncodeToString(digest[:]), time.Now().UTC())
}

func (s MFAService) generateRecoveryCodes(ctx context.Context, tx database.Executor, userID string) ([]string, error) {
	codes := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		buf := make([]byte, 5)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		code := strings.ToUpper(hex.EncodeToString(buf))
		digest := sha256.Sum256([]byte(code))
		if err := s.Store.PutRecoveryCode(ctx, tx, userID, fmt.Sprintf("recovery-%d-%s", i, userID), hex.EncodeToString(digest[:])); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func encryptMFA(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decryptMFA(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(ciphertext) < gcm.NonceSize() {
		return nil, ErrMFAUnavailable
	}
	return gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
}

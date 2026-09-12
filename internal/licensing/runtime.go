package licensing

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
)

func VerifyFile(ctx context.Context, path, projectID, edition string, features []string) (*VerificationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if TrustedPublicKeyBase64 == "" {
		return nil, errors.New("official trusted license public key is not configured")
	}
	pub, err := DecodePublicKey(TrustedPublicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode trusted license key: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read license %s: %w", path, err)
	}
	result, err := Verify(data, VerificationOptions{ExpectedProject: projectID, RequiredEdition: edition, RequiredFeatures: features, TrustedPublicKey: ed25519.PublicKey(pub)})
	return result, err
}

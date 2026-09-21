package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrInvalidObjectIntegrity = errors.New("invalid object integrity expectation")
	ErrObjectIntegrity        = errors.New("object integrity verification failed")
)

// VerifyObject streams an immutable object through the provider-neutral read
// boundary and verifies the expected byte count and SHA-256 digest. It is
// deliberately bounded and fails closed for a missing store, lazy provider
// read failure, overlong object, or digest mismatch.
func VerifyObject(ctx context.Context, store BlobStore, key string, expectedSize int64, expectedSHA256 string) error {
	if store == nil || expectedSize < 0 || expectedSize == 1<<63-1 || !validSHA256(expectedSHA256) {
		return ErrInvalidObjectIntegrity
	}
	reader, err := store.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("%w: open object", ErrObjectIntegrity)
	}
	defer reader.Close()
	hash := sha256.New()
	read, err := io.Copy(hash, io.LimitReader(reader, expectedSize+1))
	if err != nil || read != expectedSize {
		return fmt.Errorf("%w: byte count", ErrObjectIntegrity)
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expectedSHA256) {
		return fmt.Errorf("%w: checksum", ErrObjectIntegrity)
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

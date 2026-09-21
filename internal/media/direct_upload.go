package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

var ErrDirectUploadUnavailable = errors.New("direct quarantine upload is unavailable")

// DirectUploadIntent is an untrusted-client upload capability, not an asset.
// Its key is always quarantine-scoped and must be verified and promoted by a
// separate authorized completion workflow before it can become ready.
type DirectUploadIntent struct {
	ObjectKey string
	URL       string
	ExpiresAt time.Time
}

// PrepareQuarantineUpload creates an opaque, server-selected object key and a
// bounded direct PUT URL. It intentionally creates neither a public object nor
// a ready database record.
func (s Service) PrepareQuarantineUpload(ctx context.Context, expiry time.Duration) (DirectUploadIntent, error) {
	provider, ok := s.BlobStore.(QuarantineUploadURLProvider)
	if !ok || expiry < time.Second || expiry > 15*time.Minute {
		return DirectUploadIntent{}, ErrDirectUploadUnavailable
	}
	id, err := directUploadID()
	if err != nil {
		return DirectUploadIntent{}, err
	}
	key := "quarantine/" + id
	if scope, scopeErr := tenant.FromContext(ctx); scopeErr == nil {
		key = "tenants/" + scope.TenantID + "/" + key
	}
	url, err := provider.PresignedQuarantineUploadURL(ctx, key, expiry)
	if err != nil {
		return DirectUploadIntent{}, ErrDirectUploadUnavailable
	}
	return DirectUploadIntent{ObjectKey: key, URL: url, ExpiresAt: time.Now().UTC().Add(expiry)}, nil
}

func directUploadID() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

package media

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type quarantineSigner struct {
	key string
	err error
}

func (s *quarantineSigner) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (s *quarantineSigner) Get(context.Context, string) (io.ReadCloser, error)          { return nil, nil }
func (s *quarantineSigner) Delete(context.Context, string) error                        { return nil }
func (s *quarantineSigner) PresignedQuarantineUploadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	s.key = key
	if s.err != nil {
		return "", s.err
	}
	return "https://storage.example.test/" + key, nil
}

func TestPrepareQuarantineUploadUsesOpaqueTenantScopedKey(t *testing.T) {
	signer := &quarantineSigner{}
	scope, err := tenant.NewScope("tenant-a", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	intent, err := (Service{BlobStore: signer}).PrepareQuarantineUpload(tenant.WithScope(context.Background(), scope), time.Minute)
	if err != nil || !strings.HasPrefix(intent.ObjectKey, "tenants/tenant-a/quarantine/") || intent.ObjectKey != signer.key || !strings.Contains(intent.URL, intent.ObjectKey) || intent.ExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("intent=%#v signer=%#v err=%v", intent, signer, err)
	}
	if _, err := (Service{}).PrepareQuarantineUpload(context.Background(), time.Minute); !errors.Is(err, ErrDirectUploadUnavailable) {
		t.Fatalf("missing provider = %v", err)
	}
}

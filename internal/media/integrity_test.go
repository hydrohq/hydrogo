package media

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type integrityBlobStore struct {
	contents string
	err      error
}

func (s integrityBlobStore) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (s integrityBlobStore) Get(context.Context, string) (io.ReadCloser, error) {
	if s.err != nil {
		return nil, s.err
	}
	return io.NopCloser(strings.NewReader(s.contents)), nil
}
func (s integrityBlobStore) Delete(context.Context, string) error { return nil }

func TestVerifyObjectRejectsMismatchAndProviderFailure(t *testing.T) {
	const digest = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" // hello
	if err := VerifyObject(context.Background(), integrityBlobStore{contents: "hello"}, "documents/r1.pdf", 5, digest); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		store  BlobStore
		size   int64
		digest string
	}{
		{store: integrityBlobStore{contents: "hello"}, size: 4, digest: digest},
		{store: integrityBlobStore{contents: "hello"}, size: 5, digest: strings.Repeat("0", 64)},
		{store: integrityBlobStore{err: errors.New("provider unavailable")}, size: 5, digest: digest},
	} {
		if err := VerifyObject(context.Background(), test.store, "documents/r1.pdf", test.size, test.digest); !errors.Is(err, ErrObjectIntegrity) {
			t.Fatalf("verification error = %v", err)
		}
	}
	if err := VerifyObject(context.Background(), nil, "documents/r1.pdf", 5, digest); !errors.Is(err, ErrInvalidObjectIntegrity) {
		t.Fatalf("nil store = %v", err)
	}
}

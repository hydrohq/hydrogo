//go:build !hydro_no_s3

package media

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewS3BlobStoreRequiresEndpointAndBucket(t *testing.T) {
	for _, cfg := range []S3Config{{}, {Endpoint: "http://127.0.0.1:9000"}, {Bucket: "hydro"}} {
		if _, err := NewS3BlobStore(cfg); err == nil {
			t.Fatalf("config %+v unexpectedly accepted", cfg)
		}
	}
}

func TestNewS3BlobStoreRejectsUnsafeURLEndpoints(t *testing.T) {
	for _, endpoint := range []string{
		"ftp://storage.example.test:9000",
		"https://access:secret@storage.example.test:9000",
		"https://storage.example.test:9000/?region=eu",
		"https://storage.example.test:9000/#fragment",
	} {
		if _, err := NewS3BlobStore(S3Config{Endpoint: endpoint, Bucket: "hydro"}); err == nil {
			t.Fatalf("unsafe S3 endpoint accepted: %q", endpoint)
		}
	}
}

func TestS3BlobStoreUnavailableFailsClosed(t *testing.T) {
	var store *S3BlobStore
	if err := store.Put(context.Background(), "key", strings.NewReader("x"), 1, "text/plain"); err == nil {
		t.Fatal("nil store Put unexpectedly succeeded")
	}
	if _, err := store.Get(context.Background(), "key"); err == nil {
		t.Fatal("nil store Get unexpectedly succeeded")
	}
	if err := store.Delete(context.Background(), "key"); err == nil {
		t.Fatal("nil store Delete unexpectedly succeeded")
	}
}

func TestS3BlobStorePublicURLIsExplicitAndBounded(t *testing.T) {
	store, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "hydro", PublicBaseURL: "https://cdn.example.test/media/", ForcePathStyle: true})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.PublicURL("images/hero one.png")
	if err != nil || got != "https://cdn.example.test/media/images/hero%20one.png" {
		t.Fatalf("public URL = %q, err=%v", got, err)
	}
	for _, key := range []string{"../secret", "images//x", "images\\x", ""} {
		if _, err := store.PublicURL(key); err == nil {
			t.Fatalf("unsafe key accepted: %q", key)
		}
	}
}

func TestS3BlobStoreRejectsNonLoopbackHTTPPublicURL(t *testing.T) {
	if _, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "hydro", PublicBaseURL: "http://cdn.example.test"}); err == nil {
		t.Fatal("non-loopback HTTP public URL accepted")
	}
	for _, base := range []string{"https://user:secret@cdn.example.test/media", "https://cdn.example.test/media?cache=1", "https://cdn.example.test/media#fragment"} {
		if _, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "hydro", PublicBaseURL: base}); err == nil {
			t.Fatalf("unsafe public base accepted: %q", base)
		}
	}
}

func TestS3BlobStoreValidatesKeysForAllObjectOperations(t *testing.T) {
	store, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:1", Bucket: "hydro"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../secret", "images//x", "images\\x", ""} {
		if err := store.Put(context.Background(), key, strings.NewReader("x"), 1, "text/plain"); err == nil || err.Error() != "invalid object key" {
			t.Fatalf("Put key %q error = %v", key, err)
		}
		if _, err := store.Get(context.Background(), key); err == nil || err.Error() != "invalid object key" {
			t.Fatalf("Get key %q error = %v", key, err)
		}
		if err := store.Delete(context.Background(), key); err == nil || err.Error() != "invalid object key" {
			t.Fatalf("Delete key %q error = %v", key, err)
		}
	}
}

func TestS3BlobStorePresignedURLIsBounded(t *testing.T) {
	store, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "hydro", Region: "us-east-1", AccessKey: "access", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	for _, expiry := range []time.Duration{0, 500 * time.Millisecond, 25 * time.Hour} {
		if _, err := store.PresignedURL(context.Background(), "images/hero.png", expiry); err == nil {
			t.Fatalf("expiry accepted: %s", expiry)
		}
	}
}

func TestS3BlobStorePresignedQuarantineUploadURLIsNarrow(t *testing.T) {
	store, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "hydro", Region: "us-east-1", AccessKey: "access", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if url, err := store.PresignedQuarantineUploadURL(context.Background(), "tenants/tenant-a/quarantine/upload-a", time.Minute); err != nil || !strings.Contains(url, "X-Amz-Signature") {
		t.Fatalf("scoped quarantine URL = %q, %v", url, err)
	}
	for _, test := range []struct {
		key    string
		expiry time.Duration
	}{
		{key: "documents/final.pdf", expiry: time.Minute},
		{key: "quarantine/upload-a", expiry: 0},
		{key: "quarantine/upload-a", expiry: 16 * time.Minute},
	} {
		if _, err := store.PresignedQuarantineUploadURL(context.Background(), test.key, test.expiry); err == nil {
			t.Fatalf("quarantine upload accepted key=%q expiry=%s", test.key, test.expiry)
		}
	}
}

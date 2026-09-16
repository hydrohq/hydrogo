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
}

func TestS3BlobStorePresignedURLIsBounded(t *testing.T) {
	store, err := NewS3BlobStore(S3Config{Endpoint: "http://127.0.0.1:9000", Bucket: "hydro", AccessKey: "access", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	for _, expiry := range []time.Duration{0, 500 * time.Millisecond, 25 * time.Hour} {
		if _, err := store.PresignedURL(context.Background(), "images/hero.png", expiry); err == nil {
			t.Fatalf("expiry accepted: %s", expiry)
		}
	}
}

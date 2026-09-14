package media

import (
	"context"
	"strings"
	"testing"
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

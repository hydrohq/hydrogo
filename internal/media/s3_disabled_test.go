//go:build hydro_no_s3

package media

import (
	"errors"
	"testing"
)

func TestNoS3BuildRejectsProviderOperations(t *testing.T) {
	if _, err := NewS3BlobStore(S3Config{Endpoint: "https://s3.example.test", Bucket: "assets"}); !errors.Is(err, ErrS3Unavailable) {
		t.Fatalf("NewS3BlobStore error = %v, want ErrS3Unavailable", err)
	}
}

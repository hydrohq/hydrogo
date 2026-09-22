package media

import (
	"context"
	"errors"
	"io"
	"time"
)

// BlobStore is the provider-neutral boundary for immutable media objects.
// Domain and application code must only deal with keys and metadata.
type BlobStore interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

// PublicURLProvider is optional and is used only by transports that need a
// public/CDN object URL. Core storage code continues to use opaque keys.
type PublicURLProvider interface {
	PublicURL(string) (string, error)
}

// SignedURLProvider is an optional private-object capability. Callers remain
// responsible for authorization before asking for a URL.
type SignedURLProvider interface {
	PresignedURL(context.Context, string, time.Duration) (string, error)
}

// QuarantineUploadURLProvider may grant a client a short-lived direct PUT only
// for a pending/quarantine object. It never grants a URL for a final ready
// object; the application must verify and promote the object separately.
type QuarantineUploadURLProvider interface {
	PresignedQuarantineUploadURL(context.Context, string, time.Duration) (string, error)
}

// S3Config enables AWS S3 and S3-compatible services when the Edition includes
// the optional S3 provider.
type S3Config struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Region         string
	UseSSL         bool
	ForcePathStyle bool
	PublicBaseURL  string
}

var ErrS3Unavailable = errors.New("s3 media provider unavailable in this Edition build")

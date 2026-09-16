package media

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

// S3Config enables AWS S3 and S3-compatible services (MinIO, R2, B2).
// An empty endpoint/credentials keeps the default local filesystem path.
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

type S3BlobStore struct {
	client     *minio.Client
	bucket     string
	publicBase *url.URL
}

func NewS3BlobStore(cfg S3Config) (*S3BlobStore, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("s3 endpoint and bucket are required")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		if parsed.Scheme == "https" {
			cfg.UseSSL = true
		}
		endpoint = parsed.Host
	}
	var publicBase *url.URL
	if strings.TrimSpace(cfg.PublicBaseURL) != "" {
		parsed, err := url.Parse(strings.TrimSpace(cfg.PublicBaseURL))
		if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !isLoopbackHost(parsed.Hostname())) {
			return nil, errors.New("public base URL must use HTTPS unless it targets loopback")
		}
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		publicBase = parsed
	}
	options := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	}
	if cfg.ForcePathStyle {
		options.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, options)
	if err != nil {
		return nil, err
	}
	return &S3BlobStore{client: client, bucket: cfg.Bucket, publicBase: publicBase}, nil
}

func isLoopbackHost(host string) bool {
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

// PublicURL returns a CDN/object URL only when an explicit public base was
// configured. Keys are treated as opaque path segments and cannot escape the
// configured base path.
func (s *S3BlobStore) PublicURL(key string) (string, error) {
	if s == nil || s.publicBase == nil {
		return "", errors.New("public object URL is not configured")
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > 1024 {
		return "", errors.New("invalid object key")
	}
	for _, r := range key {
		if unicode.IsControl(r) || r == '\\' {
			return "", errors.New("invalid object key")
		}
	}
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("invalid object key")
		}
	}
	escaped := make([]string, len(parts))
	for i, part := range parts {
		escaped[i] = url.PathEscape(part)
	}
	copyBase := *s.publicBase
	copyBase.Path = strings.TrimRight(copyBase.Path, "/") + "/" + strings.Join(parts, "/")
	copyBase.RawPath = strings.TrimRight(copyBase.EscapedPath(), "/") + "/" + strings.Join(escaped, "/")
	return copyBase.String(), nil
}

func (s *S3BlobStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if s == nil || s.client == nil {
		return errors.New("s3 blob store is unavailable")
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3BlobStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("s3 blob store is unavailable")
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (s *S3BlobStore) Delete(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return errors.New("s3 blob store is unavailable")
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

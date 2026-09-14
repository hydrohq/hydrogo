package media

import (
	"context"
	"errors"
	"io"
	"net/url"
	"strings"

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
	client *minio.Client
	bucket string
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
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &S3BlobStore{client: client, bucket: cfg.Bucket}, nil
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

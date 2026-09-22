//go:build hydro_no_s3

package media

import (
	"context"
	"io"
	"time"
)

type S3BlobStore struct{}

func NewS3BlobStore(S3Config) (*S3BlobStore, error)   { return nil, ErrS3Unavailable }
func (*S3BlobStore) PublicURL(string) (string, error) { return "", ErrS3Unavailable }
func (*S3BlobStore) PresignedURL(context.Context, string, time.Duration) (string, error) {
	return "", ErrS3Unavailable
}
func (*S3BlobStore) PresignedQuarantineUploadURL(context.Context, string, time.Duration) (string, error) {
	return "", ErrS3Unavailable
}
func (*S3BlobStore) Put(context.Context, string, io.Reader, int64, string) error {
	return ErrS3Unavailable
}
func (*S3BlobStore) Get(context.Context, string) (io.ReadCloser, error) { return nil, ErrS3Unavailable }
func (*S3BlobStore) Delete(context.Context, string) error               { return ErrS3Unavailable }

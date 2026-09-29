//go:build hydro_no_s3

package backup

import (
	"context"
	"errors"
	"time"
)

type S3Config struct {
	Endpoint, Bucket, Region, AccessKey, SecretKey, Prefix string
	Secure, ForcePathStyle                                 bool
}
type S3Dispatcher struct{}

func NewS3Dispatcher(S3Config) (*S3Dispatcher, error) {
	return nil, errors.New("S3 support is disabled in this build")
}
func (*S3Dispatcher) UploadSnapshot(context.Context, string, string) error {
	return errors.New("S3 support is disabled in this build")
}
func (*S3Dispatcher) Prune(context.Context, time.Duration, time.Time) (int, error) {
	return 0, errors.New("S3 support is disabled in this build")
}
func (*S3Dispatcher) ListRecent(context.Context, int) ([]SnapshotSummary, error) {
	return nil, errors.New("S3 support is disabled in this build")
}
func RunDaily(ctx context.Context, _ int, _ func(context.Context) error, _ func(error)) { <-ctx.Done() }

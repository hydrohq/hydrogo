//go:build !hydro_no_s3

package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

type S3Config struct {
	Endpoint, Bucket, Region, AccessKey, SecretKey, Prefix string
	Secure, ForcePathStyle                                 bool
}
type S3Dispatcher struct {
	client         *minio.Client
	bucket, prefix string
}

func NewS3Dispatcher(cfg S3Config) (*S3Dispatcher, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("backup S3 endpoint, bucket, and credentials are required")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("invalid backup S3 endpoint")
		}
		cfg.Secure = parsed.Scheme == "https"
		endpoint = parsed.Host
	}
	prefix := strings.Trim(strings.TrimSpace(cfg.Prefix), "/")
	if prefix != "" {
		prefix += "/"
	}
	options := &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), Secure: cfg.Secure, Region: cfg.Region}
	if cfg.ForcePathStyle {
		options.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, options)
	if err != nil {
		return nil, err
	}
	return &S3Dispatcher{client: client, bucket: cfg.Bucket, prefix: prefix}, nil
}

// UploadSnapshot streams a gzip tar archive from disk to object storage.
func (d *S3Dispatcher) UploadSnapshot(ctx context.Context, snapshotDir, objectName string) error {
	if d == nil || d.client == nil {
		return errors.New("backup S3 dispatcher unavailable")
	}
	if filepath.Base(objectName) != objectName || !strings.HasSuffix(objectName, ".tar.gz") {
		return errors.New("invalid backup object name")
	}
	reader, writer := io.Pipe()
	key := d.prefix + objectName
	go func() { err := WriteSnapshotArchive(ctx, writer, snapshotDir); _ = writer.CloseWithError(err) }()
	_, err := d.client.PutObject(ctx, d.bucket, key, reader, -1, minio.PutObjectOptions{ContentType: "application/gzip", ServerSideEncryption: encrypt.NewSSE()})
	_ = reader.Close()
	if err != nil {
		return fmt.Errorf("upload backup object: %w", err)
	}
	return nil
}

// WriteSnapshotArchive streams a verified snapshot directory as gzip-compressed tar.
func (d *S3Dispatcher) Prune(ctx context.Context, retention time.Duration, now time.Time) (int, error) {
	if d == nil || d.client == nil {
		return 0, errors.New("backup S3 dispatcher unavailable")
	}
	if retention < 24*time.Hour {
		return 0, errors.New("backup retention must be at least one day")
	}
	if now.IsZero() {
		now = time.Now()
	}
	cutoff := now.Add(-retention)
	removed := 0
	for object := range d.client.ListObjects(ctx, d.bucket, minio.ListObjectsOptions{Prefix: d.prefix, Recursive: true}) {
		if object.Err != nil {
			return removed, object.Err
		}
		if !strings.HasSuffix(object.Key, ".tar.gz") || object.LastModified.IsZero() || !object.LastModified.Before(cutoff) {
			continue
		}
		if err := d.client.RemoveObject(ctx, d.bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// ListRecent returns a bounded newest-first view of scheduled cloud snapshots.
func (d *S3Dispatcher) ListRecent(ctx context.Context, limit int) ([]SnapshotSummary, error) {
	if d == nil || d.client == nil {
		return nil, errors.New("backup S3 dispatcher unavailable")
	}
	if limit < 1 || limit > 100 {
		return nil, errors.New("backup listing limit is out of range")
	}
	items := make([]SnapshotSummary, 0, limit)
	for object := range d.client.ListObjects(ctx, d.bucket, minio.ListObjectsOptions{Prefix: d.prefix, Recursive: true}) {
		if object.Err != nil {
			return nil, object.Err
		}
		if !strings.HasSuffix(object.Key, ".tar.gz") || object.LastModified.IsZero() {
			continue
		}
		items = append(items, SnapshotSummary{Name: filepath.Base(object.Key), CreatedAt: object.LastModified.UTC(), Bytes: object.Size})
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
		if len(items) > limit {
			items = items[:limit]
		}
	}
	return items, nil
}

func RunDaily(ctx context.Context, hourUTC int, run func(context.Context) error, onError func(error)) {
	if hourUTC < 0 || hourUTC > 23 || run == nil {
		return
	}
	for {
		now := time.Now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if err := run(ctx); err != nil && onError != nil {
			onError(err)
		}
	}
}

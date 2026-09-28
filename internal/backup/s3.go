//go:build !hydro_no_s3

package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
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
	go func() { err := writeSnapshotArchive(ctx, writer, snapshotDir); _ = writer.CloseWithError(err) }()
	_, err := d.client.PutObject(ctx, d.bucket, key, reader, -1, minio.PutObjectOptions{ContentType: "application/gzip", ServerSideEncryption: encrypt.NewSSE()})
	_ = reader.Close()
	if err != nil {
		return fmt.Errorf("upload backup object: %w", err)
	}
	return nil
}

func writeSnapshotArchive(ctx context.Context, dst io.Writer, root string) error {
	gz := gzip.NewWriter(dst)
	tarWriter := tar.NewWriter(gz)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("backup snapshot contains a symlink")
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tarWriter, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if closeErr := tarWriter.Close(); err == nil {
		err = closeErr
	}
	if closeErr := gz.Close(); err == nil {
		err = closeErr
	}
	return err
}

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

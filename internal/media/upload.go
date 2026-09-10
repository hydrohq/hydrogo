package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	_ "golang.org/x/image/webp"
)

var (
	ErrUploaderRequired    = errors.New("uploader is required")
	ErrUploaderUnavailable = errors.New("uploader is unavailable")
	ErrFileTooLarge        = errors.New("file exceeds maximum allowed size")
	ErrDisallowedMimeType  = errors.New("disallowed file mime type")
	ErrInvalidImageData    = errors.New("corrupt or invalid image payload")
	ErrImageTooLarge       = errors.New("image dimensions exceed maximum allowed pixels")
	ErrInvalidVisibility   = errors.New("invalid media visibility")
	ErrAssetState          = errors.New("invalid media asset state")
)

const (
	VisibilityPrivate    = "private"
	VisibilityPublic     = "public"
	maxUploadConcurrency = 1024
)

type Asset struct {
	ID               string `json:"id"`
	StorageKey       string `json:"storage_key"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	SizeBytes        int64  `json:"size_bytes"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	Visibility       string `json:"visibility"`
	Status           string `json:"status"`
}

var AllowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type UploadResult struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	StoragePath string `json:"storage_path"`
	MimeType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}

type Uploader struct {
	StorageDir string
	MaxBytes   int64
	MaxPixels  int64
	slots      chan struct{}
}

func NewUploader(storageDir string, maxBytes int64) (*Uploader, error) {
	return NewUploaderWithConcurrency(storageDir, maxBytes, 1)
}

// NewUploaderWithConcurrency creates an uploader with bounded disk/CPU work.
// The limit covers the complete temporary-file, hashing and image validation
// lifecycle of each upload.
func NewUploaderWithConcurrency(storageDir string, maxBytes int64, maxConcurrent int) (*Uploader, error) {
	if strings.TrimSpace(storageDir) == "" {
		return nil, errors.New("media storage directory is required")
	}
	if maxBytes <= 0 {
		maxBytes = 10 << 20 // 10 MB default
	}
	if maxBytes == 1<<63-1 {
		return nil, errors.New("maximum upload size is too large")
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	if maxConcurrent > maxUploadConcurrency {
		return nil, fmt.Errorf("maximum upload concurrency is %d", maxUploadConcurrency)
	}
	if err := os.MkdirAll(storageDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	info, err := os.Lstat(storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect storage directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("media storage directory must be a real directory")
	}
	return &Uploader{
		StorageDir: storageDir,
		MaxBytes:   maxBytes,
		MaxPixels:  40_000_000,
		slots:      make(chan struct{}, maxConcurrent),
	}, nil
}

func (u *Uploader) Save(r io.Reader, originalFilename string) (*UploadResult, error) {
	return u.SaveContext(context.Background(), r, originalFilename)
}

func (u *Uploader) SaveContext(ctx context.Context, r io.Reader, originalFilename string) (*UploadResult, error) {
	if u == nil {
		return nil, ErrUploaderRequired
	}
	if strings.TrimSpace(u.StorageDir) == "" || u.MaxBytes <= 0 || u.MaxPixels <= 0 {
		return nil, ErrUploaderUnavailable
	}
	if r == nil {
		return nil, errors.New("upload reader is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := u.acquire(ctx); err != nil {
		return nil, err
	}
	defer u.release()

	// 1. Create a bounded reader
	lr := &io.LimitedReader{R: r, N: u.MaxBytes + 1}

	// 2. Read initial 512 bytes for MIME detection
	header := make([]byte, 512)
	n, err := io.ReadFull(lr, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed reading file header: %w", err)
	}
	header = header[:n]

	detectedMime := http.DetectContentType(header)
	// Strip parameters like charset
	if idx := strings.Index(detectedMime, ";"); idx != -1 {
		detectedMime = strings.TrimSpace(detectedMime[:idx])
	}

	ext, allowed := AllowedMimeTypes[detectedMime]
	if !allowed {
		return nil, fmt.Errorf("%w: %s", ErrDisallowedMimeType, detectedMime)
	}

	// 3. Write to a temporary file while hashing SHA256
	tempFile, err := os.CreateTemp(u.StorageDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp upload file: %w", err)
	}
	defer func() {
		tempFile.Close()
		_ = os.Remove(tempFile.Name()) // remove if not renamed
	}()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(tempFile, hasher)

	// Write the initial header
	if _, err := multiWriter.Write(header); err != nil {
		return nil, err
	}

	// Stream the remaining body
	written, err := io.Copy(multiWriter, lr)
	if err != nil {
		return nil, err
	}

	totalSize := int64(len(header)) + written
	if totalSize > u.MaxBytes {
		return nil, ErrFileTooLarge
	}

	// 4. Verify image dimensions if image
	var width, height int
	if strings.HasPrefix(detectedMime, "image/") {
		if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		imgConfig, _, err := image.DecodeConfig(tempFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidImageData, err)
		}
		width, height = imgConfig.Width, imgConfig.Height
		if err := validateImageDimensions(width, height, u.MaxPixels); err != nil {
			return nil, err
		}
		// DecodeConfig validates only the image header. Decode the complete
		// payload as well so truncated or malformed compressed data cannot be
		// accepted into immutable storage and fail later when it is served.
		if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		decoded, _, err := image.Decode(tempFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidImageData, err)
		}
		bounds := decoded.Bounds()
		if bounds.Dx() != width || bounds.Dy() != height {
			return nil, ErrInvalidImageData
		}
	}

	if err := tempFile.Sync(); err != nil {
		return nil, fmt.Errorf("failed syncing uploaded file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return nil, fmt.Errorf("failed closing uploaded file: %w", err)
	}

	// 5. Finalize storage with content hash
	hashHex := hex.EncodeToString(hasher.Sum(nil))
	finalFilename := hashHex + ext
	finalPath := filepath.Join(u.StorageDir, finalFilename)

	if info, err := os.Lstat(finalPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("existing media path is not a regular file")
		}
		if existingHash, hashErr := fileSHA256(finalPath); hashErr != nil || existingHash != hashHex {
			return nil, errors.New("existing media content hash mismatch")
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(tempFile.Name(), finalPath); err != nil {
			return nil, fmt.Errorf("failed moving uploaded file: %w", err)
		}
	} else {
		return nil, fmt.Errorf("inspect existing media file: %w", err)
	}

	return &UploadResult{
		ID:          hashHex,
		Filename:    safeOriginalFilename(originalFilename),
		StoragePath: finalPath,
		MimeType:    detectedMime,
		SizeBytes:   totalSize,
		Width:       width,
		Height:      height,
	}, nil
}

func safeOriginalFilename(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\\", "/")
	name := path.Base(value)
	if name == "." || name == "/" || name == "" {
		return "unnamed"
	}
	return name
}

func (u *Uploader) acquire(ctx context.Context) error {
	if u.slots == nil {
		return nil
	}
	select {
	case u.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (u *Uploader) release() {
	if u.slots != nil {
		<-u.slots
	}
}

func validateImageDimensions(width, height int, maxPixels int64) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidImageData
	}
	if maxPixels <= 0 || int64(width) > maxPixels/int64(height) {
		return ErrImageTooLarge
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

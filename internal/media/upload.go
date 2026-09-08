package media

import (
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
	"path/filepath"
	"strings"

	_ "golang.org/x/image/webp"
)

var (
	ErrFileTooLarge       = errors.New("file exceeds maximum allowed size")
	ErrDisallowedMimeType = errors.New("disallowed file mime type")
	ErrInvalidImageData   = errors.New("corrupt or invalid image payload")
	ErrImageTooLarge      = errors.New("image dimensions exceed maximum allowed pixels")
	ErrInvalidVisibility  = errors.New("invalid media visibility")
	ErrAssetState         = errors.New("invalid media asset state")
)

const (
	VisibilityPrivate = "private"
	VisibilityPublic  = "public"
)

type Asset struct {
	ID               string
	StorageKey       string
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	Width            int
	Height           int
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
}

func NewUploader(storageDir string, maxBytes int64) (*Uploader, error) {
	if maxBytes <= 0 {
		maxBytes = 10 << 20 // 10 MB default
	}
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &Uploader{
		StorageDir: storageDir,
		MaxBytes:   maxBytes,
		MaxPixels:  40_000_000,
	}, nil
}

func (u *Uploader) Save(r io.Reader, originalFilename string) (*UploadResult, error) {
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
		width = imgConfig.Width
		height = imgConfig.Height
		if int64(width) > u.MaxPixels/int64(height) {
			return nil, ErrImageTooLarge
		}
	}

	tempFile.Close()

	// 5. Finalize storage with content hash
	hashHex := hex.EncodeToString(hasher.Sum(nil))
	finalFilename := hashHex + ext
	finalPath := filepath.Join(u.StorageDir, finalFilename)

	if err := os.Rename(tempFile.Name(), finalPath); err != nil {
		return nil, fmt.Errorf("failed moving uploaded file: %w", err)
	}

	return &UploadResult{
		ID:          hashHex,
		Filename:    filepath.Base(originalFilename),
		StoragePath: finalPath,
		MimeType:    detectedMime,
		SizeBytes:   totalSize,
		Width:       width,
		Height:      height,
	}, nil
}

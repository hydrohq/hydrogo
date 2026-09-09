package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

type gatedReader struct {
	data    []byte
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *gatedReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.release
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestValidateImageDimensionsRejectsInvalidDimensions(t *testing.T) {
	for _, tc := range []struct {
		name          string
		width, height int
		want          error
	}{
		{name: "zero width", width: 0, height: 10, want: ErrInvalidImageData},
		{name: "zero height", width: 10, height: 0, want: ErrInvalidImageData},
		{name: "negative width", width: -1, height: 10, want: ErrInvalidImageData},
		{name: "pixel budget", width: 11, height: 10, want: ErrImageTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateImageDimensions(tc.width, tc.height, 100); got != tc.want {
				t.Fatalf("validateImageDimensions() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUploaderSaveContextHonorsCancellationWhileBusy(t *testing.T) {
	uploader, err := NewUploaderWithConcurrency(t.TempDir(), 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	reader := &gatedReader{data: createTestPNG(4, 4), started: make(chan struct{}), release: make(chan struct{})}
	firstDone := make(chan error, 1)
	go func() {
		_, err := uploader.SaveContext(context.Background(), reader, "first.png")
		firstDone <- err
	}()
	<-reader.started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := uploader.SaveContext(ctx, bytes.NewReader(createTestPNG(4, 4)), "second.png"); !errors.Is(err, context.Canceled) {
		t.Fatalf("busy uploader error = %v, want context.Canceled", err)
	}
	close(reader.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first upload failed: %v", err)
	}
}

func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestUploadValidPNG(t *testing.T) {
	tempDir := t.TempDir()
	uploader, err := NewUploader(tempDir, 1<<20) // 1MB
	if err != nil {
		t.Fatalf("failed creating uploader: %v", err)
	}

	pngData := createTestPNG(120, 80)
	res, err := uploader.Save(bytes.NewReader(pngData), "../../dangerous/avatar.png")
	if err != nil {
		t.Fatalf("upload valid png failed: %v", err)
	}

	if res.MimeType != "image/png" {
		t.Errorf("got mime %q, want image/png", res.MimeType)
	}
	if res.Width != 120 || res.Height != 80 {
		t.Errorf("got dimensions %dx%d, want 120x80", res.Width, res.Height)
	}
	if res.Filename != "avatar.png" {
		t.Errorf("expected sanitized basename 'avatar.png', got %q", res.Filename)
	}

	// Verify file on disk
	if _, err := os.Stat(res.StoragePath); err != nil {
		t.Errorf("saved file not found on disk at %s: %v", res.StoragePath, err)
	}
}

func TestUploadDisallowedMimeType(t *testing.T) {
	tempDir := t.TempDir()
	uploader, err := NewUploader(tempDir, 1<<20)
	if err != nil {
		t.Fatalf("failed creating uploader: %v", err)
	}

	phpPayload := []byte("<?php echo 'malicious php code'; ?>")
	_, err = uploader.Save(bytes.NewReader(phpPayload), "shell.php")
	if !errors.Is(err, ErrDisallowedMimeType) {
		t.Fatalf("expected ErrDisallowedMimeType, got: %v", err)
	}
}

func TestUploadSizeLimit(t *testing.T) {
	tempDir := t.TempDir()
	uploader, err := NewUploader(tempDir, 50) // 50 bytes limit
	if err != nil {
		t.Fatalf("failed creating uploader: %v", err)
	}

	pngData := createTestPNG(100, 100) // Much larger than 50 bytes
	_, err = uploader.Save(bytes.NewReader(pngData), "big.png")
	if !errors.Is(err, ErrFileTooLarge) && !errors.Is(err, ErrInvalidImageData) {
		t.Fatalf("expected ErrFileTooLarge, got: %v", err)
	}
}

func TestUploadCorruptedImageData(t *testing.T) {
	tempDir := t.TempDir()
	uploader, err := NewUploader(tempDir, 1<<20)
	if err != nil {
		t.Fatalf("failed creating uploader: %v", err)
	}

	// First 8 bytes of PNG magic header, followed by junk
	corrupt := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("A", 100))
	_, err = uploader.Save(bytes.NewReader(corrupt), "corrupt.png")
	if !errors.Is(err, ErrInvalidImageData) {
		t.Fatalf("expected ErrInvalidImageData, got: %v", err)
	}

	// Confirm no orphan files left in storageDir
	files, _ := os.ReadDir(tempDir)
	if len(files) != 0 {
		t.Errorf("expected 0 files in storage dir after failed upload, found %d", len(files))
	}
}

func TestUploadDeduplicatesOnlyMatchingContentHash(t *testing.T) {
	tempDir := t.TempDir()
	uploader, err := NewUploader(tempDir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	pngData := createTestPNG(16, 16)
	first, err := uploader.Save(bytes.NewReader(pngData), "one.png")
	if err != nil {
		t.Fatal(err)
	}
	second, err := uploader.Save(bytes.NewReader(pngData), "two.png")
	if err != nil {
		t.Fatalf("duplicate content was rejected: %v", err)
	}
	if first.StoragePath != second.StoragePath {
		t.Fatalf("duplicate storage paths differ: %q/%q", first.StoragePath, second.StoragePath)
	}
	if err := os.WriteFile(first.StoragePath, []byte("tampered"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := uploader.Save(bytes.NewReader(pngData), "three.png"); err == nil {
		t.Fatal("tampered content-addressed file was accepted")
	}
}

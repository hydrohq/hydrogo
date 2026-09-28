package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweepStaleFilesRemovesOnlyOldTemporaryFiles(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	for _, name := range []string{"upload-orphan", "metadata-clean-orphan", "image-123.tmp", "keep.jpg"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(filepath.Join(root, "upload-orphan"), now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(root, "metadata-clean-orphan"), now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(root, "image-123.tmp"), now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	count, err := SweepStaleFiles(context.Background(), root, MinimumStaleAge, now)
	if err != nil || count != 3 {
		t.Fatalf("sweep count=%d err=%v", count, err)
	}
	if _, err := os.Stat(filepath.Join(root, "keep.jpg")); err != nil {
		t.Fatalf("ordinary media removed: %v", err)
	}
}

func TestSweepStaleFilesRejectsShortRetentionAndSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	if _, err := SweepStaleFiles(context.Background(), root, time.Hour, time.Now()); err == nil {
		t.Fatal("short retention accepted")
	}
	target := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "upload-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if n, err := SweepStaleFiles(context.Background(), root, MinimumStaleAge, time.Now().Add(48*time.Hour)); err != nil || n != 0 {
		t.Fatalf("symlink sweep count=%d err=%v", n, err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target removed: %v", err)
	}
}

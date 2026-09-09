package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRestorePathsRejectsNestedDestination(t *testing.T) {
	root := t.TempDir()
	for _, destination := range []string{filepath.Join(root, "staging"), root} {
		if err := validateRestorePaths(root, destination); err == nil {
			t.Fatalf("destination %q was accepted", destination)
		}
	}
	outside := filepath.Join(filepath.Dir(root), "restore-target")
	if err := validateRestorePaths(root, outside); err != nil {
		t.Fatalf("outside destination rejected: %v", err)
	}
}

func TestValidateRestorePathsRejectsSymlinkRoots(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "source-link")
	if err := os.Symlink(source, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := validateRestorePaths(link, filepath.Join(root, "destination")); err == nil {
		t.Fatal("symlink source root was accepted")
	}
}

func TestValidateRestorePathsAllowsSiblingWithSharedPrefix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "snapshot")
	destination := filepath.Join(filepath.Dir(root), "snapshot-copy")
	if err := validateRestorePaths(root, destination); err != nil {
		t.Fatalf("sibling destination rejected: %v", err)
	}
}

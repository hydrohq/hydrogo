package backup

import (
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

func TestValidateRestorePathsAllowsSiblingWithSharedPrefix(t *testing.T) {
	root := filepath.Join(t.TempDir(), "snapshot")
	destination := filepath.Join(filepath.Dir(root), "snapshot-copy")
	if err := validateRestorePaths(root, destination); err != nil {
		t.Fatalf("sibling destination rejected: %v", err)
	}
}

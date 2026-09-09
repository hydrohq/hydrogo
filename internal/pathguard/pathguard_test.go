package pathguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectSymlinkComponents(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(target, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := RejectSymlinkComponents(filepath.Join(link, "nested")); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("nested symlink error = %v", err)
	}
}

func TestRejectSymlinkComponentsAllowsMissingTail(t *testing.T) {
	root := t.TempDir()
	if err := RejectSymlinkComponents(filepath.Join(root, "new", "nested")); err != nil {
		t.Fatalf("missing path rejected: %v", err)
	}
}

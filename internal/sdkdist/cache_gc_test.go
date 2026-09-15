package sdkdist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneCacheDryRunAndApply(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "v1.0.0", "hydro-linux-amd64")
	current := filepath.Join(root, "v2.0.0", "hydro-linux-amd64")
	if err := os.MkdirAll(filepath.Dir(old), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(current), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current, []byte("current"), 0o640); err != nil {
		t.Fatal(err)
	}
	planned, err := PruneCache(root, map[string]bool{"v2.0.0": true}, true)
	if err != nil || len(planned) != 1 || planned[0].Version != "v1.0.0" {
		t.Fatalf("dry run = %#v, err=%v", planned, err)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatalf("dry run removed artifact: %v", err)
	}
	if _, err := PruneCache(root, map[string]bool{"v2.0.0": true}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old artifact still exists: %v", err)
	}
	if _, err := os.Stat(current); err != nil {
		t.Fatalf("current artifact removed: %v", err)
	}
}

func TestListCacheRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "v1.0.0")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(root, "outside"), link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ListCache(root); err == nil {
		t.Fatal("expected symlink rejection")
	}
}

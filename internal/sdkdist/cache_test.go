package sdkdist

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCachePathIsDeterministicAndScoped(t *testing.T) {
	root := t.TempDir()
	path, err := CachePath(root, "hydro", "v1.2.3", Target{OS: "linux", Arch: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "v1.2.3", "hydro-linux-amd64")
	if path != want {
		t.Fatalf("cache path = %q, want %q", path, want)
	}
	if !strings.HasPrefix(path, root) {
		t.Fatalf("cache path escaped root: %q", path)
	}
}

func TestCachePathRejectsTraversalVersion(t *testing.T) {
	for _, version := range []string{"../v1", "v1/../x", "", ".."} {
		if _, err := CachePath(t.TempDir(), "hydro", version, Target{OS: "linux", Arch: "amd64"}); err == nil {
			t.Fatalf("unsafe version accepted: %q", version)
		}
	}
}

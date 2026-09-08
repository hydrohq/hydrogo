package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneSnapshotsKeepsNewestAndPinned(t *testing.T) {
	root := t.TempDir()
	created := []struct {
		name string
		when time.Time
	}{
		{name: "snapshot-old", when: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "snapshot-pinned", when: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{name: "snapshot-new", when: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
	}
	for _, item := range created {
		dir := filepath.Join(root, item.name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		manifest := Manifest{Version: 1, CreatedAt: item.when, Database: FileDigest{Path: "database", SHA256: "0", Bytes: 0}}
		if err := WriteManifest(filepath.Join(dir, "manifest.json"), manifest); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := PruneSnapshots(root, SnapshotRetention{Keep: 1, Pinned: map[string]struct{}{"snapshot-pinned": {}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != "snapshot-old" {
		t.Fatalf("removed snapshots = %#v", removed)
	}
	for _, name := range []string{"snapshot-pinned", "snapshot-new"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("retained snapshot %s missing: %v", name, err)
		}
	}
}

func TestPruneSnapshotsFailsClosedOnMalformedManifest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "partial")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := PruneSnapshots(root, SnapshotRetention{Keep: 1}); err == nil {
		t.Fatal("malformed snapshot was pruned")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
}

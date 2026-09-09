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

func TestPlanSnapshotPruneDoesNotMutate(t *testing.T) {
	root := t.TempDir()
	writeSnapshot(t, root, "old", time.Unix(1, 0))
	writeSnapshot(t, root, "new", time.Unix(2, 0))
	planned, err := PlanSnapshotPrune(root, SnapshotRetention{Keep: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(planned) != 1 || planned[0] != "old" {
		t.Fatalf("planned removals = %#v", planned)
	}
	if _, err := os.Stat(filepath.Join(root, "old")); err != nil {
		t.Fatalf("plan mutated snapshot tree: %v", err)
	}
}

func TestPlanSnapshotPruneRejectsSymlinkRoot(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "snapshots")
	if err := os.MkdirAll(target, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "snapshots-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := PlanSnapshotPrune(link, SnapshotRetention{Keep: 1}); err == nil {
		t.Fatal("symlinked retention root was accepted")
	}
}

func TestSnapshotRetentionRejectsEmptyRoot(t *testing.T) {
	for _, operation := range []struct {
		name string
		call func() error
	}{
		{name: "plan", call: func() error {
			_, err := PlanSnapshotPrune("   ", SnapshotRetention{Keep: 1})
			return err
		}},
		{name: "prune", call: func() error {
			_, err := PruneSnapshots("", SnapshotRetention{Keep: 1})
			return err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.call(); err == nil {
				t.Fatal("empty retention root was accepted")
			}
		})
	}
}

func writeSnapshot(t *testing.T, root, name string, created time.Time) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(filepath.Join(dir, "manifest.json"), Manifest{Version: 1, CreatedAt: created, Database: FileDigest{Path: "database", SHA256: "0", Bytes: 0}}); err != nil {
		t.Fatal(err)
	}
}

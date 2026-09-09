package backup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadRestoreMarkerRejectsOversizedInput(t *testing.T) {
	root := t.TempDir()
	markerPath := filepath.Join(root, "RESTORE_QUARANTINED.json")
	if err := os.WriteFile(markerPath, bytes.Repeat([]byte("x"), maxRestoreMarkerBytes+1), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := readRestoreMarker(root); err == nil {
		t.Fatal("oversized restore marker was accepted")
	}
}

func TestActivateStagingMediaRetainsPreviousRoot(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	active := filepath.Join(root, "active")
	rollback := filepath.Join(root, "rollback")
	if err := os.MkdirAll(filepath.Join(staging, "media"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(active, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(active, "old.css"), []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	marker := RestoreMarker{ManifestVersion: 1, Controls: RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}}
	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "RESTORE_QUARANTINED.json"), data, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "media", "new.css"), []byte("new"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "database"), []byte("database"), 0o640); err != nil {
		t.Fatal(err)
	}
	databaseDigest, err := digestFile(filepath.Join(staging, "database"), "database")
	if err != nil {
		t.Fatal(err)
	}
	mediaDigest, err := digestFile(filepath.Join(staging, "media", "new.css"), "media/new.css")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(filepath.Join(staging, "manifest.json"), Manifest{Version: 1, Database: databaseDigest, Media: []FileDigest{mediaDigest}}); err != nil {
		t.Fatal(err)
	}

	if err := ActivateStagingMedia(staging, active, rollback); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(active, "media", "new.css")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(active, "RESTORE_ACTIVE.json")); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(filepath.Join(rollback, "old.css"))
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old" {
		t.Fatalf("rollback asset = %q", old)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatalf("staging root still exists: %v", err)
	}
}

func TestActivateStagingMediaRequiresQuarantineControls(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	if err := os.MkdirAll(staging, 0o750); err != nil {
		t.Fatal(err)
	}
	marker := RestoreMarker{ManifestVersion: 1}
	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "RESTORE_QUARANTINED.json"), data, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := ActivateStagingMedia(staging, filepath.Join(root, "active"), filepath.Join(root, "rollback")); err == nil {
		t.Fatal("activation without restore controls succeeded")
	}
}

func TestActivateStagingMediaRequiresVerifiedManifest(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	if err := os.MkdirAll(staging, 0o750); err != nil {
		t.Fatal(err)
	}
	marker := RestoreMarker{ManifestVersion: 1, Controls: RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}}
	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "RESTORE_QUARANTINED.json"), data, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := ActivateStagingMedia(staging, filepath.Join(root, "active"), filepath.Join(root, "rollback")); err == nil {
		t.Fatal("activation accepted staging without a verified manifest")
	}
}

func TestActivateStagingMediaValidatesDistinctPathsAndMissingMarker(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	active := filepath.Join(root, "active")
	rollback := filepath.Join(root, "rollback")

	// Missing staging directory / marker
	if err := ActivateStagingMedia(staging, active, rollback); err == nil {
		t.Fatal("expected error when staging directory does not exist")
	}

	// Empty paths
	if err := ActivateStagingMedia("", active, rollback); err == nil {
		t.Fatal("expected error with empty staging path")
	}

	// Identical paths
	if err := ActivateStagingMedia(staging, staging, rollback); err == nil {
		t.Fatal("expected error with identical staging and active paths")
	}
	if err := ActivateStagingMedia(staging, active, active); err == nil {
		t.Fatal("expected error with identical active and rollback paths")
	}
}

func TestActivateStagingMediaRejectsSymlinkRoots(t *testing.T) {
	root := t.TempDir()
	realStaging := filepath.Join(root, "real-staging")
	if err := os.MkdirAll(realStaging, 0o750); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(root, "staging-link")
	if err := os.Symlink(realStaging, symlink); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := ActivateStagingMedia(symlink, filepath.Join(root, "active"), filepath.Join(root, "rollback")); err == nil {
		t.Fatal("activation accepted a symlink staging root")
	}
}

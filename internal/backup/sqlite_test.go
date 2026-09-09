package backup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSnapshotSQLiteManifestVerifiesAndDetectsTampering(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "source.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY); INSERT INTO test DEFAULT VALUES"); err != nil {
		t.Fatal(err)
	}
	mediaDir := filepath.Join(root, "media")
	if err := os.MkdirAll(filepath.Join(mediaDir, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "nested", "a.txt"), []byte("asset"), 0o640); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "snapshot")
	manifest, err := SnapshotSQLite(context.Background(), db, mediaDir, destination)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Media) != 1 || manifest.Media[0].Path != "media/nested/a.txt" {
		t.Fatalf("unexpected media manifest: %#v", manifest.Media)
	}
	stored, err := ReadManifest(filepath.Join(destination, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Database != manifest.Database || len(stored.Media) != len(manifest.Media) {
		t.Fatalf("stored manifest differs: %#v/%#v", stored, manifest)
	}
	if err := Verify(destination, manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "media", "nested", "unlisted.txt"), []byte("unlisted"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Verify(destination, manifest); err == nil {
		t.Fatal("unlisted snapshot file passed verification")
	}
	if err := os.Remove(filepath.Join(destination, "media", "nested", "unlisted.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "media", "nested", "a.txt"), []byte("tampered"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Verify(destination, manifest); err == nil {
		t.Fatal("tampered media passed verification")
	}
}

func TestSnapshotSQLiteEnforcesPreflightBudgets(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY); INSERT INTO test DEFAULT VALUES"); err != nil {
		t.Fatal(err)
	}
	mediaDir := filepath.Join(root, "media")
	if err := os.MkdirAll(mediaDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "asset.bin"), []byte("12345"), 0o640); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "snapshot")
	if _, err := SnapshotSQLiteWithLimits(context.Background(), db, mediaDir, destination, SnapshotLimits{MaxMediaBytes: 4}); !errors.Is(err, ErrSnapshotBudget) {
		t.Fatalf("media budget error = %v", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("over-budget destination still exists: %v", err)
	}
	if _, err := SnapshotSQLiteWithLimits(context.Background(), db, mediaDir, destination, SnapshotLimits{MaxDuration: time.Nanosecond}); !errors.Is(err, ErrSnapshotBudget) {
		t.Fatalf("duration budget error = %v", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("timed-out destination still exists: %v", err)
	}
}

func TestSnapshotSQLiteRejectsSymlinkedMedia(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY); INSERT INTO test DEFAULT VALUES"); err != nil {
		t.Fatal(err)
	}
	mediaDir := filepath.Join(root, "media")
	if err := os.MkdirAll(mediaDir, 0o750); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(mediaDir, "linked.txt")); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	destination := filepath.Join(root, "snapshot")
	if _, err := SnapshotSQLite(context.Background(), db, mediaDir, destination); err == nil {
		t.Fatal("snapshot accepted a symlinked media entry")
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed snapshot destination still exists: %v", err)
	}
}

func TestSnapshotSQLiteRejectsMissingMediaRoot(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY); INSERT INTO test DEFAULT VALUES"); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "snapshot")
	if _, err := SnapshotSQLite(context.Background(), db, filepath.Join(root, "missing-media"), destination); err == nil {
		t.Fatal("snapshot accepted a missing media root")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("failed snapshot destination still exists: %v", err)
	}
}

func TestSnapshotSQLiteRejectsEmptyMediaRoot(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "snapshot")
	if _, err := SnapshotSQLite(context.Background(), db, "  ", destination); err == nil {
		t.Fatal("snapshot accepted an empty media root")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("failed snapshot destination still exists: %v", err)
	}
}

func TestRestoreToStagingRequiresQuarantineControls(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "media"), 0o750); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(root, "snapshot")
	manifest, err := SnapshotSQLite(context.Background(), db, filepath.Join(root, "media"), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := RestoreToStaging(snapshot, filepath.Join(root, "staging"), manifest, RestoreControls{}); err == nil {
		t.Fatal("restore without controls succeeded")
	}
	staging := filepath.Join(root, "staging")
	if err := RestoreToStaging(snapshot, staging, manifest, RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staging, "RESTORE_QUARANTINED.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreToStagingPreservesRetainedMedia(t *testing.T) {
	root := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(root, "source.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE pages (id TEXT PRIMARY KEY); INSERT INTO pages VALUES ('page-1')"); err != nil {
		t.Fatal(err)
	}
	mediaDir := filepath.Join(root, "media")
	if err := os.MkdirAll(filepath.Join(mediaDir, "retained"), 0o750); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(mediaDir, "retained", "hero.webp")
	if err := os.WriteFile(asset, []byte("immutable-media"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(root, "snapshot")
	manifest, err := SnapshotSQLite(context.Background(), db, mediaDir, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(root, "staging")
	if err := RestoreToStaging(snapshot, staging, manifest, RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(filepath.Join(staging, "media", "retained", "hero.webp"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "immutable-media" {
		t.Fatalf("restored media = %q", restored)
	}
}

func TestBackupRejectsManifestPathTraversal(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.json")
	manifest := Manifest{Version: 1, Database: FileDigest{Path: "database", SHA256: "", Bytes: 0}, Media: []FileDigest{{Path: "media/../outside", SHA256: "", Bytes: 0}}}
	if err := WriteManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(manifestPath); err == nil {
		t.Fatal("path traversal manifest was accepted")
	}
	if err := Verify(root, manifest); err == nil {
		t.Fatal("path traversal manifest passed verification")
	}
}

func TestVerifyRequiresMatchingStoredManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "database"), []byte("database"), 0o640); err != nil {
		t.Fatal(err)
	}
	digest, err := digestFile(filepath.Join(root, "database"), "database")
	if err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Version: 1, Database: digest}
	if err := WriteManifest(filepath.Join(root, "manifest.json"), manifest); err != nil {
		t.Fatal(err)
	}
	if err := Verify(root, manifest); err != nil {
		t.Fatalf("matching stored manifest rejected: %v", err)
	}
	stored := manifest
	stored.CreatedAt = time.Now().UTC()
	if err := WriteManifest(filepath.Join(root, "manifest.json"), stored); err != nil {
		t.Fatal(err)
	}
	if err := Verify(root, manifest); err == nil {
		t.Fatal("mismatched stored manifest unexpectedly verified")
	}
	if err := os.Remove(filepath.Join(root, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if err := Verify(root, manifest); err == nil {
		t.Fatal("snapshot without stored manifest unexpectedly verified")
	}
}

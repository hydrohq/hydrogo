package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

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
	if err := Verify(destination, manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "media", "nested", "a.txt"), []byte("tampered"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Verify(destination, manifest); err == nil {
		t.Fatal("tampered media passed verification")
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

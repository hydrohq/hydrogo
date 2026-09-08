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

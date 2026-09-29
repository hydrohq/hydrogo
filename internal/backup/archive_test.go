package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotArchiveRoundTripAndBudget(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "media"), 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte("snapshot contents")
	if err := os.WriteFile(filepath.Join(root, "media", "asset.bin"), want, 0o600); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err := WriteSnapshotArchive(context.Background(), &archive, root); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "restored")
	if err := ExtractSnapshotArchive(context.Background(), archivePath, destination, 1<<20); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "media", "asset.bin"))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("restored asset = %q, %v", got, err)
	}
	tooSmall := filepath.Join(t.TempDir(), "too-small")
	if err := ExtractSnapshotArchive(context.Background(), archivePath, tooSmall, 1); err != ErrSnapshotBudget {
		t.Fatalf("budget error = %v", err)
	}
	if _, err := os.Stat(tooSmall); !os.IsNotExist(err) {
		t.Fatalf("partial destination was retained: %v", err)
	}
}

func TestSignedSnapshotArchiveRequiresTrustedKeyAndVerifiesManifest(t *testing.T) {
	root := t.TempDir()
	database := []byte("database")
	if err := os.WriteFile(filepath.Join(root, "database"), database, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(database)
	manifest := Manifest{Version: 1, CreatedAt: time.Now().UTC(), Database: FileDigest{Path: "database", SHA256: hex.EncodeToString(digest[:]), Bytes: int64(len(database))}}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), manifestBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	var archive bytes.Buffer
	if err := WriteSignedSnapshotArchive(context.Background(), &archive, root, privateKey); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "signed.tar.gz")
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "restore")
	if err := ExtractSignedSnapshotArchive(context.Background(), archivePath, destination, 1<<20, privateKey.Public().(ed25519.PublicKey)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "signature.json")); !os.IsNotExist(err) {
		t.Fatalf("signature envelope was retained in snapshot: %v", err)
	}
	wrongKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{8}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
	wrongDestination := filepath.Join(t.TempDir(), "wrong-key")
	if err := ExtractSignedSnapshotArchive(context.Background(), archivePath, wrongDestination, 1<<20, wrongKey); err == nil {
		t.Fatal("accepted an archive from an untrusted signing key")
	}
	if _, err := os.Stat(wrongDestination); !os.IsNotExist(err) {
		t.Fatalf("failed destination was retained: %v", err)
	}
}

func TestParseBackupSigningKey(t *testing.T) {
	seed := hex.EncodeToString(bytes.Repeat([]byte{4}, ed25519.SeedSize))
	if _, err := ParseSigningSeed(seed); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSigningSeed("invalid"); err == nil {
		t.Fatal("accepted invalid seed")
	}
	publicKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{4}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
	if _, err := ParseSigningPublicKey(hex.EncodeToString(publicKey)); err != nil {
		t.Fatal(err)
	}
}

func TestExtractSnapshotArchiveRejectsTraversalAndLinks(t *testing.T) {
	for _, header := range []*tar.Header{
		{Name: "../escape", Mode: 0o600, Size: 1, Typeflag: tar.TypeReg},
		{Name: "link", Mode: 0o777, Linkname: "target", Typeflag: tar.TypeSymlink},
	} {
		var payload bytes.Buffer
		gz := gzip.NewWriter(&payload)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size > 0 {
			if _, err := tw.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		archivePath := filepath.Join(t.TempDir(), "unsafe.tar.gz")
		if err := os.WriteFile(archivePath, payload.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(t.TempDir(), "restore")
		if err := ExtractSnapshotArchive(context.Background(), archivePath, destination, 1<<20); err == nil {
			t.Fatalf("accepted unsafe entry %q", header.Name)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatalf("partial destination was retained: %v", err)
		}
	}
}

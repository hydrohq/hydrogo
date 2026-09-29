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
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultArchiveExtractLimit int64 = 10 << 30
const maxArchiveEntries = 100_000

type SnapshotSummary struct {
	Name      string
	CreatedAt time.Time
	Bytes     int64
}

type archiveSignature struct {
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

// WriteSnapshotArchive streams a snapshot directory as a gzip-compressed tar archive.
func WriteSnapshotArchive(ctx context.Context, dst io.Writer, root string) error {
	return writeSnapshotArchive(ctx, dst, root, nil)
}

// WriteSignedSnapshotArchive adds an Ed25519 signature for the checksum manifest.
func WriteSignedSnapshotArchive(ctx context.Context, dst io.Writer, root string, privateKey ed25519.PrivateKey) error {
	if len(privateKey) != ed25519.PrivateKeySize {
		return errors.New("backup signing key is invalid")
	}
	if err := validateBackupRoot(root); err != nil {
		return err
	}
	manifest, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil || int64(len(manifest)) > MaxManifestBytes {
		return errors.New("backup manifest is unavailable")
	}
	parsedManifest, err := ReadManifest(filepath.Join(root, "manifest.json"))
	if err != nil || Verify(root, parsedManifest) != nil {
		return errors.New("backup snapshot is invalid")
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyHash := sha256.Sum256(publicKey)
	signature := archiveSignature{KeyID: hex.EncodeToString(keyHash[:]), PublicKey: hex.EncodeToString(publicKey), Signature: hex.EncodeToString(ed25519.Sign(privateKey, manifest))}
	encoded, err := json.Marshal(signature)
	if err != nil {
		return err
	}
	return writeSnapshotArchive(ctx, dst, root, encoded)
}

func writeSnapshotArchive(ctx context.Context, dst io.Writer, root string, signature []byte) error {
	if dst == nil {
		return errors.New("backup archive destination is required")
	}
	if err := validateBackupRoot(root); err != nil {
		return err
	}
	gz := gzip.NewWriter(dst)
	tarWriter := tar.NewWriter(gz)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("backup snapshot contains a symlink")
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err = tarWriter.WriteHeader(header); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tarWriter, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err == nil && len(signature) > 0 {
		header := &tar.Header{Name: "signature.json", Mode: 0o600, Size: int64(len(signature)), Typeflag: tar.TypeReg}
		if err = tarWriter.WriteHeader(header); err == nil {
			_, err = tarWriter.Write(signature)
		}
	}
	if closeErr := tarWriter.Close(); err == nil {
		err = closeErr
	}
	if closeErr := gz.Close(); err == nil {
		err = closeErr
	}
	return err
}

// ParseSigningSeed parses the 32-byte hexadecimal seed used for archive signing.
func ParseSigningSeed(seed string) (ed25519.PrivateKey, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(seed))
	if err != nil || len(decoded) != ed25519.SeedSize {
		return nil, errors.New("backup signing seed must be 64 hexadecimal characters")
	}
	return ed25519.NewKeyFromSeed(decoded), nil
}

// ParseSigningPublicKey parses the trusted 32-byte Ed25519 public key.
func ParseSigningPublicKey(encoded string) (ed25519.PublicKey, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, errors.New("backup verification key must be 64 hexadecimal characters")
	}
	return ed25519.PublicKey(decoded), nil
}

// ExtractSignedSnapshotArchive verifies the trusted key and manifest signature before returning.
func ExtractSignedSnapshotArchive(ctx context.Context, archivePath, destination string, maxBytes int64, trusted ed25519.PublicKey) error {
	if len(trusted) != ed25519.PublicKeySize {
		return errors.New("a trusted backup verification key is required")
	}
	if err := ExtractSnapshotArchive(ctx, archivePath, destination, maxBytes); err != nil {
		return err
	}
	cleanFailure := func(err error) error { _ = os.RemoveAll(destination); return err }
	data, err := os.ReadFile(filepath.Join(destination, "signature.json"))
	if err != nil || len(data) > 4096 {
		return cleanFailure(errors.New("backup archive signature is missing"))
	}
	var signature archiveSignature
	if err := json.Unmarshal(data, &signature); err != nil {
		return cleanFailure(errors.New("backup archive signature is invalid"))
	}
	publicKey, pubErr := hex.DecodeString(signature.PublicKey)
	sig, sigErr := hex.DecodeString(signature.Signature)
	keyHash := sha256.Sum256(trusted)
	if pubErr != nil || sigErr != nil || !bytes.Equal(publicKey, trusted) || signature.KeyID != hex.EncodeToString(keyHash[:]) || len(sig) != ed25519.SignatureSize {
		return cleanFailure(errors.New("backup archive signature does not match the trusted key"))
	}
	manifestPath := filepath.Join(destination, "manifest.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil || int64(len(manifest)) > MaxManifestBytes || !ed25519.Verify(trusted, manifest, sig) {
		return cleanFailure(errors.New("backup manifest signature verification failed"))
	}
	if err := os.Remove(filepath.Join(destination, "signature.json")); err != nil {
		return cleanFailure(err)
	}
	decodedManifest, err := ReadManifest(manifestPath)
	if err != nil {
		return cleanFailure(errors.New("backup manifest is invalid"))
	}
	if err := Verify(destination, decodedManifest); err != nil {
		return cleanFailure(errors.New("backup snapshot verification failed"))
	}
	return nil
}

// ExtractSnapshotArchive safely unpacks a bounded HydroGo snapshot archive into a new directory.
func ExtractSnapshotArchive(ctx context.Context, archivePath, destination string, maxBytes int64) error {
	if strings.TrimSpace(archivePath) == "" || strings.TrimSpace(destination) == "" {
		return errors.New("backup archive and destination are required")
	}
	if maxBytes <= 0 {
		maxBytes = DefaultArchiveExtractLimit
	}
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	gz, err := gzip.NewReader(archive)
	if err != nil {
		return errors.New("invalid backup archive")
	}
	defer gz.Close()
	archiveLimit := int64(maxArchiveEntries) * 1024
	if maxBytes > math.MaxInt64-archiveLimit {
		archiveLimit = math.MaxInt64
	} else {
		archiveLimit += maxBytes
	}
	limited := &io.LimitedReader{R: gz, N: archiveLimit}
	root, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		return fmt.Errorf("create restore destination: %w", err)
	}
	cleanFailure := func(err error) error { _ = os.RemoveAll(root); return err }
	tr := tar.NewReader(limited)
	seen := make(map[string]struct{})
	var total int64
	entries := 0
	for {
		if err := ctx.Err(); err != nil {
			return cleanFailure(err)
		}
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			if _, err := io.Copy(io.Discard, limited); err != nil || limited.N == 0 {
				return cleanFailure(errors.New("invalid backup archive"))
			}
			return nil
		}
		if err != nil {
			return cleanFailure(errors.New("invalid backup archive"))
		}
		entries++
		if entries > maxArchiveEntries {
			return cleanFailure(ErrSnapshotBudget)
		}
		name := filepath.Clean(filepath.FromSlash(header.Name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return cleanFailure(errors.New("backup archive contains an unsafe path"))
		}
		if _, exists := seen[name]; exists {
			return cleanFailure(errors.New("backup archive contains duplicate paths"))
		}
		seen[name] = struct{}{}
		path := filepath.Join(root, name)
		if rel, relErr := filepath.Rel(root, path); relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return cleanFailure(errors.New("backup archive contains an unsafe path"))
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if header.Size != 0 {
				return cleanFailure(errors.New("backup archive directory entry is invalid"))
			}
			if err := os.MkdirAll(path, 0o700); err != nil {
				return cleanFailure(err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > maxBytes-total {
				return cleanFailure(ErrSnapshotBudget)
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return cleanFailure(err)
			}
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return cleanFailure(err)
			}
			written, copyErr := io.CopyN(file, tr, header.Size)
			closeErr := file.Close()
			total += written
			if copyErr != nil {
				return cleanFailure(copyErr)
			}
			if closeErr != nil {
				return cleanFailure(closeErr)
			}
		default:
			return cleanFailure(errors.New("backup archive contains an unsupported entry"))
		}
	}
}

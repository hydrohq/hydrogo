// Package backup contains bounded, staging-first recovery primitives.
package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type Manifest struct {
	Version   int          `json:"version"`
	CreatedAt time.Time    `json:"created_at"`
	Database  FileDigest   `json:"database"`
	Media     []FileDigest `json:"media"`
}

type RestoreControls struct {
	SessionsRevoked   bool `json:"sessions_revoked"`
	EgressQuarantined bool `json:"egress_quarantined"`
	OldWritesBlocked  bool `json:"old_writes_blocked"`
}

type RestoreMarker struct {
	ManifestVersion int             `json:"manifest_version"`
	VerifiedAt      time.Time       `json:"verified_at"`
	Controls        RestoreControls `json:"controls"`
}

// SnapshotSQLite creates a consistent SQLite snapshot and hashes every media file.
// The caller is responsible for pausing application writes and background jobs.
func SnapshotSQLite(ctx context.Context, db *sql.DB, mediaDir, destination string) (Manifest, error) {
	if db == nil || strings.TrimSpace(destination) == "" {
		return Manifest{}, errors.New("database and destination are required")
	}
	mediaDir, err := filepath.Abs(mediaDir)
	if err != nil {
		return Manifest{}, err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return Manifest{}, err
	}
	if _, err := os.Stat(destination); err == nil {
		return Manifest{}, fmt.Errorf("destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	if err := os.MkdirAll(destination, 0o750); err != nil {
		return Manifest{}, err
	}
	dbDestination := filepath.Join(destination, "database")
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dbDestination); err != nil {
		return Manifest{}, fmt.Errorf("sqlite snapshot: %w", err)
	}
	dbDigest, err := digestFile(dbDestination, "database")
	if err != nil {
		return Manifest{}, err
	}
	media, err := digestTree(mediaDir)
	if err != nil {
		return Manifest{}, err
	}
	for _, file := range media {
		source := filepath.Join(destination, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(source), 0o750); err != nil {
			return Manifest{}, err
		}
		if err := copyFile(filepath.Join(mediaDir, filepath.FromSlash(strings.TrimPrefix(file.Path, "media/"))), source); err != nil {
			return Manifest{}, err
		}
	}
	return Manifest{Version: 1, CreatedAt: time.Now().UTC(), Database: dbDigest, Media: media}, nil
}

func WriteManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o640)
}

func ReadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	if manifest.Version != 1 || manifest.Database.Path != "database" {
		return Manifest{}, errors.New("unsupported backup manifest")
	}
	return manifest, nil
}

// Verify checks a snapshot directory without activating it.
func Verify(root string, manifest Manifest) error {
	db, err := digestFile(filepath.Join(root, manifest.Database.Path), "database")
	if err != nil {
		return err
	}
	if db.SHA256 != manifest.Database.SHA256 || db.Bytes != manifest.Database.Bytes {
		return errors.New("database checksum mismatch")
	}
	for _, expected := range manifest.Media {
		actual, err := digestFile(filepath.Join(root, expected.Path), expected.Path)
		if err != nil {
			return err
		}
		if actual.SHA256 != expected.SHA256 || actual.Bytes != expected.Bytes {
			return fmt.Errorf("media checksum mismatch: %s", expected.Path)
		}
	}
	return nil
}

// RestoreToStaging verifies a snapshot and copies it to a new, inactive target.
// It never replaces an existing directory and writes a marker that operators can
// require before enabling traffic or workers.
func RestoreToStaging(source, destination string, manifest Manifest, controls RestoreControls) error {
	if !controls.SessionsRevoked || !controls.EgressQuarantined || !controls.OldWritesBlocked {
		return errors.New("restore controls are incomplete")
	}
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	destination, err = filepath.Abs(destination)
	if err != nil {
		return err
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("restore destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := Verify(source, manifest); err != nil {
		return fmt.Errorf("verify restore source: %w", err)
	}
	if err := copyTree(source, destination); err != nil {
		return err
	}
	marker := RestoreMarker{ManifestVersion: manifest.Version, VerifiedAt: time.Now().UTC(), Controls: controls}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destination, "RESTORE_QUARANTINED.json"), append(data, '\n'), 0o640)
}

func digestTree(root string) ([]FileDigest, error) {
	files := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	sort.Strings(files)
	result := make([]FileDigest, 0, len(files))
	for _, rel := range files {
		digest, err := digestFile(filepath.Join(root, filepath.FromSlash(rel)), filepath.ToSlash(filepath.Join("media", rel)))
		if err != nil {
			return nil, err
		}
		result = append(result, digest)
	}
	return result, nil
}

func digestFile(path, manifestPath string) (FileDigest, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileDigest{}, err
	}
	defer file.Close()
	hash := sha256.New()
	bytes, err := io.Copy(hash, file)
	if err != nil {
		return FileDigest{}, err
	}
	return FileDigest{Path: filepath.ToSlash(manifestPath), SHA256: hex.EncodeToString(hash.Sum(nil)), Bytes: bytes}, nil
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported backup entry: %s", rel)
		}
		return copyFile(path, target)
	})
}

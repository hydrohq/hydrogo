// Package backup contains bounded, staging-first recovery primitives.
package backup

import (
	"bytes"
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

// SnapshotLimits bounds the work performed before a snapshot can be used.
// Zero values leave the corresponding limit disabled.
type SnapshotLimits struct {
	MaxDatabaseBytes int64
	MaxMediaBytes    int64
	MaxDuration      time.Duration
}

var ErrSnapshotBudget = errors.New("backup snapshot budget exceeded")

// SnapshotSQLite creates a consistent SQLite snapshot and hashes every media file.
// The caller is responsible for pausing application writes and background jobs.
func SnapshotSQLite(ctx context.Context, db *sql.DB, mediaDir, destination string) (Manifest, error) {
	return SnapshotSQLiteWithLimits(ctx, db, mediaDir, destination, SnapshotLimits{})
}

// SnapshotSQLiteWithLimits performs a preflight size check and enforces a
// caller-provided wall-clock budget. A failed or over-budget snapshot never
// leaves a usable destination directory behind.
func SnapshotSQLiteWithLimits(ctx context.Context, db *sql.DB, mediaDir, destination string, limits SnapshotLimits) (Manifest, error) {
	if db == nil || strings.TrimSpace(mediaDir) == "" || strings.TrimSpace(destination) == "" {
		return Manifest{}, errors.New("database, media directory, and destination are required")
	}
	started := time.Now()
	checkBudget := func() error {
		if limits.MaxDuration > 0 && time.Since(started) > limits.MaxDuration {
			return ErrSnapshotBudget
		}
		return nil
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
	cleanup := func() {
		_ = os.RemoveAll(destination)
	}
	databaseBytes, err := sqliteDatabaseBytes(ctx, db)
	if err != nil {
		cleanup()
		return Manifest{}, err
	}
	if limits.MaxDatabaseBytes > 0 && databaseBytes > limits.MaxDatabaseBytes {
		cleanup()
		return Manifest{}, fmt.Errorf("%w: database is %d bytes, limit is %d", ErrSnapshotBudget, databaseBytes, limits.MaxDatabaseBytes)
	}
	media, err := digestTree(mediaDir)
	if err != nil {
		cleanup()
		return Manifest{}, err
	}
	var mediaBytes int64
	for _, file := range media {
		mediaBytes += file.Bytes
	}
	if limits.MaxMediaBytes > 0 && mediaBytes > limits.MaxMediaBytes {
		cleanup()
		return Manifest{}, fmt.Errorf("%w: media is %d bytes, limit is %d", ErrSnapshotBudget, mediaBytes, limits.MaxMediaBytes)
	}
	if err := checkBudget(); err != nil {
		cleanup()
		return Manifest{}, err
	}
	dbDestination := filepath.Join(destination, "database")
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dbDestination); err != nil {
		cleanup()
		return Manifest{}, fmt.Errorf("sqlite snapshot: %w", err)
	}
	dbDigest, err := digestFile(dbDestination, "database")
	if err != nil {
		cleanup()
		return Manifest{}, err
	}
	if limits.MaxDatabaseBytes > 0 && dbDigest.Bytes > limits.MaxDatabaseBytes {
		cleanup()
		return Manifest{}, fmt.Errorf("%w: snapshot database is %d bytes, limit is %d", ErrSnapshotBudget, dbDigest.Bytes, limits.MaxDatabaseBytes)
	}
	if err := checkBudget(); err != nil {
		cleanup()
		return Manifest{}, err
	}
	for _, file := range media {
		source := filepath.Join(destination, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(source), 0o750); err != nil {
			cleanup()
			return Manifest{}, err
		}
		if err := copyFile(filepath.Join(mediaDir, filepath.FromSlash(strings.TrimPrefix(file.Path, "media/"))), source); err != nil {
			cleanup()
			return Manifest{}, err
		}
		if err := checkBudget(); err != nil {
			cleanup()
			return Manifest{}, err
		}
	}
	manifest := Manifest{Version: 1, CreatedAt: time.Now().UTC(), Database: dbDigest, Media: media}
	if err := WriteManifest(filepath.Join(destination, "manifest.json"), manifest); err != nil {
		cleanup()
		return Manifest{}, err
	}
	// Media is hashed before copying so the source can be measured and bounded.
	// Verify the assembled tree again before declaring success; a concurrent
	// writer or a copy-time failure must never produce a seemingly valid backup
	// whose manifest does not match its contents.
	if err := Verify(destination, manifest); err != nil {
		cleanup()
		return Manifest{}, fmt.Errorf("verify snapshot: %w", err)
	}
	return manifest, nil
}

func sqliteDatabaseBytes(ctx context.Context, db *sql.DB) (int64, error) {
	var pages, pageSize int64
	if err := db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); err != nil {
		return 0, fmt.Errorf("read sqlite page count: %w", err)
	}
	if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, fmt.Errorf("read sqlite page size: %w", err)
	}
	return pages * pageSize, nil
}

func WriteManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".hydro-manifest-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o640); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
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
	if err := validateMediaManifest(manifest.Media); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Verify checks a snapshot directory without activating it.
func Verify(root string, manifest Manifest) error {
	if manifest.Version != 1 || manifest.Database.Path != "database" {
		return errors.New("unsupported backup manifest")
	}
	if err := validateMediaManifest(manifest.Media); err != nil {
		return err
	}
	stored, err := ReadManifest(filepath.Join(root, "manifest.json"))
	if err != nil {
		return fmt.Errorf("read snapshot manifest: %w", err)
	}
	wantBytes, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode snapshot manifest: %w", err)
	}
	gotBytes, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("encode stored snapshot manifest: %w", err)
	}
	if !bytes.Equal(wantBytes, gotBytes) {
		return errors.New("snapshot manifest mismatch")
	}
	db, err := digestFile(filepath.Join(root, manifest.Database.Path), "database")
	if err != nil {
		return err
	}
	if db.SHA256 != manifest.Database.SHA256 || db.Bytes != manifest.Database.Bytes {
		return errors.New("database checksum mismatch")
	}
	expectedEntries := map[string]struct{}{"database": {}, "manifest.json": {}, "RESTORE_QUARANTINED.json": {}, "RESTORE_ACTIVE.json": {}}
	for _, expected := range manifest.Media {
		actual, err := digestFile(filepath.Join(root, expected.Path), expected.Path)
		if err != nil {
			return err
		}
		if actual.SHA256 != expected.SHA256 || actual.Bytes != expected.Bytes {
			return fmt.Errorf("media checksum mismatch: %s", expected.Path)
		}
		expectedEntries[expected.Path] = struct{}{}
	}
	if err := validateSnapshotEntries(root, expectedEntries); err != nil {
		return err
	}
	return nil
}

func validateSnapshotEntries(root string, expected map[string]struct{}) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported snapshot symlink: %s", rel)
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported snapshot entry: %s", rel)
		}
		if _, ok := expected[rel]; !ok {
			return fmt.Errorf("snapshot entry is not listed in manifest: %s", rel)
		}
		return nil
	})
}

func validateMediaManifest(files []FileDigest) error {
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		path := filepath.ToSlash(file.Path)
		clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		if path == "" || path != clean || !strings.HasPrefix(path, "media/") || strings.Contains(path, "../") || filepath.IsAbs(filepath.FromSlash(path)) {
			return fmt.Errorf("invalid media manifest path: %q", file.Path)
		}
		if _, ok := seen[path]; ok {
			return fmt.Errorf("duplicate media manifest path: %q", path)
		}
		seen[path] = struct{}{}
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
	if err := validateRestorePaths(source, destination); err != nil {
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
		_ = os.RemoveAll(destination)
		return err
	}
	marker := RestoreMarker{ManifestVersion: manifest.Version, VerifiedAt: time.Now().UTC(), Controls: controls}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	if err := os.WriteFile(filepath.Join(destination, "RESTORE_QUARANTINED.json"), append(data, '\n'), 0o640); err != nil {
		_ = os.RemoveAll(destination)
		return err
	}
	return nil
}

func digestTree(root string) ([]FileDigest, error) {
	files := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == root && !info.IsDir() {
			return fmt.Errorf("media root is not a directory: %s", root)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			return fmt.Errorf("unsupported media symlink: %s", filepath.ToSlash(rel))
		}
		if info.IsDir() {
			return nil
		}
		if info.Mode().IsRegular() {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		}
		return fmt.Errorf("unsupported media entry: %s", path)
	})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("media root does not exist: %s", root)
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
	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
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

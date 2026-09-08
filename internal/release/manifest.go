// Package release builds deterministic release manifests and signatures.
package release

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrSecretArtifact = errors.New("secret-like artifact cannot enter a release manifest")

type Artifact struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	FormatVersion      int               `json:"format_version"`
	Edition            string            `json:"edition"`
	CoreVersion        string            `json:"core_version"`
	SDKVersion         string            `json:"sdk_version"`
	SourceRevision     string            `json:"source_revision"`
	Toolchain          string            `json:"toolchain"`
	Modules            []string          `json:"modules"`
	MigrationChecksums map[string]string `json:"migration_checksums"`
	Licenses           []string          `json:"licenses"`
	Artifacts          []Artifact        `json:"artifacts"`
}

type Signed struct {
	Manifest  json.RawMessage `json:"manifest"`
	PublicKey string          `json:"public_key"`
	Signature string          `json:"signature"`
}

type Input struct {
	Edition            string
	CoreVersion        string
	SDKVersion         string
	SourceRevision     string
	Toolchain          string
	Modules            []string
	MigrationChecksums map[string]string
}

func Build(root string, input Input) (Manifest, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(input.Edition) == "" || strings.TrimSpace(input.CoreVersion) == "" {
		return Manifest{}, errors.New("release identity is required")
	}
	manifest := Manifest{FormatVersion: 1, Edition: input.Edition, CoreVersion: input.CoreVersion, SDKVersion: input.SDKVersion, SourceRevision: input.SourceRevision, Toolchain: input.Toolchain, Modules: append([]string(nil), input.Modules...), MigrationChecksums: cloneMap(input.MigrationChecksums)}
	sort.Strings(manifest.Modules)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if secretLike(relative) {
			return ErrSecretArtifact
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		manifest.Artifacts = append(manifest.Artifacts, Artifact{Path: relative, Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])})
		if strings.EqualFold(filepath.Base(relative), "LICENSE") || strings.HasPrefix(strings.ToUpper(filepath.Base(relative)), "LICENSE.") {
			manifest.Licenses = append(manifest.Licenses, relative)
		}
		return nil
	})
	if err != nil {
		return Manifest{}, err
	}
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].Path < manifest.Artifacts[j].Path })
	sort.Strings(manifest.Licenses)
	return manifest, nil
}

func Marshal(manifest Manifest) ([]byte, error) {
	if manifest.FormatVersion != 1 || strings.TrimSpace(manifest.Edition) == "" || strings.TrimSpace(manifest.CoreVersion) == "" {
		return nil, errors.New("invalid release manifest")
	}
	return json.MarshalIndent(manifest, "", "  ")
}

func Sign(manifest Manifest, privateKey ed25519.PrivateKey) (Signed, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return Signed{}, errors.New("invalid signing key")
	}
	data, err := Marshal(manifest)
	if err != nil {
		return Signed{}, err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return Signed{Manifest: data, PublicKey: base64.RawStdEncoding.EncodeToString(publicKey), Signature: base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, data))}, nil
}

func Verify(signed Signed) (Manifest, error) {
	publicKey, err := base64.RawStdEncoding.DecodeString(signed.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return Manifest{}, errors.New("invalid release public key")
	}
	signature, err := base64.RawStdEncoding.DecodeString(signed.Signature)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(publicKey), signed.Manifest, signature) {
		return Manifest{}, errors.New("release signature verification failed")
	}
	var manifest Manifest
	if err := json.Unmarshal(signed.Manifest, &manifest); err != nil {
		return Manifest{}, err
	}
	if _, err := Marshal(manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid signed manifest: %w", err)
	}
	return manifest, nil
}

func secretLike(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return name == ".env" || strings.HasSuffix(name, ".key") || strings.HasSuffix(name, ".pem") || strings.Contains(name, "secret") || strings.Contains(name, "credential")
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

// RetainAssets verifies the asset inventory from a release manifest and
// materializes it in a shared directory used during the rollback window.
// Existing files are immutable: an identical file is kept, while a different
// file at the same path is rejected.
func RetainAssets(sourceRoot string, manifest Manifest, destinationRoot string) error {
	sourceRoot, destinationRoot, err := absoluteRoots(sourceRoot, destinationRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destinationRoot, 0o750); err != nil {
		return err
	}
	for _, artifact := range manifest.Artifacts {
		if !isRetainedAsset(artifact.Path) {
			continue
		}
		source, err := safeArtifactPath(sourceRoot, artifact.Path)
		if err != nil {
			return err
		}
		if err := verifyArtifact(source, artifact); err != nil {
			return fmt.Errorf("asset %s: %w", artifact.Path, err)
		}
		destination, err := safeArtifactPath(destinationRoot, artifact.Path)
		if err != nil {
			return err
		}
		if existing, err := os.Stat(destination); err == nil {
			if existing.IsDir() {
				return fmt.Errorf("retained asset path is a directory: %s", artifact.Path)
			}
			if err := verifyArtifact(destination, artifact); err != nil {
				return fmt.Errorf("retained asset conflict %s: %w", artifact.Path, err)
			}
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
			return err
		}
		temporary, err := os.CreateTemp(filepath.Dir(destination), ".hydro-retain-*")
		if err != nil {
			return err
		}
		temporaryPath := temporary.Name()
		ok := false
		defer func() {
			if !ok {
				_ = os.Remove(temporaryPath)
			}
		}()
		if err := copyBytes(temporary, source); err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Chmod(0o640); err != nil {
			_ = temporary.Close()
			return err
		}
		if err := temporary.Close(); err != nil {
			return err
		}
		if err := os.Rename(temporaryPath, destination); err != nil {
			return err
		}
		ok = true
	}
	return nil
}

// VerifyRetainedAssets checks every retained asset in a manifest without
// changing the shared directory.
func VerifyRetainedAssets(root string, manifest Manifest) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	for _, artifact := range manifest.Artifacts {
		if !isRetainedAsset(artifact.Path) {
			continue
		}
		path, err := safeArtifactPath(root, artifact.Path)
		if err != nil {
			return err
		}
		if err := verifyArtifact(path, artifact); err != nil {
			return fmt.Errorf("asset %s: %w", artifact.Path, err)
		}
	}
	return nil
}

func absoluteRoots(sourceRoot, destinationRoot string) (string, string, error) {
	if strings.TrimSpace(sourceRoot) == "" || strings.TrimSpace(destinationRoot) == "" {
		return "", "", errors.New("source and destination roots are required")
	}
	sourceRoot, err := filepath.Abs(sourceRoot)
	if err != nil {
		return "", "", err
	}
	destinationRoot, err = filepath.Abs(destinationRoot)
	if err != nil {
		return "", "", err
	}
	return sourceRoot, destinationRoot, nil
}

func isRetainedAsset(path string) bool {
	return strings.HasPrefix(path, "static/") || strings.HasPrefix(path, "assets/") || strings.HasPrefix(path, "public/assets/")
}

func safeArtifactPath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("invalid artifact path: %s", relative)
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("artifact path escapes root: %s", relative)
	}
	return filepath.Join(root, clean), nil
}

func verifyArtifact(path string, expected Artifact) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if int64(len(data)) != expected.Size {
		return fmt.Errorf("size mismatch: got %d want %d", len(data), expected.Size)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != expected.SHA256 {
		return errors.New("checksum mismatch")
	}
	return nil
}

func copyBytes(destination *os.File, source string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	_, err = io.Copy(destination, input)
	return err
}

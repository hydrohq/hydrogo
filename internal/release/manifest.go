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

package adminstudio

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

const (
	maxAdminExtensionManifestBytes = 1 << 20
	maxAdminExtensionKeyBytes      = 64 << 10
	maxAdminExtensionFiles         = 128
)

var ErrInvalidAdminExtensionBundle = errors.New("invalid admin extension bundle")

// LoadSignedAdminExtensions loads operator-approved signed manifest files. An
// empty directory disables the feature; no routes or background work are
// created in that case.
func LoadSignedAdminExtensions(manifestDir, trustedKeysFile string) (ExtensionRegistry, error) {
	if strings.TrimSpace(manifestDir) == "" {
		return NewExtensionRegistry(nil)
	}
	if strings.TrimSpace(trustedKeysFile) == "" {
		return ExtensionRegistry{}, fmt.Errorf("%w: trusted keys file is required", ErrInvalidAdminExtensionBundle)
	}
	trustedKeys, err := readTrustedAdminExtensionKeys(trustedKeysFile)
	if err != nil {
		return ExtensionRegistry{}, err
	}
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return ExtensionRegistry{}, fmt.Errorf("%w: read manifest directory: %v", ErrInvalidAdminExtensionBundle, err)
	}
	if len(entries) > maxAdminExtensionFiles {
		return ExtensionRegistry{}, fmt.Errorf("%w: too many manifest files", ErrInvalidAdminExtensionBundle)
	}
	manifests := make([]extensionv1.AdminExtensionManifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(manifestDir, entry.Name())
		if filepath.Clean(path) == filepath.Clean(trustedKeysFile) {
			continue
		}
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return ExtensionRegistry{}, fmt.Errorf("%w: manifest %q is not a regular file", ErrInvalidAdminExtensionBundle, entry.Name())
		}
		if info.Size() <= 0 || info.Size() > maxAdminExtensionManifestBytes {
			return ExtensionRegistry{}, fmt.Errorf("%w: manifest %q exceeds size limit", ErrInvalidAdminExtensionBundle, entry.Name())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return ExtensionRegistry{}, fmt.Errorf("%w: read manifest %q: %v", ErrInvalidAdminExtensionBundle, entry.Name(), err)
		}
		manifest, err := extensionv1.ParseSignedAdminExtensionManifest(data, trustedKeys)
		if err != nil {
			return ExtensionRegistry{}, fmt.Errorf("%w: verify manifest %q: %v", ErrInvalidAdminExtensionBundle, entry.Name(), err)
		}
		manifests = append(manifests, manifest)
	}
	return NewExtensionRegistry(manifests)
}

func readTrustedAdminExtensionKeys(path string) (map[string]ed25519.PublicKey, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxAdminExtensionKeyBytes {
		return nil, fmt.Errorf("%w: invalid trusted keys file", ErrInvalidAdminExtensionBundle)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read trusted keys: %v", ErrInvalidAdminExtensionBundle, err)
	}
	var encoded map[string]string
	if err := json.Unmarshal(data, &encoded); err != nil || len(encoded) == 0 {
		return nil, fmt.Errorf("%w: malformed trusted keys", ErrInvalidAdminExtensionBundle)
	}
	keys := make(map[string]ed25519.PublicKey, len(encoded))
	for keyID, value := range encoded {
		if strings.TrimSpace(keyID) == "" {
			return nil, fmt.Errorf("%w: empty key id", ErrInvalidAdminExtensionBundle)
		}
		decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		}
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("%w: invalid key %q", ErrInvalidAdminExtensionBundle, keyID)
		}
		keys[keyID] = ed25519.PublicKey(decoded)
	}
	return keys, nil
}

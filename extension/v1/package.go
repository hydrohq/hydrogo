package v1

import (
	"archive/zip"
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
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	ExtensionPackageFormatVersion = 1
	MaxExtensionPackageFiles      = 4096
	MaxExtensionPackageFileBytes  = 64 << 20
)

var (
	ErrInvalidExtensionPackage          = errors.New("invalid extension package")
	ErrInvalidExtensionPackageSignature = errors.New("invalid extension package signature")
)

// PackageFile is the signed integrity record for one package asset. The
// package archive is deliberately outside this contract; the signed record is
// portable across zip, tar, or deployment-specific storage.
type PackageFile struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// PackageMetadata is the canonical content covered by SignedPackage.Signature.
type PackageMetadata struct {
	FormatVersion int                    `json:"format_version"`
	Manifest      AdminExtensionManifest `json:"manifest"`
	Files         []PackageFile          `json:"files"`
}

// SignedPackage is the vendor-produced metadata envelope accepted by a
// customer runtime. The key advertised by the package is never trusted unless
// it exists in the operator's trusted key map.
type SignedPackage struct {
	Package   PackageMetadata `json:"package"`
	KeyID     string          `json:"key_id"`
	Signature string          `json:"signature"`
}

// HashFiles creates a deterministic flat integrity list from package assets.
// Input keys are archive-relative paths and values are their exact bytes.
func HashFiles(assets map[string][]byte) ([]PackageFile, error) {
	if len(assets) == 0 || len(assets) > MaxExtensionPackageFiles {
		return nil, fmt.Errorf("%w: invalid file count", ErrInvalidExtensionPackage)
	}
	files := make([]PackageFile, 0, len(assets))
	for name, data := range assets {
		clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
		if clean == "." || clean != name || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, ":") {
			return nil, fmt.Errorf("%w: unsafe asset path", ErrInvalidExtensionPackage)
		}
		if int64(len(data)) > MaxExtensionPackageFileBytes {
			return nil, fmt.Errorf("%w: asset too large", ErrInvalidExtensionPackage)
		}
		digest := sha256.Sum256(data)
		files = append(files, PackageFile{Path: clean, Bytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:])})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (p PackageMetadata) Validate() error {
	if p.FormatVersion != ExtensionPackageFormatVersion || len(p.Files) == 0 || len(p.Files) > MaxExtensionPackageFiles {
		return ErrInvalidExtensionPackage
	}
	if err := p.Manifest.Validate(); err != nil {
		return fmt.Errorf("%w: manifest: %v", ErrInvalidExtensionPackage, err)
	}
	previous := ""
	for _, file := range p.Files {
		if file.Path == "" || file.Path != path.Clean(file.Path) || strings.HasPrefix(file.Path, "../") || strings.HasPrefix(file.Path, "/") || file.Bytes < 0 || file.Bytes > MaxExtensionPackageFileBytes || len(file.SHA256) != sha256.Size*2 {
			return ErrInvalidExtensionPackage
		}
		if previous != "" && previous >= file.Path {
			return ErrInvalidExtensionPackage
		}
		if _, err := hex.DecodeString(file.SHA256); err != nil {
			return ErrInvalidExtensionPackage
		}
		previous = file.Path
	}
	return nil
}

func SignPackage(metadata PackageMetadata, keyID string, privateKey ed25519.PrivateKey) (SignedPackage, error) {
	if metadata.Validate() != nil || keyID == "" || len(privateKey) != ed25519.PrivateKeySize {
		return SignedPackage{}, ErrInvalidExtensionPackageSignature
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return SignedPackage{}, ErrInvalidExtensionPackageSignature
	}
	return SignedPackage{Package: metadata, KeyID: keyID, Signature: base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))}, nil
}

func (s SignedPackage) Verify(trustedKeys map[string]ed25519.PublicKey) (PackageMetadata, error) {
	if s.Package.Validate() != nil || s.KeyID == "" || strings.TrimSpace(s.Signature) == "" {
		return PackageMetadata{}, ErrInvalidExtensionPackageSignature
	}
	key, ok := trustedKeys[s.KeyID]
	if !ok || len(key) != ed25519.PublicKeySize {
		return PackageMetadata{}, ErrInvalidExtensionPackageSignature
	}
	signature, err := base64.RawStdEncoding.DecodeString(s.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return PackageMetadata{}, ErrInvalidExtensionPackageSignature
	}
	payload, err := json.Marshal(s.Package)
	if err != nil || !ed25519.Verify(key, payload, signature) {
		return PackageMetadata{}, ErrInvalidExtensionPackageSignature
	}
	return s.Package, nil
}

// VerifyAssets checks the exact bytes extracted from a verified package.
func (p PackageMetadata) VerifyAssets(assets map[string][]byte) error {
	files, err := HashFiles(assets)
	if err != nil || len(files) != len(p.Files) {
		return ErrInvalidExtensionPackage
	}
	for i := range files {
		if files[i] != p.Files[i] {
			return ErrInvalidExtensionPackage
		}
	}
	return nil
}

// WriteArchive writes a deterministic .hx archive. The signed package record
// is stored as package.json and assets are stored at their archive-relative
// paths. No executable entrypoint is implied by this format.
func WriteArchive(destination string, signed SignedPackage, assets map[string][]byte) error {
	if signed.Package.Validate() != nil || len(signed.Signature) == 0 || signed.KeyID == "" {
		return ErrInvalidExtensionPackage
	}
	if err := signed.Package.VerifyAssets(assets); err != nil {
		return err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- operator-chosen package paths; the destination is created exclusively (O_EXCL), assets are regular files from the named directory
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	metadata, err := json.Marshal(signed)
	if err != nil {
		return err
	}
	if err := writeZipEntry(archive, "package.json", metadata); err != nil {
		return err
	}
	names := make([]string, 0, len(assets))
	for name := range assets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := writeZipEntry(archive, name, assets[name]); err != nil {
			return err
		}
	}
	return archive.Close()
}

// ReadArchive reads a bounded .hx archive without trusting paths or executing
// any contents. Signature verification is intentionally a separate operation.
func ReadArchive(source string) (SignedPackage, map[string][]byte, error) {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return SignedPackage{}, nil, err
	}
	defer archive.Close()
	var signed SignedPackage
	assets := make(map[string][]byte)
	seenPackage := false
	for _, entry := range archive.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		clean := path.Clean(strings.ReplaceAll(entry.Name, "\\", "/"))
		if clean != entry.Name || clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, ":") {
			return SignedPackage{}, nil, ErrInvalidExtensionPackage
		}
		reader, err := entry.Open()
		if err != nil {
			return SignedPackage{}, nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(reader, MaxExtensionPackageFileBytes+1))
		_ = reader.Close()
		if readErr != nil || int64(len(data)) > MaxExtensionPackageFileBytes {
			return SignedPackage{}, nil, ErrInvalidExtensionPackage
		}
		if clean == "package.json" {
			if seenPackage {
				return SignedPackage{}, nil, ErrInvalidExtensionPackage
			}
			seenPackage = true
			if err := json.Unmarshal(data, &signed); err != nil {
				return SignedPackage{}, nil, ErrInvalidExtensionPackage
			}
			continue
		}
		if _, exists := assets[clean]; exists {
			return SignedPackage{}, nil, ErrInvalidExtensionPackage
		}
		assets[clean] = data
	}
	if signed.Package.Validate() != nil || signed.KeyID == "" || signed.Signature == "" || signed.Package.VerifyAssets(assets) != nil {
		return SignedPackage{}, nil, ErrInvalidExtensionPackage
	}
	return signed, assets, nil
}

func writeZipEntry(archive *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(0o600)
	header.Modified = zipEpoch
	entry, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = entry.Write(data)
	return err
}

// LoadAssets reads regular files beneath root using stable slash-separated
// archive paths. Symlinks are rejected so a package cannot escape its root.
func LoadAssets(root string) (map[string][]byte, error) {
	assets := make(map[string][]byte)
	// Reads go through the root, which refuses to leave it even if a path
	// component is swapped for a symlink after the walk looked at it.
	confined, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer confined.Close()
	err = filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink asset", ErrInvalidExtensionPackage)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%w: non-regular asset", ErrInvalidExtensionPackage)
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		data, err := confined.ReadFile(rel)
		if err != nil {
			return err
		}
		assets[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	return assets, nil
}

var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

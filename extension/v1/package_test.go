package v1

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

func TestExtensionPackageSignsSortedAssetHashesAndVerifies(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manifest := validAdminManifest()
	files, err := HashFiles(map[string][]byte{"assets/app.js": []byte("app"), "manifest.json": []byte("manifest")})
	if err != nil {
		t.Fatal(err)
	}
	metadata := PackageMetadata{FormatVersion: ExtensionPackageFormatVersion, Manifest: manifest, Files: files}
	signed, err := SignPackage(metadata, "vendor-2026", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := signed.Verify(map[string]ed25519.PublicKey{"vendor-2026": publicKey})
	if err != nil {
		t.Fatal(err)
	}
	if err := verified.VerifyAssets(map[string][]byte{"assets/app.js": []byte("app"), "manifest.json": []byte("manifest")}); err != nil {
		t.Fatal(err)
	}
	if err := verified.VerifyAssets(map[string][]byte{"assets/app.js": []byte("tampered"), "manifest.json": []byte("manifest")}); !errors.Is(err, ErrInvalidExtensionPackage) {
		t.Fatalf("tampered assets error = %v", err)
	}
}

func TestHashFilesRejectsUnsafeOrDuplicateShape(t *testing.T) {
	for _, name := range []string{"../secret", "/absolute", `dir\\file`, "C:secret"} {
		if _, err := HashFiles(map[string][]byte{name: []byte("x")}); !errors.Is(err, ErrInvalidExtensionPackage) {
			t.Fatalf("path %q error = %v", name, err)
		}
	}
}

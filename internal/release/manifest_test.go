package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestIsDeterministicAndSigned(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.bin"), []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "LICENSE"), []byte("license"), 0o644); err != nil {
		t.Fatal(err)
	}
	input := Input{Edition: "client", CoreVersion: "0.1.0", SDKVersion: "1.0.0-candidate", SourceRevision: "abc", Toolchain: "go1.27.0", Modules: []string{"news", "site"}, MigrationChecksums: map[string]string{"m1": "hash"}}
	first, err := Build(root, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(root, input)
	if err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := Marshal(first)
	secondBytes, _ := Marshal(second)
	if string(firstBytes) != string(secondBytes) {
		t.Fatal("manifest changed between identical builds")
	}
	_, private, _ := ed25519.GenerateKey(rand.Reader)
	signed, err := Sign(first, private)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := Verify(signed)
	if err != nil || verified.Edition != "client" {
		t.Fatalf("verified manifest = %+v/%v", verified, err)
	}
	signed.Manifest[0] ^= 1
	if _, err := Verify(signed); err == nil {
		t.Fatal("tampered manifest signature was accepted")
	}
}

func TestManifestRejectsSecretArtifact(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "prod.pem"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, Input{Edition: "client", CoreVersion: "0.1.0"}); !errors.Is(err, ErrSecretArtifact) {
		t.Fatalf("secret artifact error = %v", err)
	}
}

func TestManifestProvenanceAttestationBindsSubject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hydro"), []byte("binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := Build(root, Input{Edition: "client", CoreVersion: "1.0.0", SourceRevision: "abc", Toolchain: "go1.27"})
	if err != nil {
		t.Fatal(err)
	}
	_, private, _ := ed25519.GenerateKey(rand.Reader)
	signed, err := Attest(manifest, "ci/github-actions", private)
	if err != nil {
		t.Fatal(err)
	}
	attestation, err := VerifyAttestation(signed, manifest)
	if err != nil || attestation.Builder != "ci/github-actions" {
		t.Fatalf("attestation = %#v/%v", attestation, err)
	}
	manifest.CoreVersion = "1.0.1"
	if _, err := VerifyAttestation(signed, manifest); err == nil {
		t.Fatal("attestation accepted a changed manifest")
	}
}

func TestRetainAssetsVerifiesInventoryAndRejectsReplacement(t *testing.T) {
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "static", "release"), 0o750); err != nil {
		t.Fatal(err)
	}
	assetPath := filepath.Join(source, "static", "release", "app.123.css")
	if err := os.WriteFile(assetPath, []byte("old-release-css"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "main.bin"), []byte("binary"), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest, err := Build(source, Input{Edition: "client", CoreVersion: "0.1.0"})
	if err != nil {
		t.Fatal(err)
	}
	shared := t.TempDir()
	if err := RetainAssets(source, manifest, shared); err != nil {
		t.Fatal(err)
	}
	if err := VerifyRetainedAssets(shared, manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "static", "release", "app.123.css"), []byte("replacement"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := RetainAssets(source, manifest, shared); err == nil {
		t.Fatal("changed retained asset was accepted")
	}
}

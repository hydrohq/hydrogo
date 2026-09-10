package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyArtifactRejectsInvalidDeclaredSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.css")
	if err := os.WriteFile(path, []byte("css"), 0o640); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("css"))
	checksum := hex.EncodeToString(digest[:])
	for _, size := range []int64{-1, 1<<63 - 1} {
		if err := verifyArtifact(path, Artifact{Path: "static/asset.css", Size: size, SHA256: checksum}); err == nil {
			t.Fatalf("artifact with declared size %d was accepted", size)
		}
	}
}

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

func TestManifestRejectsSymlinkArtifact(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.bin")
	if err := os.WriteFile(target, []byte("outside"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "app.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := Build(root, Input{Edition: "client", CoreVersion: "0.1.0"}); err == nil {
		t.Fatal("manifest build followed a symlink artifact")
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
	if _, err := VerifyAttestationWithPublicKey(signed, manifest, private.Public().(ed25519.PublicKey)); err != nil {
		t.Fatalf("trusted-key attestation verification failed: %v", err)
	}
	manifest.CoreVersion = "1.0.1"
	if _, err := VerifyAttestation(signed, manifest); err == nil {
		t.Fatal("attestation accepted a changed manifest")
	}
	if _, err := VerifyAttestationWithPublicKey(signed, manifest, private.Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("changed manifest passed trusted-key attestation verification")
	}
	otherPublic, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := VerifyAttestationWithPublicKey(signed, manifest, otherPublic); err == nil {
		t.Fatal("attestation accepted an unrelated trusted key")
	}

	// Attest rejects empty builder or invalid key
	if _, err := Attest(manifest, "", private); err == nil {
		t.Fatal("expected error with empty builder")
	}
	if _, err := Attest(manifest, "ci", []byte("short")); err == nil {
		t.Fatal("expected error with short private key")
	}

	// VerifyAttestation rejects corrupted signature
	corrupted := signed
	corrupted.Signature = "invalid-base64"
	if _, err := VerifyAttestation(corrupted, manifest); err == nil {
		t.Fatal("expected error on invalid base64 signature")
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

func TestRetainAssetsValidatesRootsAndPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("secret"), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		FormatVersion: 1,
		Edition:       "site",
		CoreVersion:   "1.0.0",
		Artifacts: []Artifact{
			{Path: "static/../secret.txt", Size: 6, SHA256: "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b"},
		},
	}
	if err := RetainAssets("", manifest, "dest"); err == nil {
		t.Fatal("expected error with empty source root")
	}
	if err := RetainAssets(root, manifest, ""); err == nil {
		t.Fatal("expected error with empty destination root")
	}
	if err := RetainAssets(root, manifest, filepath.Join(root, "dest")); err == nil {
		t.Fatal("expected traversal path to be rejected even when target exists")
	}
	if err := VerifyRetainedAssets(root, manifest); err == nil {
		t.Fatal("expected error for escaping artifact path in VerifyRetainedAssets")
	}
	if err := VerifyRetainedAssets(" ", manifest); err == nil {
		t.Fatal("expected empty asset root to be rejected")
	}
}

func TestAssetRetentionRejectsSymlinkRoots(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "root-link")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	manifest := Manifest{FormatVersion: 1, Edition: "site", CoreVersion: "1.0.0"}
	if err := VerifyRetainedAssets(link, manifest); err == nil {
		t.Fatal("verify accepted a symlink asset root")
	}
	if err := RetainAssets(link, manifest, filepath.Join(parent, "destination")); err == nil {
		t.Fatal("retain accepted a symlink source root")
	}
}

func TestRetainAssetsRejectsSymlinkedDestinationParent(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "static"), 0o750); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(root, "static", "app.css")
	if err := os.WriteFile(asset, []byte("css"), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest, err := Build(root, Input{Edition: "site", CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "destination-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := RetainAssets(root, manifest, filepath.Join(link, "shared")); err == nil {
		t.Fatal("retain accepted a symlinked destination parent")
	}
}

func TestRetainAssetsRejectsDuplicatePath(t *testing.T) {
	root := t.TempDir()
	asset := filepath.Join(root, "static", "app.css")
	if err := os.MkdirAll(filepath.Dir(asset), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asset, []byte("css"), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{FormatVersion: 1, Edition: "site", CoreVersion: "1.0.0", Artifacts: []Artifact{
		{Path: "static/app.css", Size: 3, SHA256: "36e64f19f57a05c8cd5b6bf7eff72703b4bbab19def7832eb4e686e7fa482eef"},
		{Path: "static/app.css", Size: 3, SHA256: "36e64f19f57a05c8cd5b6bf7eff72703b4bbab19def7832eb4e686e7fa482eef"},
	}}
	if err := RetainAssets(root, manifest, filepath.Join(root, "dest")); err == nil {
		t.Fatal("duplicate retained asset path was accepted")
	}
}

func TestRetainAssetsRejectsSymlinkOutsideRoot(t *testing.T) {
	source := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.css")
	if err := os.WriteFile(outside, []byte("outside"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(source, "static", "link.css")
	if err := os.MkdirAll(filepath.Dir(link), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	digest := sha256.Sum256([]byte("outside"))
	manifest := Manifest{FormatVersion: 1, Edition: "site", CoreVersion: "1.0.0", Artifacts: []Artifact{{
		Path: "static/link.css", Size: int64(len("outside")), SHA256: hex.EncodeToString(digest[:]),
	}}}
	if err := RetainAssets(source, manifest, filepath.Join(source, "dest")); err == nil {
		t.Fatal("asset symlink escaping source root was accepted")
	}
}

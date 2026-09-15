package sdkdist

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/release"
)

func TestInstallArtifactIsVerifiedAndIdempotent(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "download")
	if err := os.WriteFile(source, []byte("runtime"), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest, err := release.Build(root, release.Input{Edition: "client", CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	var expected release.Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == "download" {
			expected = artifact
		}
	}
	destination := filepath.Join(root, "cache", "hydro")
	if err := InstallArtifact(source, destination, expected); err != nil {
		t.Fatalf("install artifact: %v", err)
	}
	if err := InstallArtifact(source, destination, expected); err != nil {
		t.Fatalf("idempotent install: %v", err)
	}
	if err := os.WriteFile(source, []byte("tampered"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := InstallArtifact(source, filepath.Join(root, "cache", "other"), expected); err == nil {
		t.Fatal("tampered source was accepted")
	}
}

func TestInstallArtifactNeverOverwritesDifferentDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "download")
	if err := os.WriteFile(source, []byte("runtime"), 0o640); err != nil {
		t.Fatal(err)
	}
	manifest, err := release.Build(root, release.Input{Edition: "client", CoreVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	var expected release.Artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.Path == "download" {
			expected = artifact
		}
	}
	destination := filepath.Join(root, "cache", "hydro")
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := InstallArtifact(source, destination, expected); err == nil {
		t.Fatal("different destination was overwritten")
	}
}

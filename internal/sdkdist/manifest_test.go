package sdkdist

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/release"
)

func TestResolveRuntimeArtifact(t *testing.T) {
	manifest := release.Manifest{Artifacts: []release.Artifact{
		{Path: "bin/hydro-linux-amd64", Size: 10, SHA256: "abc"},
		{Path: "bin/hydro-windows-amd64.exe", Size: 11, SHA256: "def"},
	}}
	artifact, err := ResolveRuntimeArtifact(manifest, "hydro", Target{OS: "windows", Arch: "amd64"})
	if err != nil || artifact.Path != "bin/hydro-windows-amd64.exe" {
		t.Fatalf("resolved artifact = %#v/%v", artifact, err)
	}
}

func TestResolveRuntimeArtifactFailsClosed(t *testing.T) {
	manifest := release.Manifest{Artifacts: []release.Artifact{{Path: "hydro-linux-amd64", Size: 0, SHA256: ""}}}
	if _, err := ResolveRuntimeArtifact(manifest, "hydro", Target{OS: "linux", Arch: "amd64"}); err == nil {
		t.Fatal("invalid artifact metadata was accepted")
	}
	if _, err := ResolveRuntimeArtifact(release.Manifest{}, "hydro", Target{OS: "linux", Arch: "amd64"}); err == nil {
		t.Fatal("missing runtime artifact was accepted")
	}
}

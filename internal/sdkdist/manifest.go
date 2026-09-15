package sdkdist

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/release"
)

// ResolveRuntimeArtifact selects exactly one canonical runtime artifact from a
// signed release manifest. The signature itself is verified by release.Verify;
// this function only performs target selection and ambiguity checks.
func ResolveRuntimeArtifact(manifest release.Manifest, product string, target Target) (release.Artifact, error) {
	want, err := ArtifactName(product, target)
	if err != nil {
		return release.Artifact{}, err
	}
	var match release.Artifact
	found := 0
	for _, artifact := range manifest.Artifacts {
		normalized := filepath.ToSlash(artifact.Path)
		if normalized == "" || filepath.IsAbs(artifact.Path) || strings.HasPrefix(normalized, "../") || strings.Contains(normalized, "/../") {
			return release.Artifact{}, fmt.Errorf("runtime artifact path is unsafe: %q", artifact.Path)
		}
		if filepath.Base(filepath.FromSlash(artifact.Path)) != want {
			continue
		}
		found++
		match = artifact
	}
	if found == 0 {
		return release.Artifact{}, fmt.Errorf("runtime artifact %q is not present in release manifest", want)
	}
	if found != 1 {
		return release.Artifact{}, fmt.Errorf("release manifest contains %d runtime artifacts named %q", found, want)
	}
	if strings.TrimSpace(match.SHA256) == "" || match.Size < 0 {
		return release.Artifact{}, fmt.Errorf("runtime artifact %q has invalid metadata", want)
	}
	return match, nil
}

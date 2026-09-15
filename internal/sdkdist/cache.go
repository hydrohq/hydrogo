package sdkdist

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+~-]{0,63}$`)

// CachePath returns the deterministic location for one runtime artifact. It
// does not create directories or execute the artifact.
func CachePath(root, product, version string, target Target) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("runtime cache root is required")
	}
	version = strings.TrimSpace(version)
	if !versionPattern.MatchString(version) || strings.Contains(version, "..") {
		return "", fmt.Errorf("invalid runtime version")
	}
	name, err := ArtifactName(product, target)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(rootAbs, version, name), nil
}

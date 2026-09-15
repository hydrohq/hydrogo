// Package sdkdist contains provider-neutral SDK/runtime distribution helpers.
package sdkdist

import (
	"fmt"
	"runtime"
	"strings"
)

type Target struct {
	OS   string
	Arch string
}

func CurrentTarget() Target {
	return Target{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

func ParseTarget(osName, arch string) (Target, error) {
	target := Target{OS: strings.ToLower(strings.TrimSpace(osName)), Arch: strings.ToLower(strings.TrimSpace(arch))}
	if target.OS == "" || target.Arch == "" {
		return Target{}, fmt.Errorf("target OS and architecture are required")
	}
	switch target.OS {
	case "linux", "windows", "darwin":
	default:
		return Target{}, fmt.Errorf("unsupported target OS %q", target.OS)
	}
	switch target.Arch {
	case "amd64", "arm64":
	default:
		return Target{}, fmt.Errorf("unsupported target architecture %q", target.Arch)
	}
	return target, nil
}

// ArtifactName returns the canonical runtime filename used by the broker and
// installers. It contains no user-controlled path components.
func ArtifactName(product string, target Target) (string, error) {
	product = strings.TrimSpace(product)
	if product == "" || strings.ContainsAny(product, `/\\`) || strings.Contains(product, "..") {
		return "", fmt.Errorf("invalid artifact product")
	}
	validated, err := ParseTarget(target.OS, target.Arch)
	if err != nil {
		return "", err
	}
	suffix := ""
	if validated.OS == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("%s-%s-%s%s", product, validated.OS, validated.Arch, suffix), nil
}

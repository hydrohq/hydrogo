package v1

import (
	"fmt"
	"strconv"
	"strings"
)

// Major returns the API major from a semantic version or candidate version.
func Major(version string) (int, error) {
	value := strings.TrimSpace(version)
	if value == "" {
		return 0, fmt.Errorf("SDK version is required")
	}
	parts := strings.SplitN(value, ".", 2)
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 1 {
		return 0, fmt.Errorf("invalid SDK version %q", version)
	}
	return major, nil
}

// CompatibleWith applies ADR-0010's same-major support rule. Candidate builds
// remain explicit and are not silently treated as stable releases.
func CompatibleWith(requested, runtime string) error {
	requestedMajor, err := Major(requested)
	if err != nil {
		return err
	}
	runtimeMajor, err := Major(runtime)
	if err != nil {
		return err
	}
	if requestedMajor != runtimeMajor {
		return fmt.Errorf("SDK major %d is incompatible with runtime major %d", requestedMajor, runtimeMajor)
	}
	return nil
}

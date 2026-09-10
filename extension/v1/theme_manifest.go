package v1

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type ThemeManifest struct {
	Name    string            `json:"name" yaml:"name"`
	Version string            `json:"version" yaml:"version"`
	Tokens  map[string]string `json:"tokens" yaml:"tokens"`
	Slots   []string          `json:"slots" yaml:"slots"`
	Assets  []string          `json:"assets" yaml:"assets"`
}

var ErrInvalidThemeManifest = errors.New("invalid theme manifest")
var themeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func (m ThemeManifest) Validate() error {
	if !themeName.MatchString(m.Name) || strings.TrimSpace(m.Version) == "" {
		return ErrInvalidThemeManifest
	}
	if len(m.Tokens) > 128 || len(m.Slots) > 64 || len(m.Assets) > 128 {
		return ErrInvalidThemeManifest
	}
	seen := map[string]struct{}{}
	for _, slot := range m.Slots {
		if !themeName.MatchString(slot) {
			return fmt.Errorf("%w: invalid slot", ErrInvalidThemeManifest)
		}
		if _, ok := seen[slot]; ok {
			return ErrInvalidThemeManifest
		}
		seen[slot] = struct{}{}
	}
	for key, value := range m.Tokens {
		if !themeName.MatchString(key) || strings.TrimSpace(value) == "" || len(value) > 256 {
			return ErrInvalidThemeManifest
		}
	}
	for _, asset := range m.Assets {
		if strings.TrimSpace(asset) == "" || strings.Contains(asset, "..") || strings.HasPrefix(asset, "/") {
			return ErrInvalidThemeManifest
		}
	}
	return nil
}

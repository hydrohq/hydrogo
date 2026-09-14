package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ThemeManifest struct {
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version" yaml:"version"`
	DesignSystem string            `json:"design_system" yaml:"design_system"`
	Tokens       map[string]string `json:"tokens" yaml:"tokens"`
	Slots        []string          `json:"slots" yaml:"slots"`
	Assets       []string          `json:"assets" yaml:"assets"`
}

var ErrInvalidThemeManifest = errors.New("invalid theme manifest")
var themeName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func (m ThemeManifest) Validate() error {
	if !themeName.MatchString(m.Name) || strings.TrimSpace(m.Version) == "" {
		return ErrInvalidThemeManifest
	}
	if m.DesignSystem != "" && (!themeName.MatchString(m.DesignSystem) || len(m.DesignSystem) > 64) {
		return fmt.Errorf("%w: invalid design system", ErrInvalidThemeManifest)
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
		if !safeThemePath(asset) {
			return ErrInvalidThemeManifest
		}
	}
	return nil
}

func safeThemePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.Contains(value, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

// ThemeBundle is a validated, filesystem-backed theme package. Only files
// declared by theme.json are exposed; callers choose when to render it.
type ThemeBundle struct {
	Manifest ThemeManifest
	Root     string
	Layout   string
}

type PageTemplateData struct {
	Title     string
	Slug      string
	Locale    string
	Direction string
	Content   template.HTML
	Slots     map[string]template.HTML
}

func LoadTheme(root, name string) (ThemeBundle, error) {
	if !themeName.MatchString(name) {
		return ThemeBundle{}, ErrInvalidThemeManifest
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return ThemeBundle{}, err
	}
	themeRoot := filepath.Join(base, name)
	rel, err := filepath.Rel(base, themeRoot)
	if err != nil || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return ThemeBundle{}, ErrInvalidThemeManifest
	}
	manifestBytes, err := os.ReadFile(filepath.Join(themeRoot, "theme.json"))
	if err != nil {
		return ThemeBundle{}, fmt.Errorf("read theme manifest: %w", err)
	}
	var manifest ThemeManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return ThemeBundle{}, fmt.Errorf("decode theme manifest: %w", err)
	}
	if manifest.Name != name {
		return ThemeBundle{}, ErrInvalidThemeManifest
	}
	if err := manifest.Validate(); err != nil {
		return ThemeBundle{}, err
	}
	layoutPath := filepath.Join(themeRoot, "layouts", "base.html")
	layout, err := os.ReadFile(layoutPath)
	if err != nil {
		return ThemeBundle{}, fmt.Errorf("read theme layout: %w", err)
	}
	for _, asset := range manifest.Assets {
		if _, err := os.Stat(filepath.Join(themeRoot, filepath.FromSlash(asset))); err != nil {
			return ThemeBundle{}, fmt.Errorf("theme asset %q: %w", asset, err)
		}
	}
	return ThemeBundle{Manifest: manifest, Root: themeRoot, Layout: string(layout)}, nil
}

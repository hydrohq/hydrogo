package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

var ErrInvalidAdminExtension = errors.New("invalid admin extension manifest")

// AdminExtensionManifest is the runtime-loadable, declarative contract for a
// customer-owned Admin Studio surface. It contains no Go handlers and cannot
// grant permissions; the host still enforces the listed permission boundary.
type AdminExtensionManifest struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	EntrypointURL string              `json:"entrypoint_url"`
	Surfaces      []AdminSurface      `json:"surfaces"`
	Subscriptions []AdminSubscription `json:"subscriptions,omitempty"`
}

// AdminSurface is rendered by the host as an isolated external UI surface,
// normally an iframe on the manifest origin. It is never compiled into Core.
type AdminSurface struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Path        string   `json:"path"`
	Placement   string   `json:"placement"`
	Permissions []string `json:"permissions,omitempty"`
}

// AdminSubscription declares an outbound event delivered by the host's
// configured webhook/event gateway. It does not execute customer code in Core.
type AdminSubscription struct {
	Event string `json:"event"`
}

// Validate checks the untrusted manifest before it reaches the admin shell.
// Only HTTPS entrypoints are accepted; every surface is constrained to that
// same origin and to a normalized path.
func (m AdminExtensionManifest) Validate() error {
	if !validIdentifier(m.ID) || strings.TrimSpace(m.Name) == "" || strings.TrimSpace(m.Version) == "" || len(m.Surfaces) == 0 {
		return ErrInvalidAdminExtension
	}
	entrypoint, err := url.Parse(strings.TrimSpace(m.EntrypointURL))
	if err != nil || entrypoint.Scheme != "https" || entrypoint.Host == "" || entrypoint.User != nil {
		return fmt.Errorf("%w: entrypoint must be an absolute HTTPS URL without credentials", ErrInvalidAdminExtension)
	}
	seen := make(map[string]struct{}, len(m.Surfaces))
	for _, surface := range m.Surfaces {
		if !validIdentifier(surface.ID) || strings.TrimSpace(surface.Label) == "" || strings.TrimSpace(surface.Placement) == "" || !validPath(surface.Path) {
			return ErrInvalidAdminExtension
		}
		if _, ok := seen[surface.ID]; ok {
			return ErrInvalidAdminExtension
		}
		seen[surface.ID] = struct{}{}
		for _, permission := range surface.Permissions {
			if !validPermission(permission) {
				return ErrInvalidAdminExtension
			}
		}
	}
	seenEvents := make(map[string]struct{}, len(m.Subscriptions))
	for _, subscription := range m.Subscriptions {
		if !validEvent(subscription.Event) {
			return ErrInvalidAdminExtension
		}
		if _, ok := seenEvents[subscription.Event]; ok {
			return ErrInvalidAdminExtension
		}
		seenEvents[subscription.Event] = struct{}{}
	}
	return nil
}

// ParseAdminExtensionManifest decodes and validates a customer-owned manifest
// before the host registers any navigation or surface.
func ParseAdminExtensionManifest(data []byte) (AdminExtensionManifest, error) {
	var manifest AdminExtensionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return AdminExtensionManifest{}, fmt.Errorf("%w: malformed JSON", ErrInvalidAdminExtension)
	}
	if err := manifest.Validate(); err != nil {
		return AdminExtensionManifest{}, err
	}
	return manifest, nil
}

// SurfaceURL resolves a validated surface against the manifest entrypoint.
func (m AdminExtensionManifest) SurfaceURL(surfaceID string) (string, bool) {
	if m.Validate() != nil {
		return "", false
	}
	for _, surface := range m.Surfaces {
		if surface.ID == surfaceID {
			base := strings.TrimRight(m.EntrypointURL, "/")
			return base + surface.Path, true
		}
	}
	return "", false
}

func validIdentifier(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func validPath(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.Contains(path, "..") && len(path) <= 256
}

func validPermission(value string) bool { return validIdentifier(value) && len(value) <= 96 }

func validEvent(value string) bool {
	if strings.TrimSpace(value) == "" || len(value) > 96 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// SortAdminExtensions returns a deterministic copy for registry rendering.
func SortAdminExtensions(manifests []AdminExtensionManifest) []AdminExtensionManifest {
	copyManifests := append([]AdminExtensionManifest(nil), manifests...)
	sort.Slice(copyManifests, func(i, j int) bool { return copyManifests[i].ID < copyManifests[j].ID })
	return copyManifests
}

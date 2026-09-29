package adminstudio

import (
	"errors"
	"sort"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

var ErrInvalidAdminExtensionRegistry = errors.New("invalid admin extension registry")

// ExtensionRegistry contains only validated declarative manifests. It never
// loads Go plugins or executes customer code inside the HydroGo process.
type ExtensionRegistry struct {
	manifests []extensionv1.AdminExtensionManifest
	channels  *ChannelManager
}

func NewExtensionRegistry(manifests []extensionv1.AdminExtensionManifest) (ExtensionRegistry, error) {
	copyManifests := extensionv1.SortAdminExtensions(manifests)
	seen := make(map[string]struct{}, len(copyManifests))
	seenPaths := make(map[string]struct{})
	for _, manifest := range copyManifests {
		if err := manifest.Validate(); err != nil {
			return ExtensionRegistry{}, ErrInvalidAdminExtensionRegistry
		}
		if _, ok := seen[manifest.ID]; ok {
			return ExtensionRegistry{}, ErrInvalidAdminExtensionRegistry
		}
		seen[manifest.ID] = struct{}{}
		for _, surface := range manifest.Surfaces {

			if _, ok := seenPaths[surface.Path]; ok {

				return ExtensionRegistry{}, ErrInvalidAdminExtensionRegistry

			}

			seenPaths[surface.Path] = struct{}{}

		}
	}
	return ExtensionRegistry{manifests: copyManifests, channels: NewChannelManager()}, nil
}

func (r ExtensionRegistry) BindChannel(id, challenge string, principal ChannelPrincipal) error {
	if r.channels == nil {
		return ErrExtensionChannelDenied
	}
	return r.channels.Bind(id, challenge, principal)
}

func (r ExtensionRegistry) RevokeChannel(id string, principal ChannelPrincipal) {
	if r.channels != nil {
		r.channels.Revoke(id, principal)
	}
}

func (r ExtensionRegistry) Manifests() []extensionv1.AdminExtensionManifest {
	return append([]extensionv1.AdminExtensionManifest(nil), r.manifests...)
}

func (r ExtensionRegistry) Manifest(id string) (extensionv1.AdminExtensionManifest, bool) {
	for _, manifest := range r.manifests {
		if manifest.ID == id {
			return manifest, true
		}
	}
	return extensionv1.AdminExtensionManifest{}, false
}

// SortExtensionRegistry is useful when a host merges manifests discovered from
// multiple deployment directories before constructing its final registry.
func SortExtensionRegistry(manifests []extensionv1.AdminExtensionManifest) []extensionv1.AdminExtensionManifest {
	result := extensionv1.SortAdminExtensions(manifests)
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

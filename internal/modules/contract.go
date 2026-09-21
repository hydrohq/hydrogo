package modules

import (
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/extension"
)

// Manifest is the declarative resource contract for a module. Resource names
// are metadata until the module is explicitly enabled; the registry never
// executes migrations or starts workers by itself.
type Manifest struct {
	ID           string
	Dependencies []string
	Routes       []RouteSpec
	Migrations   []string
	Assets       []string
	Workers      []WorkerSpec
}

type RouteSpec struct {
	Pattern string
	Public  bool
}

type WorkerSpec struct {
	ID string
}

// Declarative is an optional v1 extension. Legacy modules can continue to
// implement only extension.Module while Editions migrate incrementally.
type Declarative interface {
	ModuleManifest() Manifest
}

// RegisterIfEnabled prevents inactive modules from contributing routes or
// runtime resource metadata. It is the preferred registration path for an
// Edition composition root.
func (r *Registry) RegisterIfEnabled(m extension.Module, enabled bool) error {
	if !enabled {
		return nil
	}
	return r.Register(m)
}

// RegisterEnabled is an explicit alias for Editions that prefer a descriptive
// name at their composition root.
func (r *Registry) RegisterEnabled(m extension.Module, enabled bool) error {
	if !enabled {
		return nil
	}
	return r.Register(m)
}

// Manifests returns deterministic declarations for registered declarative
// modules. Unknown legacy modules are omitted rather than assigned resources.
func (r *Registry) Manifests() []Manifest {
	if r == nil {
		return nil
	}
	result := make([]Manifest, 0)
	for _, module := range r.modules {
		if declarative, ok := module.(Declarative); ok {
			manifest := declarative.ModuleManifest()
			manifest.ID = strings.ToLower(strings.TrimSpace(manifest.ID))
			result = append(result, manifest)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

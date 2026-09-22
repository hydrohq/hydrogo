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
	Admin        AdminManifest
}

// AdminManifest is optional presentation metadata for the authenticated
// admin surface. It contributes no route or authorization by itself; those
// remain explicit module contracts.
type AdminManifest struct {
	Route string
	Label string
	Icon  string
	Group string
	Order int
}

type AdminNavigationItem struct {
	ModuleID string
	Route    string
	Label    string
	Icon     string
	Group    string
	Order    int
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

// AdminNavigation returns only valid admin declarations from active,
// declarative modules. It is deterministic so generated HTML and cache keys
// remain stable across process restarts.
func (r *Registry) AdminNavigation() []AdminNavigationItem {
	if r == nil {
		return nil
	}
	items := make([]AdminNavigationItem, 0)
	for _, manifest := range r.Manifests() {
		admin := manifest.Admin
		if strings.TrimSpace(admin.Route) == "" || !strings.HasPrefix(admin.Route, "/") || strings.HasPrefix(admin.Route, "//") || strings.TrimSpace(admin.Label) == "" {
			continue
		}
		items = append(items, AdminNavigationItem{ModuleID: manifest.ID, Route: admin.Route, Label: admin.Label, Icon: admin.Icon, Group: admin.Group, Order: admin.Order})
	}
	sort.Slice(items, func(i, j int) bool {
		if strings.ToLower(items[i].Group) != strings.ToLower(items[j].Group) {
			return strings.ToLower(items[i].Group) < strings.ToLower(items[j].Group)
		}
		if items[i].Order != items[j].Order {
			return items[i].Order < items[j].Order
		}
		return items[i].ModuleID < items[j].ModuleID
	})
	return items
}

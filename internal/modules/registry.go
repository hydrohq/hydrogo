package modules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/extension"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

type Registry struct {
	modules map[string]extension.Module
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]extension.Module),
	}
}

func (r *Registry) Register(m extension.Module) error {
	if r == nil {
		return fmt.Errorf("cannot register module in nil registry")
	}
	if guards.IsNil(m) {
		return fmt.Errorf("cannot register nil module")
	}
	name := strings.ToLower(strings.TrimSpace(m.Name()))
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %q is already registered", name)
	}
	r.modules[name] = m
	return nil
}

func (r *Registry) Routes() []router.Route {
	if r == nil {
		return nil
	}
	var routes []router.Route
	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		m := r.modules[name]
		for _, mr := range m.Routes() {
			access := router.Protected
			if mr.Public {
				access = router.Public
			}
			routes = append(routes, router.Route{
				Pattern: mr.Pattern,
				Access:  access,
				Handler: mr.Handler,
			})
		}
	}
	return routes
}

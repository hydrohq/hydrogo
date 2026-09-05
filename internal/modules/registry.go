package modules

import (
	"fmt"

	"github.com/ardaozyildirim/hydrogo/extension"
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
	if m == nil {
		return fmt.Errorf("cannot register nil module")
	}
	name := m.Name()
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
	var routes []router.Route
	for _, m := range r.modules {
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

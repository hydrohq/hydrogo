package extension

import (
	"net/http"
)

// ModuleRoute defines an HTTP route contributed by a module.
//
// Deprecated: use extension/v1.ModuleRoute for new integrations.
type ModuleRoute struct {
	Pattern string
	Handler http.Handler
	Public  bool
}

// Module defines the contract for optional modular feature extensions in HydroGo.
//
// Deprecated: use extension/v1.Module for new integrations.
type Module interface {
	Name() string
	Routes() []ModuleRoute
}

package extension

import (
	"net/http"
)

// ModuleRoute defines an HTTP route contributed by a module.
type ModuleRoute struct {
	Pattern string
	Handler http.Handler
	Public  bool
}

// Module defines the contract for optional modular feature extensions in HydroGo.
type Module interface {
	Name() string
	Routes() []ModuleRoute
}

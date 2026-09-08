package v1

import "net/http"

type ModuleRoute struct {
	Pattern string
	Handler http.Handler
	Public  bool
}

type Module interface {
	Name() string
	Routes() []ModuleRoute
}

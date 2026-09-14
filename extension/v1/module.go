package v1

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/extension"
)

// ModuleRoute is the versioned spelling of the public module route contract.
// It aliases the original contract so existing editions remain source
// compatible while new editions can depend only on extension/v1.
type ModuleRoute = extension.ModuleRoute

// Module defines the smallest stable module boundary available to Client
// Editions. Module implementations must keep authorization and persistence
// policy in their own service layer.
type Module = extension.Module

// PublicRoute is a convenience constructor for a public route declaration.
// Protected routes should be declared explicitly with ModuleRoute so callers
// cannot accidentally infer access from a handler name.
func PublicRoute(pattern string, handler http.Handler) ModuleRoute {
	return ModuleRoute{Pattern: pattern, Handler: handler, Public: true}
}

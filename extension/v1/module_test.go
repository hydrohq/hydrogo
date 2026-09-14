package v1

import (
	"net/http"
	"testing"
)

type versionedModule struct{}

var _ Module = versionedModule{}

func (versionedModule) Name() string { return "example" }
func (versionedModule) Routes() []ModuleRoute {
	return []ModuleRoute{PublicRoute("GET /example", http.NotFoundHandler())}
}

func TestVersionedModuleContractAliasesPublicBoundary(t *testing.T) {
	var module Module = versionedModule{}
	if module.Name() != "example" || len(module.Routes()) != 1 || !module.Routes()[0].Public {
		t.Fatalf("unexpected versioned module contract: %#v", module.Routes())
	}
}

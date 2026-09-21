package modules

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension"
	"github.com/ardaozyildirim/hydrogo/internal/router"
)

type dummyModule struct {
	name   string
	routes []extension.ModuleRoute
}

type declarativeModule struct{ dummyModule }

func (declarativeModule) ModuleManifest() Manifest {
	return Manifest{ID: "sample", Dependencies: []string{"pages"}, Migrations: []string{"20260921_sample"}, Assets: []string{"sample.css"}, Workers: []WorkerSpec{{ID: "sample-index"}}}
}

func TestRegisterRejectsTypedNilModule(t *testing.T) {
	var module *dummyModule
	if err := NewRegistry().Register(module); err == nil {
		t.Fatal("typed nil module was accepted")
	}
}

func TestNilRegistryFailsClosedWithoutPanic(t *testing.T) {
	var reg *Registry
	if err := reg.Register(dummyModule{name: "sample"}); err == nil {
		t.Fatal("nil registry accepted a module")
	}
	if routes := reg.Routes(); routes != nil {
		t.Fatalf("nil registry returned routes: %#v", routes)
	}
}

func (d dummyModule) Name() string                    { return d.name }
func (d dummyModule) Routes() []extension.ModuleRoute { return d.routes }

func TestZeroOverheadWhenNoModulesRegistered(t *testing.T) {
	reg := NewRegistry()
	routes := reg.Routes()
	if len(routes) != 0 {
		t.Fatalf("expected 0 routes when no modules registered, got %d", len(routes))
	}
}

func TestModuleRouteRegistration(t *testing.T) {
	reg := NewRegistry()
	mod := dummyModule{
		name: "sample-feature",
		routes: []extension.ModuleRoute{
			{
				Pattern: "GET /sample",
				Public:  true,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			},
		},
	}

	if err := reg.Register(mod); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	routes := reg.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Pattern != "GET /sample" || routes[0].Access != router.Public {
		t.Fatalf("unexpected route configuration: %+v", routes[0])
	}

	r, err := router.New(routes, nil)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sample", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestRegisterCanonicalizesModuleNamesAndRejectsDuplicates(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(dummyModule{name: "  News  "}); err != nil {
		t.Fatalf("canonical registration failed: %v", err)
	}
	if err := reg.Register(dummyModule{name: "news"}); err == nil {
		t.Fatal("canonical duplicate module name was accepted")
	}
}

func TestRegisterRejectsWhitespaceOnlyModuleName(t *testing.T) {
	if err := NewRegistry().Register(dummyModule{name: " \t "}); err == nil {
		t.Fatal("whitespace-only module name was accepted")
	}
}

func TestInactiveDeclarativeModuleContributesNoResources(t *testing.T) {
	reg := NewRegistry()
	module := declarativeModule{dummyModule{name: "sample"}}
	if err := reg.RegisterIfEnabled(module, false); err != nil {
		t.Fatal(err)
	}
	if len(reg.Routes()) != 0 || len(reg.Manifests()) != 0 {
		t.Fatalf("inactive module contributed resources: routes=%v manifests=%v", reg.Routes(), reg.Manifests())
	}
	if err := reg.RegisterIfEnabled(module, true); err != nil {
		t.Fatal(err)
	}
	if len(reg.Manifests()) != 1 || reg.Manifests()[0].Migrations[0] != "20260921_sample" {
		t.Fatalf("unexpected manifest: %#v", reg.Manifests())
	}
}

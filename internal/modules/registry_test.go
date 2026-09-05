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

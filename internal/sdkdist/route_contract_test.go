package sdkdist_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	auth "github.com/ardaozyildirim/hydrogo/internal/auth"
	memberTransport "github.com/ardaozyildirim/hydrogo/modules/members/transport"
	newsTransport "github.com/ardaozyildirim/hydrogo/modules/news/transport"
	pageTransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
)

func TestCheckedInOpenAPIPathsMatchRuntimeRouteContracts(t *testing.T) {
	routes := make(map[string]bool)
	add := func(pattern string) {
		method, path, ok := strings.Cut(pattern, " ")
		if !ok {
			t.Fatalf("invalid route pattern %q", pattern)
		}
		routes[strings.ToUpper(method)+" "+path] = true
		if strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api/v1/") {
			routes[strings.ToUpper(method)+" /api/v1"+strings.TrimPrefix(path, "/api")] = true
		}
	}
	for _, route := range (pageTransport.Handler{}).Routes() {
		add(route.Pattern)
	}
	for _, route := range (newsTransport.Handler{}).Routes() {
		add(route.Pattern)
	}
	for _, route := range (auth.MobileHTTPHandler{}).RouterRoutes() {
		add(route.Pattern)
	}
	for _, route := range (memberTransport.Handler{}).Routes() {
		add(route.Pattern)
	}

	for _, name := range []string{"public-content-v1.openapi.json", "mobile-v1.openapi.json", "member-mobile-v1.openapi.json"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "docs", "api", name))
			if err != nil {
				t.Fatal(err)
			}
			var spec struct {
				Paths map[string]map[string]json.RawMessage `json:"paths"`
			}
			if err := json.Unmarshal(data, &spec); err != nil {
				t.Fatal(err)
			}
			for path, operations := range spec.Paths {
				for method := range operations {
					key := strings.ToUpper(method) + " " + path
					if !routes[key] {
						t.Errorf("OpenAPI operation %s has no runtime route", key)
					}
				}
			}
		})
	}
}

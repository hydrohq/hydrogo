package transport

import (
	"strings"
	"testing"
)

func TestRoutesSplitPublicAndAdminSurfaces(t *testing.T) {
	want := map[string]bool{"GET /api/v1/news": false, "GET /api/v1/news/by-slug": false, "GET /api/v1/news/{locale}/{slug}": false}
	for _, route := range (Handler{}).Routes() {
		if _, ok := want[route.Pattern]; ok {
			if route.Protected {
				t.Fatalf("%s must be public", route.Pattern)
			}
			want[route.Pattern] = true
		}
		if strings.Contains(route.Pattern, adminBase) && !route.Protected {
			t.Fatalf("%s must be protected", route.Pattern)
		}
	}
	for pattern, found := range want {
		if !found {
			t.Fatalf("missing public news route %s", pattern)
		}
	}
}

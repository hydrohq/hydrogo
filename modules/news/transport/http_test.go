package transport

import (
	"testing"
)

func TestRoutesIncludeCSRFProtectedAdminNewsForms(t *testing.T) {
	routes := (Handler{}).Routes()
	want := map[string]bool{
		"GET /admin/news/{id}":           true,
		"POST /admin/news/{id}":          true,
		"POST /admin/news/{id}/schedule": true,
		"POST /admin/news/{id}/publish":  true,
		"POST /admin/news/{id}/retract":  true,
	}
	for _, route := range routes {
		if _, ok := want[route.Pattern]; ok {
			if !route.Protected {
				t.Errorf("route %q is not protected", route.Pattern)
			}
			delete(want, route.Pattern)
		}
	}
	for pattern := range want {
		t.Errorf("missing admin route %q", pattern)
	}
}

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

func TestRoutesIncludePublicPublishedNewsList(t *testing.T) {
	foundList, foundBySlug := false, false
	for _, route := range (Handler{}).Routes() {
		if route.Pattern == "GET /api/news" {
			if route.Protected {
				t.Fatal("published news list must be public")
			}
			foundList = true
		}
		if route.Pattern == "GET /api/news/by-slug" {
			if route.Protected {
				t.Fatal("published news lookup must be public")
			}
			foundBySlug = true
		}
	}
	if !foundList || !foundBySlug {
		t.Fatalf("missing public news routes list=%t by_slug=%t", foundList, foundBySlug)
	}
}

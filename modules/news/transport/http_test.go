package transport

import "testing"

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

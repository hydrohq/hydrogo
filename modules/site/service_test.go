package site

import "testing"

func TestNormalizeNavigationMutatesRepositoryPayload(t *testing.T) {
	items := []NavigationItem{{ID: "home", Label: "Home", Href: "/", Position: 0}}
	locale, normalized, err := normalizeNavigation(" EN ", items)
	if err != nil {
		t.Fatalf("normalizeNavigation() error = %v", err)
	}
	if locale != "en" || normalized[0].Locale != "en" {
		t.Fatalf("normalized navigation = %#v, locale=%q", normalized, locale)
	}
}

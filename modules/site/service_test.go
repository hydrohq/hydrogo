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
	if normalized[0].Target != "_self" {
		t.Fatalf("legacy navigation target = %q", normalized[0].Target)
	}
}

func TestNormalizeNavigationValidatesTreeAndBoundsDepth(t *testing.T) {
	valid := []NavigationItem{
		{ID: "root", Label: "Root", Href: "/", Position: 0},
		{ID: "child", ParentID: "root", Target: "_blank", Label: "Child", Href: "/child", Position: 1},
	}
	if _, _, err := normalizeNavigation("en", valid); err != nil {
		t.Fatalf("valid tree rejected: %v", err)
	}
	for name, items := range map[string][]NavigationItem{
		"missing parent": {{ID: "child", ParentID: "missing", Label: "Child", Href: "/child", Position: 0}},
		"cycle": {
			{ID: "a", ParentID: "b", Label: "A", Href: "/a", Position: 0},
			{ID: "b", ParentID: "a", Label: "B", Href: "/b", Position: 1},
		},
		"invalid target": {{ID: "root", Target: "javascript", Label: "Root", Href: "/", Position: 0}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := normalizeNavigation("en", items); err == nil {
				t.Fatal("invalid menu accepted")
			}
		})
	}
}

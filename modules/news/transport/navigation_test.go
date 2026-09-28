package transport

import (
	"strings"
	"testing"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

func TestNewsLayoutRendersLocalizedNestedNavigation(t *testing.T) {
	items := []v1.ThemeNavigationItem{
		{ID: "root", Label: "Services", Href: "/en/services", Position: 0},
		{ID: "child", ParentID: "root", Label: "Consulting", Href: "/en/consulting", Position: 1, Target: "_blank"},
	}
	markup, err := v1.RenderThemeNavigation(items)
	if err != nil {
		t.Fatal(err)
	}
	page := newsDocumentStart("en", "News", markup)
	for _, marker := range []string{`lang="en"`, `href="/en/services"`, `class="site-navigation-child-list"`, `rel="noopener noreferrer"`, "Consulting"} {
		if !strings.Contains(page, marker) {
			t.Errorf("news layout missing %q", marker)
		}
	}
	if strings.Contains(page, `/public/pages/tr/showcase/html`) {
		t.Fatal("localized menu did not replace the fallback links")
	}
}

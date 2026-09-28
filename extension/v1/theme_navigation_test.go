package v1

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

func TestRenderThemeNavigationBuildsEscapedNestedMenus(t *testing.T) {
	got, err := RenderThemeNavigation([]ThemeNavigationItem{
		{ID: "root", Label: `<Services>`, Href: "/services", Position: 0},
		{ID: "child", ParentID: "root", Label: "Consulting", Href: "/consulting?a=1&b=2", Target: "_blank", Position: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`&lt;Services&gt;`, `class="site-navigation-child-list"`, `href="/consulting?a=1&amp;b=2"`, `target="_blank" rel="noopener noreferrer"`} {
		if !strings.Contains(string(got), marker) {
			t.Errorf("navigation output missing %q: %s", marker, got)
		}
	}
}

func TestRenderThemeNavigationRejectsUnsafeOrBrokenTrees(t *testing.T) {
	for name, items := range map[string][]ThemeNavigationItem{
		"external":       {{ID: "x", Label: "X", Href: "https://example.test"}},
		"missing parent": {{ID: "x", ParentID: "missing", Label: "X", Href: "/x"}},
		"cycle": {
			{ID: "a", ParentID: "b", Label: "A", Href: "/a"},
			{ID: "b", ParentID: "a", Label: "B", Href: "/b"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderThemeNavigation(items); err == nil {
				t.Fatal("invalid navigation accepted")
			}
		})
	}
}

func TestThemeLayoutCanRenderTheLocalizedNavigationSlot(t *testing.T) {
	nav, err := RenderThemeNavigation([]ThemeNavigationItem{{ID: "home", Label: "Accueil", Href: "/fr", Position: 0}})
	if err != nil {
		t.Fatal(err)
	}
	bundle := ThemeBundle{Manifest: ThemeManifest{Name: "test", Version: "1", Slots: []string{"navigation"}}, Layout: `<!doctype html><html lang="{{.Locale}}"><nav>{{if .Navigation}}{{slot "navigation"}}{{end}}</nav></html>`}
	var output bytes.Buffer
	if err := bundle.RenderPage(&output, PageTemplateData{Locale: "fr", Navigation: []ThemeNavigationItem{{ID: "home"}}, Slots: map[string]template.HTML{"navigation": nav}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `lang="fr"`) || !strings.Contains(output.String(), `>Accueil</a>`) {
		t.Fatalf("rendered theme page = %s", output.String())
	}
}

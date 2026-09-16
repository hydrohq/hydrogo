package adminstudio

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
)

func TestRenderTaxonomyEditorEscapesAndSortsTerms(t *testing.T) {
	var b strings.Builder
	terms := []v1.TaxonomyTerm{{ID: `z"`, Slug: "z", Kind: "tag", Labels: map[string]string{"en": "Z <unsafe>"}}, {ID: "a", Slug: "a", Kind: "category", Labels: map[string]string{"tr": "Alfa"}}}
	if err := RenderTaxonomyEditor(&b, TaxonomyEditorOptions{Terms: terms, Selected: []string{`z"`}}); err != nil {
		t.Fatal(err)
	}
	html := b.String()
	if !strings.Contains(html, `value="z&#34;" checked`) || strings.Contains(html, "<unsafe>") {
		t.Fatalf("unsafe or unselected taxonomy output: %s", html)
	}
	if strings.Index(html, "Alfa") > strings.Index(html, "Z ") {
		t.Fatalf("terms were not deterministic: %s", html)
	}
}

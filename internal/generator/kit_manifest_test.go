package generator

import (
	"strings"
	"testing"
)

func TestRenderKitManifestIsTypedAndDeterministic(t *testing.T) {
	got := renderKitManifest(Spec{Name: "article", Kits: []string{"search", "seo"}})
	if !strings.Contains(got, `ArticleKits = ["search", "seo"] as const`) {
		t.Fatalf("kit manifest = %s", got)
	}
}

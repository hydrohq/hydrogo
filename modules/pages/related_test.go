package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRelatedPagesTemplFiltersAndEscapes(t *testing.T) {
	items := []Page{
		{ID: "current", Locale: "en", Slug: "current", Title: "Current", Status: StatusPublished},
		{ID: "draft", Locale: "en", Slug: "draft", Title: "Draft", Status: StatusDraft},
		{ID: "related", Locale: "en", Slug: "related-story", Title: `<script>alert(1)</script>`, Status: StatusPublished},
		{ID: "wrong-locale", Locale: "fr", Slug: "autre", Title: "Autre", Status: StatusPublished},
	}
	var output bytes.Buffer
	if err := RelatedPages("current", "en", items).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	if !strings.Contains(markup, `/public/pages/en/related-story/html`) || !strings.Contains(markup, `&lt;script&gt;`) {
		t.Fatalf("related card missing or unescaped: %s", markup)
	}
	for _, forbidden := range []string{"Current", "Draft", "Autre", "<script>"} {
		if strings.Contains(markup, forbidden) {
			t.Fatalf("unexpected related markup %q: %s", forbidden, markup)
		}
	}
}

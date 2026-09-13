package news

import (
	"strings"
	"testing"
)

func TestValidateRejectsOversizedMetadata(t *testing.T) {
	base := Item{ID: "news-1", Locale: "tr", Slug: "duyuru", Title: "Duyuru", Status: StatusDraft}
	cases := []struct {
		name  string
		value func(*Item)
	}{
		{"id", func(v *Item) { v.ID = strings.Repeat("x", 129) }},
		{"slug", func(v *Item) { v.Slug = strings.Repeat("x", 257) }},
		{"title", func(v *Item) { v.Title = strings.Repeat("x", 257) }},
		{"seo description", func(v *Item) { v.SEODescription = strings.Repeat("x", 1001) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			item := base
			tc.value(&item)
			if err := Validate(item); err == nil {
				t.Fatal("expected oversized metadata to be rejected")
			}
		})
	}
}

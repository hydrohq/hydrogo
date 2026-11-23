package blueprints

import (
	"github.com/hydrohq/hydrogo/internal/contenttypes"
)

// NewsDefinitions returns the collection schemas for a modern digital news portal.
func NewsDefinitions() []contenttypes.Definition {
	return []contenttypes.Definition{
		{
			ID:          "site_settings",
			Label:       "Portal & Publication Settings",
			Description: "Newspaper/portal name, tagline, accent colors, masthead and announcements.",
			Group:       "Publication Management",
			Icon:        "newspaper",
			TitleField:  "brand_name",
			Public:      true,
			Singleton:   true,
			Fields: []contenttypes.Field{
				{ID: "brand_name", Label: "Portal Name", Kind: "text", Required: true, List: true},
				{ID: "tagline", Label: "Publication Motto / Tagline", Kind: "text", List: true},
				{ID: "primary_color", Label: "Primary Accent Color", Kind: "text"},
				{ID: "bg_tone", Label: "Background Tone", Kind: "text"},
				{ID: "font_preset", Label: "Typography Palette", Kind: "select", Options: []string{"v0_geist", "editorial_playfair", "artisan_garamond", "monumental_cinzel", "avantgarde_syne", "modern_grotesk"}},
				{ID: "breaking_news", Label: "Breaking News Headline", Kind: "text", List: true},
				{ID: "contact_email", Label: "Editorial Desk Email", Kind: "text", List: true},
			},
		},
		{
			ID:          "news_categories",
			Label:       "News Categories",
			Description: "Headlines, Technology, Economy, Culture, Science, etc.",
			Group:       "Publication Management",
			Icon:        "folder",
			TitleField:  "name",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "name", Label: "Category Name", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true, Filterable: true},
				{ID: "sort_order", Label: "Sort Order", Kind: "number", List: true},
				{ID: "description", Label: "Category Description", Kind: "text"},
			},
		},
		{
			ID:          "news_articles",
			Label:       "News & Articles",
			Description: "Published headlines, analyses and opinion pieces.",
			Group:       "Publication Management",
			Icon:        "file-text",
			TitleField:  "title",
			Public:      true,
			Fields: []contenttypes.Field{
				{ID: "title", Label: "Headline", Kind: "text", Required: true, List: true, Searchable: true},
				{ID: "slug", Label: "URL Slug", Kind: "text", Required: true, Unique: true, List: true},
				{ID: "category_slug", Label: "Category", Kind: "text", Widget: "relation", To: "news_categories", ValueField: "slug", Required: true, List: true, Filterable: true},
				{ID: "excerpt", Label: "Summary / Standfirst", Kind: "textarea", Required: true, List: true, Searchable: true},
				{ID: "body", Label: "Article Body", Kind: "richtext", Required: true, Searchable: true},
				{ID: "image_url", Label: "Cover Image", Kind: "text", Widget: "media", List: true},
				{ID: "author_name", Label: "Author / Reporter", Kind: "text", List: true, Searchable: true},
				{ID: "reading_time", Label: "Reading Time (min)", Kind: "number", List: true},
				{ID: "published_at", Label: "Published At", Kind: "text", List: true},
				{ID: "is_lead", Label: "Top Story of the Day", Kind: "boolean", List: true, Filterable: true},
				{ID: "is_trending", Label: "Most Read", Kind: "boolean", List: true, Filterable: true},
			},
		},
	}
}

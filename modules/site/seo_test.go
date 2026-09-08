package site

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSEOCanonicalAndFallbacks(t *testing.T) {
	view, err := BuildSEO(SEOInput{Origin: "https://example.com/", Locale: "tr", Path: "/Hizmetler/", Title: "Hizmetler", Description: "Açıklama"})
	if err != nil {
		t.Fatal(err)
	}
	if view.CanonicalURL != "https://example.com/tr/hizmetler" || view.OGTitle != "Hizmetler" || view.OGDescription != "Açıklama" {
		t.Fatalf("SEO view = %#v", view)
	}
	if _, err := BuildSEO(SEOInput{Origin: "javascript:alert(1)", Path: "/x"}); err == nil {
		t.Fatal("unsafe SEO origin accepted")
	}
}

func TestRenderSitemapEscapesAndDeduplicates(t *testing.T) {
	data, err := RenderSitemap([]SitemapEntry{{URL: "https://example.com/a?x=1&y=2", LastModified: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)}, {URL: "https://example.com/a?x=1&y=2"}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, "<loc>") != 1 || !strings.Contains(text, "&amp;") || !strings.Contains(text, "2026-09-08") {
		t.Fatalf("sitemap = %s", text)
	}
}

func TestNavigationRejectsNetworkPathURL(t *testing.T) {
	if err := ValidateNavigation(NavigationItem{ID: "x", Locale: "en", Label: "x", Href: "//evil.example", Position: 0}); err == nil {
		t.Fatal("network path navigation URL accepted")
	}
}

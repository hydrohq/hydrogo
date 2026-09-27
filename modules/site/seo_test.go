package site

import (
	"bytes"
	"context"
	"strconv"
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
	unicodeView, err := BuildSEO(SEOInput{Origin: "https://example.com", Locale: "TR", Path: "/cafe\u0301"})
	if err != nil || unicodeView.CanonicalURL != "https://example.com/tr/caf%C3%A9" {
		t.Fatalf("Unicode canonical URL = %#v/%v", unicodeView, err)
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

func TestSplitSitemapHonorsCountAndByteCapsAndRendersIndex(t *testing.T) {
	entries := []SitemapEntry{
		{URL: "https://example.com/a"},
		{URL: "https://example.com/b"},
		{URL: "https://example.com/c"},
		{URL: "https://example.com/d"},
		{URL: "https://example.com/e"},
	}
	parts, err := SplitSitemap(entries, 2, 0)
	if err != nil || len(parts) != 3 || len(parts[0]) != 2 || len(parts[2]) != 1 {
		t.Fatalf("sitemap parts=%v err=%v", len(parts), err)
	}
	index, err := RenderSitemapIndex("https://example.com", len(parts))
	if err != nil || strings.Count(string(index), "<sitemap>") != 3 || !strings.Contains(string(index), "sitemap-pages.xml?part=2") {
		t.Fatalf("sitemap index=%s err=%v", index, err)
	}
	if _, err := SplitSitemap([]SitemapEntry{{URL: "https://example.com/" + strings.Repeat("x", 512)}}, 10, 256); err == nil {
		t.Fatal("oversized single sitemap entry was accepted")
	}
}

func TestStreamSitemapPartBoundsMemoryAndSeparatesCategories(t *testing.T) {
	entries := []SitemapEntry{{URL: "https://example.com/a"}, {URL: "https://example.com/b"}, {URL: "https://example.com/c"}, {URL: "https://example.com/d"}, {URL: "https://example.com/e"}}
	stream := func(_ context.Context, category string, visit func(SitemapEntry) error) error {
		for _, entry := range entries {
			entry.Category = category
			if err := visit(entry); err != nil {
				return err
			}
		}
		return nil
	}
	var part bytes.Buffer
	count, err := StreamSitemapPart(context.Background(), stream, "pages", 2, 2, 0, &part)
	if err != nil || count != 3 || strings.Count(part.String(), "<url>") != 2 || !strings.Contains(part.String(), "https://example.com/c") || !strings.Contains(part.String(), "https://example.com/d") {
		t.Fatalf("streamed part count=%d data=%s err=%v", count, part.String(), err)
	}
	index, err := RenderCategorizedSitemapIndex("https://example.com", map[string]int{"pages": 3, "articles": 1})
	if err != nil || !strings.Contains(string(index), "/sitemap-articles.xml?part=1") || !strings.Contains(string(index), "/sitemap-pages.xml?part=3") {
		t.Fatalf("categorized index=%s err=%v", index, err)
	}
}

func TestStreamSitemapAutomaticallySplitsAfterProtocolURLCap(t *testing.T) {
	stream := func(_ context.Context, _ string, visit func(SitemapEntry) error) error {
		for i := 0; i <= MaxSitemapURLs; i++ {
			if err := visit(SitemapEntry{URL: "https://example.com/" + strconv.Itoa(i)}); err != nil {
				return err
			}
		}
		return nil
	}
	parts, err := StreamSitemapPart(context.Background(), stream, "pages", 0, MaxSitemapURLs, MaxSitemapBytes, nil)
	if err != nil || parts != 2 {
		t.Fatalf("streamed sitemap parts=%d err=%v", parts, err)
	}
}

func TestNavigationRejectsNetworkPathURL(t *testing.T) {
	if err := ValidateNavigation(NavigationItem{ID: "x", Locale: "en", Label: "x", Href: "//evil.example", Position: 0}); err == nil {
		t.Fatal("network path navigation URL accepted")
	}
	if err := ValidateNavigation(NavigationItem{ID: "x", Locale: "en", Label: "x", Href: "https:evil.example", Position: 0}); err == nil {
		t.Fatal("absolute URI without host was accepted")
	}
}

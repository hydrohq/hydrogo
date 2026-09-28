package transport

import (
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestPageSEOMetadataIncludesBrandAndVersionedOGImage(t *testing.T) {
	handler := Handler{SEOOrigin: "https://client.example", PublicSEOSettings: func(context.Context) PublicSEOSettings {
		return PublicSEOSettings{Brand: "Client Brand", FontCSS: "/fonts/font-inter.css"}
	}}
	page := pages.Page{Locale: "tr", Slug: "home", Title: "Yeni Başlık"}
	meta, err := handler.pageSEOMeta(context.Background(), page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(meta, `https://client.example/tr/home`) || !strings.Contains(meta, `og:image`) || !strings.Contains(meta, `?v=`) || !strings.Contains(meta, `/fonts/font-inter.css`) {
		t.Fatalf("SEO tags missing canonical or versioned OG image: %s", meta)
	}
	first := handler.pageOGInput(page, handler.publicSEOSettings(context.Background()))
	handler.PublicSEOSettings = func(context.Context) PublicSEOSettings { return PublicSEOSettings{Brand: "Other Brand"} }
	second := handler.pageOGInput(page, handler.publicSEOSettings(context.Background()))
	if first.Brand == second.Brand {
		t.Fatal("brand metadata callback was ignored")
	}
}

func TestPageSEOOnlyEmitsGeneratedLocalFontStylesheets(t *testing.T) {
	page := pages.Page{Locale: "tr", Slug: "home", Title: "Home"}
	for _, value := range []string{"https://evil.test/a.css", "/fonts/secret.css", "/fonts/font-../x.css"} {
		handler := Handler{PublicSEOSettings: func(context.Context) PublicSEOSettings { return PublicSEOSettings{FontCSS: value} }}
		meta, err := handler.pageSEOMeta(context.Background(), page)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(meta, "rel=\"stylesheet\"") {
			t.Errorf("unsafe font URL emitted: %s", meta)
		}
	}
}

func TestPageHeadIncludesValidatedAppearanceTokens(t *testing.T) {
	handler := Handler{SEOOrigin: "https://hydro.invalid", PublicSEOSettings: func(context.Context) PublicSEOSettings {
		return PublicSEOSettings{AppearanceCSS: `:root{--brand:#155EEF;--accent:#155EEF}`, FaviconURL: "/media/favicon.svg", SocialImageURL: "/media/share.webp"}
	}}
	meta, err := handler.pageSEOMeta(context.Background(), pages.Page{Locale: "en", Slug: "home", Title: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{`<style nonce="">:root{--brand:#155EEF;--accent:#155EEF}</style>`, `rel="icon" href="/media/favicon.svg"`, `og:image" content="https://hydro.invalid/media/share.webp`} {
		if !strings.Contains(meta, marker) {
			t.Errorf("appearance head missing %q: %s", marker, meta)
		}
	}
}

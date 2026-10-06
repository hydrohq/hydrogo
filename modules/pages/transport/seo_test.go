package transport

import (
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

type alternatePagesStub struct{ translations []pages.Page }

func (alternatePagesStub) GetContent(context.Context, database.Executor, string) (pages.Page, error) {
	return pages.Page{}, nil
}

func TestFloatingContactUsesConfiguredSupportOnly(t *testing.T) {
	handler := Handler{PublicSupport: adminstudio.SupportConfig{Enabled: true, Phone: "+1 415 555 0132", WhatsAppURL: "https://wa.me/14155550132"}}
	got := handler.floatingContact(context.Background(), "en")
	if !strings.Contains(got, `aria-label="Contact options"`) || !strings.Contains(got, `href="tel:+14155550132"`) || !strings.Contains(got, `https://wa.me/14155550132`) {
		t.Fatalf("configured public contact missing: %s", got)
	}
	if got := (Handler{}).floatingContact(context.Background(), "tr"); got != "" {
		t.Fatalf("disabled support rendered: %s", got)
	}
}
func (alternatePagesStub) GetPublishedByRoute(context.Context, database.Executor, string, string) (pages.Page, error) {
	return pages.Page{}, nil
}
func (alternatePagesStub) ListPublished(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return nil, nil
}
func (alternatePagesStub) CreateDraftContent(context.Context, database.Executor, pages.Page) error {
	return nil
}
func (alternatePagesStub) UpdateDraftContent(context.Context, database.Executor, pages.Page, int) error {
	return nil
}
func (alternatePagesStub) Publish(context.Context, database.Executor, string, int) error { return nil }
func (s alternatePagesStub) ListPublishedTranslations(context.Context, database.Executor, string, int) ([]pages.Page, error) {
	return s.translations, nil
}

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

func TestPageSEOMetadataLinksOnlyPublishedLocaleAlternates(t *testing.T) {
	repository := alternatePagesStub{translations: []pages.Page{
		{Locale: "tr", Slug: "about"},
		{Locale: "en", Slug: "about"},
		{Locale: "fr", Slug: "different"},
	}}
	handler := Handler{SEOOrigin: "https://client.example", Content: service.ContentService{Pages: repository}}
	meta, err := handler.pageSEOMeta(context.Background(), pages.Page{Locale: "tr", Slug: "about", Title: "Hakkımızda"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`hreflang="tr" href="https://client.example/public/pages/tr/about/html"`, `hreflang="en" href="https://client.example/public/pages/en/about/html"`} {
		if !strings.Contains(meta, want) {
			t.Errorf("alternate metadata missing %q: %s", want, meta)
		}
	}
	if strings.Contains(meta, `hreflang="fr"`) {
		t.Fatalf("different-slug page was linked as translation: %s", meta)
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
	for _, marker := range []string{`<style>:root{--brand:#155EEF;--accent:#155EEF}</style>`, `rel="icon" href="/media/favicon.svg"`, `og:image" content="https://hydro.invalid/media/share.webp`} {
		if !strings.Contains(meta, marker) {
			t.Errorf("appearance head missing %q: %s", marker, meta)
		}
	}
}

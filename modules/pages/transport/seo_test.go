package transport

import (
	"context"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestPageSEOMetadataIncludesBrandAndVersionedOGImage(t *testing.T) {
	handler := Handler{SEOOrigin: "https://client.example", OGBrand: func(context.Context) string { return "Client Brand" }}
	page := pages.Page{Locale: "tr", Slug: "home", Title: "Yeni Başlık"}
	meta, err := handler.pageSEOMeta(context.Background(), page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(meta, `https://client.example/tr/home`) || !strings.Contains(meta, `og:image`) || !strings.Contains(meta, `?v=`) {
		t.Fatalf("SEO tags missing canonical or versioned OG image: %s", meta)
	}
	first := handler.pageOGInput(context.Background(), page)
	handler.OGBrand = func(context.Context) string { return "Other Brand" }
	second := handler.pageOGInput(context.Background(), page)
	if first.Brand == second.Brand {
		t.Fatal("brand metadata callback was ignored")
	}
}

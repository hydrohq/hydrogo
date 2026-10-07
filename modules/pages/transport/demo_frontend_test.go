package transport

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestPublicPageAllowedHidesSeededDemoByDefault(t *testing.T) {
	h := Handler{}
	if h.publicPageAllowed(pages.Page{ID: "demo-showcase"}) {
		t.Fatal("seeded demo page should be hidden by default")
	}
	if !h.publicPageAllowed(pages.Page{ID: "page-custom-about"}) {
		t.Fatal("customer page should remain public")
	}
	if !(Handler{AllowDemoFrontend: true}).publicPageAllowed(pages.Page{ID: "demo-showcase"}) {
		t.Fatal("explicit demo frontend flag should expose seeded demo page")
	}
}

func TestPageLayoutCopyLocalizesBuiltinsAndPreservesEditionTemplates(t *testing.T) {
	if name, description := pageLayoutCopy(pages.BuiltinLayoutTemplates()[0], "tr"); name != "Boş sayfa" || description == "" {
		t.Fatalf("Turkish builtin copy = %q, %q", name, description)
	}
	custom := pages.LayoutTemplate{ID: "edition_custom", Name: "Edition custom", Description: "Customer supplied"}
	if name, description := pageLayoutCopy(custom, "tr"); name != custom.Name || description != custom.Description {
		t.Fatalf("edition template copy changed: %q, %q", name, description)
	}
	if name, _ := pageLayoutCopy(custom, "en"); name != custom.Name {
		t.Fatalf("English edition template name changed: %q", name)
	}
}

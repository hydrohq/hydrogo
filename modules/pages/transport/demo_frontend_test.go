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

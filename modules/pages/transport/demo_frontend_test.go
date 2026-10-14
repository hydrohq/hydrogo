package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
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

func TestPageEditorRoutesAllowEditorAndDenyViewer(t *testing.T) {
	newPage := func(role auth.Role) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/admin/pages/new", nil)
		return req.WithContext(auth.WithUser(req.Context(), &auth.User{ID: string(role), Role: role, Active: true}))
	}
	h := Handler{}
	for _, tc := range []struct {
		role auth.Role
		want int
	}{{auth.RoleEditor, http.StatusOK}, {auth.RoleAdmin, http.StatusOK}, {auth.RoleViewer, http.StatusForbidden}} {
		res := httptest.NewRecorder()
		h.adminNewPage(res, newPage(tc.role))
		// The bare handler has no form collection, so allowed roles may get a
		// render error; only the permission decision is asserted here.
		if denied := res.Code == http.StatusForbidden; denied != (tc.want == http.StatusForbidden) {
			t.Errorf("GET /admin/pages/new as %s = %d, want %d", tc.role, res.Code, tc.want)
		}
	}
	viewer := httptest.NewRequest(http.MethodPost, "/admin/pages", nil)
	viewer = viewer.WithContext(auth.WithUser(viewer.Context(), &auth.User{ID: "viewer", Role: auth.RoleViewer, Active: true}))
	res := httptest.NewRecorder()
	h.adminCreateForm(res, viewer)
	if res.Code != http.StatusForbidden {
		t.Fatalf("POST /admin/pages as viewer = %d, want %d", res.Code, http.StatusForbidden)
	}
}

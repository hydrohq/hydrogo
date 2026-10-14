package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

var inlineScriptOrStyle = regexp.MustCompile(`<script(?:\s[^>]*)?>[^<]|<script>|<style[\s>]|\sstyle="`)

// assertExternalBehaviour checks a site admin screen loads its behaviour from
// a static file and carries no inline script or style the CSP would block.
func assertExternalBehaviour(t *testing.T, body, script string) {
	t.Helper()
	if match := inlineScriptOrStyle.FindString(body); match != "" {
		t.Errorf("inline script or style %q in page", match)
	}
	if !strings.Contains(body, `<script defer src="`+script+`"></script>`) {
		t.Errorf("page does not load %s", script)
	}
}

func TestSiteAdminScreensUseStaticScripts(t *testing.T) {
	admin := &auth.User{ID: "admin", Role: auth.RoleAdmin, Active: true}
	for _, tc := range []struct {
		path, start, script string
		serve               func(http.ResponseWriter, *http.Request)
	}{
		{"/admin/translations", `<main class="studio-card translation-editor"`, "/admin-assets/js/translations.js", Handler{Store: fakeStore{}, Service: &site.Service{}}.translations},
		{"/admin/site/appearance", `<section class="card"`, "/admin-assets/js/appearance.js", Handler{Store: fakeStore{}, Service: &site.Service{}}.appearance},
	} {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		request = request.WithContext(auth.WithUser(request.Context(), admin))
		response := httptest.NewRecorder()
		tc.serve(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d", tc.path, response.Code)
		}
		body := response.Body.String()
		// The admin shell (head, AdminEnd scripts) is out of scope; check only
		// the screen body up to its own script tag.
		start := strings.Index(body, tc.start)
		if start < 0 {
			t.Fatalf("%s: screen body %q not found", tc.path, tc.start)
		}
		body = body[start:]
		if end := strings.Index(body, tc.script); end >= 0 {
			body = body[:end+len(tc.script)+len(`"></script>`)]
		}
		assertExternalBehaviour(t, body, tc.script)
	}
}

func TestNavigationEditorRowsShareOneTemplate(t *testing.T) {
	items := []site.NavigationItem{{ID: "root", Label: "Services", Href: "/services", Visible: true}, {ID: "child", ParentID: "root", Label: `Consulting <b>`, Href: "/consulting", Target: "_blank", Position: 1}}
	view := navigationEditorView{Locale: "en", Locales: []string{"en", "tr"}, CSRF: "token"}
	for _, item := range items {
		view.Rows = append(view.Rows, navigationRowView{Key: item.ID, Item: item, Position: "0", Parents: navigationParentOptions(item.ID, items)})
	}
	view.Template = navigationRowView{Key: navigationNewRowKey, Item: site.NavigationItem{Target: "_self", Visible: true}, Parents: navigationParentOptions("", nil)}
	var out strings.Builder
	if err := navigationEditorPage(view).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	assertExternalBehaviour(t, body, "/admin-assets/js/navigation-editor.js")
	for _, marker := range []string{`id="nav-parent-child"`, `<option value="root" selected`, `id="nav-label-__new__"`, `data-row-key="__new__"`, `Consulting &lt;b&gt;`, `aria-current="page"`, `name="_csrf" value="token"`} {
		if !strings.Contains(body, marker) {
			t.Errorf("navigation editor missing %q", marker)
		}
	}
}

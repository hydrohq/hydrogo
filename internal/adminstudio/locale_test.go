package adminstudio

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderCreateFormLocaleSelector(t *testing.T) {
	c := Collection{ID: "pages", Label: "Pages", Fields: []Field{{ID: "title", Label: "Başlık", Kind: FieldText}}, Actions: []Action{{Label: "Pages", Href: "/admin/pages"}}}
	var out bytes.Buffer
	if err := RenderCreateForm(&out, FormData{Collection: c, Action: "/admin/pages", CSRFToken: "csrf", Locales: []string{"tr", "en", "<bad>"}, CurrentLocale: "en"}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if !strings.Contains(html, `name="locale"`) || !strings.Contains(html, `value="en" selected`) || !strings.Contains(html, `&lt;bad&gt;`) {
		t.Fatalf("locale selector missing or unsafe: %s", html)
	}
}

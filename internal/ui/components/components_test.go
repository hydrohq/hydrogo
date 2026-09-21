package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var out bytes.Buffer
	if err := c.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestButtonEscapesAndAllowlistsHTMX(t *testing.T) {
	out := render(t, Button(ButtonProps{Label: `<script>alert(1)</script>`, HTMX: HTMX{Post: "/save", Target: "#result", Swap: "outerHTML"}}))
	if !strings.Contains(out, `hx-post="/save"`) || !strings.Contains(out, `hx-target="#result"`) || !strings.Contains(out, `hx-swap="outerHTML"`) {
		t.Fatalf("missing allowlisted attributes: %s", out)
	}
	if strings.Contains(out, "<script>") || strings.Contains(out, "hx-on") {
		t.Fatalf("unsafe or non-allowlisted output: %s", out)
	}
}

func TestFormAndInputRenderCSRFAndEscapedValues(t *testing.T) {
	out := render(t, Form(FormProps{Action: `/projects/<x>`, CSRF: `a"b`, Body: Input(InputProps{ID: "title", Name: "title", Label: "Title", Required: true})}))
	if !strings.Contains(out, `name="_csrf" value="a&#34;b"`) || !strings.Contains(out, `action="/projects/&lt;x&gt;"`) || !strings.Contains(out, `required="required"`) {
		t.Fatalf("unexpected form output: %s", out)
	}
}

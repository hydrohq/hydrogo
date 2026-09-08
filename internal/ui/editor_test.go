package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderEditorUsesLocalAssetsAndCSPNonce(t *testing.T) {
	var out bytes.Buffer
	err := RenderEditor(&out, EditorData{
		Action:    "/admin/pages/1",
		CSRFToken: "csrf-token",
		Nonce:     "nonce-value",
		Title:     "Draft",
		BodyDelta: `{"ops":[{"insert":"Hello\n"}]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{
		`/vendor/htmx/htmx.min.js`,
		`/vendor/quill/quill.js`,
		`/vendor/quill/quill.snow.css`,
		`nonce="nonce-value"`,
		`htmx.config.allowEval = false`,
		`htmx.config.allowScriptTags = false`,
		`name="body_delta"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered editor missing %q", want)
		}
	}
	for _, forbidden := range []string{"https://cdn.", "http://cdn.", "onclick=", "onload="} {
		if strings.Contains(strings.ToLower(html), forbidden) {
			t.Errorf("rendered editor contains forbidden %q", forbidden)
		}
	}
}

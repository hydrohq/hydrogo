package adminstudio

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

func TestRenderFormBuilderEscapesSchemaAndProvidesCSPEditor(t *testing.T) {
	var out strings.Builder
	err := RenderFormBuilder(&out, FormBuilderData{Form: forms.Form{ID: "contact", Title: `A <form>`, Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: `E "mail"`, Required: true}}}, Action: "/admin/forms/contact", CSRFToken: "csrf&", Nonce: "nonce"})
	if err != nil {
		t.Fatal(err)
	}
	value := out.String()
	for _, marker := range []string{"data-form-builder", "data-form-field", `nonce="nonce"`, "csrf&amp;", "A &lt;form&gt;", "data-form-add-field"} {
		if !strings.Contains(value, marker) {
			t.Fatalf("missing %q in %s", marker, value)
		}
	}
}

func TestRenderFormBuilderRejectsInvalidSchema(t *testing.T) {
	var out strings.Builder
	if err := RenderFormBuilder(&out, FormBuilderData{Form: forms.Form{ID: "bad id"}}); err == nil {
		t.Fatal("invalid form accepted")
	}
}

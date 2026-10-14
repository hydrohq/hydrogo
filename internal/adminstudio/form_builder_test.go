package adminstudio

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

func TestRenderFormBuilderEscapesSchemaAndProvidesCSPEditor(t *testing.T) {
	var out strings.Builder
	err := RenderFormBuilder(&out, FormBuilderData{Form: forms.Form{ID: "contact", Title: `A <form>`, Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: `E "mail"`, Required: true}}}, Action: "/admin/forms/contact", CSRFToken: "csrf&"})
	if err != nil {
		t.Fatal(err)
	}
	value := out.String()
	for _, marker := range []string{"data-form-builder", "data-form-field", "csrf&amp;", "A &lt;form&gt;", "data-form-add-field", `name="conversion_tracking"`, `name="google_ads_conversion_id"`, `name="meta_pixel_event_name"`, "mevcut etiketler"} {
		if !strings.Contains(value, marker) {
			t.Fatalf("missing %q in %s", marker, value)
		}
	}
	if !strings.Contains(value, `<script defer src="/admin-assets/js/form-builder.js"></script>`) || strings.Contains(value, "<script>") || strings.Contains(value, "<style") {
		t.Fatalf("form builder must load its script from a static file without inline code: %s", value)
	}
}

func TestRenderFormBuilderRejectsInvalidSchema(t *testing.T) {
	var out strings.Builder
	if err := RenderFormBuilder(&out, FormBuilderData{Form: forms.Form{ID: "bad id"}}); err == nil {
		t.Fatal("invalid form accepted")
	}
}

package ui

import (
	"strings"
	"testing"
)

func TestSharedComponentsEscapeUntrustedValues(t *testing.T) {
	for name, rendered := range map[string]string{
		"button": string(Button(ButtonOptions{Label: `<script>alert(1)</script>`, Class: `x" onfocus="bad`})),
		"field":  string(Field(FieldOptions{ID: `a" onfocus="bad`, Name: "q", Label: "Ad", Value: `<img src=x>`})),
		"error":  string(Error(`<script>alert(1)</script>`)),
		"table":  string(Table("Kayıtlar", []TableColumn{{Label: "Başlık"}}, [][]string{{`<script>`}})),
	} {
		if strings.Contains(rendered, "<script>") || strings.Contains(rendered, "onfocus=\"bad\"") {
			t.Fatalf("%s rendered unescaped content: %s", name, rendered)
		}
	}
}

func TestSharedComponentsContracts(t *testing.T) {
	if got := string(Button(ButtonOptions{Label: "Kaydet", Type: "submit", Disabled: true})); !strings.Contains(got, `class="button"`) || !strings.Contains(got, `aria-disabled="true"`) {
		t.Fatalf("button contract missing: %s", got)
	}
	if got := string(Field(FieldOptions{ID: "name", Name: "name", Label: "Ad", Type: "textarea", Required: true})); !strings.Contains(got, `<textarea`) || !strings.Contains(got, `aria-required="true"`) {
		t.Fatalf("field contract missing: %s", got)
	}
	if got := string(Pagination(2, 3, "/p/1", "/p/3")); !strings.Contains(got, `rel="prev"`) || !strings.Contains(got, `rel="next"`) {
		t.Fatalf("pagination contract missing: %s", got)
	}
	if got := string(Empty("")); !strings.Contains(got, `role="status"`) {
		t.Fatalf("empty state contract missing: %s", got)
	}
}

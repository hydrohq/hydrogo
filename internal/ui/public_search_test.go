package ui

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPublicSearchBoxUsesBoundedHTMXContract(t *testing.T) {
	var b bytes.Buffer
	if err := PublicSearchBox().Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`hx-get="/api/v1/search/autocomplete"`, `hx-trigger="keyup changed delay:250ms"`, `role="combobox"`, `/vendor/htmx/htmx.min.js`} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}
func TestSearchSuggestionsEscapeContentAndLinks(t *testing.T) {
	var b bytes.Buffer
	err := SearchSuggestions([]SearchSuggestion{{Title: `<img src=x>`, Category: "Page", URL: "/public/pages/tr/safe/html"}}).Render(context.Background(), &b)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "<img") || !strings.Contains(b.String(), "&lt;img") {
		t.Fatalf("unescaped result=%s", b.String())
	}
}

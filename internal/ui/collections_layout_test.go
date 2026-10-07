package ui

import (
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

func TestCollectionRecordFormLayoutClasses(t *testing.T) {
	schema := collections.Schema{Fields: []collections.Field{
		{ID: "code", Label: "Code", Kind: collections.Text},
		{ID: "title", Label: "Title", Kind: collections.Text},
	}}
	fields := CollectionRecordFields(schema, nil)
	if got := collectionRecordFormClass(fields); got != "studio-form collection-record-form" {
		t.Fatalf("two-field form class = %q", got)
	}
	single := CollectionRecordFields(collections.Schema{Fields: schema.Fields[:1]}, nil)
	if got := collectionRecordFormClass(single); got != "studio-form collection-record-form single-field" {
		t.Fatalf("single-field form class = %q", got)
	}
	for _, selector := range []string{
		".collection-record-form label:last-of-type{grid-column:auto}",
		".collection-record-form.single-field label:last-of-type{grid-column:1/-1}",
		".collection-record-form>label:has(textarea){grid-column:1/-1!important}",
	} {
		if !strings.Contains(AdminCollectionRecordFormCSS, selector) {
			t.Errorf("collection form layout CSS missing %q", selector)
		}
	}
}

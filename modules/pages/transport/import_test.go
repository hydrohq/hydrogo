package transport

import (
	"testing"
)

func TestPlanImportSkipsExistingAndRepeatedRows(t *testing.T) {
	rows := [][]string{{"tr", "about", "About", "Text"}, {"tr", "about", "Again", "Text"}, {"", "bad", "", ""}, {"en", "contact", "Contact", "Text"}}
	columns := map[string]int{"locale": 0, "slug": 1, "title": 2, "body": 3}
	valid, duplicates, invalid := planImport(rows, columns, map[string]bool{"tr\x00about": true})
	if len(valid) != 1 || valid[0].Slug != "contact" || duplicates != 2 || invalid != 1 {
		t.Fatalf("plan = valid:%v duplicates:%d invalid:%d", valid, duplicates, invalid)
	}
}

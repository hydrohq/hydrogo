package contenttypes

import "testing"

func TestStockContentTypesAreValid(t *testing.T) {
	for _, dir := range []string{"../../content-types", "../../content-types/crm", "../../content-types/erp"} {
		definitions, err := LoadDir(dir)
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		if len(definitions) == 0 {
			t.Fatalf("%s: no definitions loaded", dir)
		}
		for _, definition := range definitions {
			if _, err := definition.Schema(); err != nil {
				t.Errorf("%s/%s: %v", dir, definition.ID, err)
			}
		}
	}
}

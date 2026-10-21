package contenttypes

import "testing"

func TestHideUnreadableFieldsFollowsFieldPermissions(t *testing.T) {
	_, types := newServiceFor(t, `id: margins
label: Margins
fields:
  - {id: name, label: Name, kind: text}
  - {id: margin, label: Margin, kind: number, permission: crm.margin.read}
`)
	collections := func() int {
		return len(AdminCollections(types, false)[0].Fields)
	}
	if collections() != 2 {
		t.Fatalf("fields = %d", collections())
	}
	hidden := HideUnreadableFields(AdminCollections(types, false), types, func(string) bool { return false })
	if len(hidden[0].Fields) != 1 || hidden[0].Fields[0].ID != "name" {
		t.Fatalf("without the permission: %+v", hidden[0].Fields)
	}
	shown := HideUnreadableFields(AdminCollections(types, false), types, func(p string) bool { return p == "crm.margin.read" })
	if len(shown[0].Fields) != 2 {
		t.Fatalf("with the permission: %+v", shown[0].Fields)
	}
}

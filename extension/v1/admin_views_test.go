package v1

import "testing"

func TestValidateAdminViewBoundsAndAllowlist(t *testing.T) {
	fields := map[string]QueryField{"title": {Name: "title"}, "status": {Name: "status", Operators: map[string]bool{"equals": true}}}
	view, err := ValidateAdminView(AdminView{ID: "published", Name: "Published", Columns: []string{"status", "title"}, Filters: []AdminViewFilter{{Field: "status", Operator: "equals", Value: "published"}}}, fields)
	if err != nil {
		t.Fatal(err)
	}
	if view.Columns[0] != "status" || view.Columns[1] != "title" {
		t.Fatalf("columns not canonicalized: %#v", view.Columns)
	}
	if _, err := ValidateAdminView(AdminView{ID: "bad", Name: "Bad", Columns: []string{"secret"}}, fields); err == nil {
		t.Fatal("unknown column accepted")
	}
	if _, err := ValidateAdminView(AdminView{ID: "bad", Name: "Bad", Filters: []AdminViewFilter{{Field: "status", Operator: "contains", Value: "x"}}}, fields); err == nil {
		t.Fatal("operator not restricted by field metadata")
	}
}

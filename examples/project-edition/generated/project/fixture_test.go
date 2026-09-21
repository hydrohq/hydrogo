package generated

import "testing"

func TestProjectFixtureShape(t *testing.T) {
	item := Project{ID: "fixture-project"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}

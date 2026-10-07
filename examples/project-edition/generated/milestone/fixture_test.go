package generated

import "testing"

func TestMilestoneFixtureShape(t *testing.T) {
	item := Milestone{ID: "fixture-milestone"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}

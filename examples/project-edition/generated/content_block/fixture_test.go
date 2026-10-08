package generated

import "testing"

func TestContentBlockFixtureShape(t *testing.T) {
	item := ContentBlock{ID: "fixture-content_block"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}

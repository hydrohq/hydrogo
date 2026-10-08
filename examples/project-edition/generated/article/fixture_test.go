package generated

import "testing"

func TestArticleFixtureShape(t *testing.T) {
	item := Article{ID: "fixture-article"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}

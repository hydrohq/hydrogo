package generated

import "testing"

func TestMediaArticleFixtureShape(t *testing.T) {
	item := MediaArticle{ID: "fixture-media_article"}
	if item.ID == "" {
		t.Fatal("fixture ID must not be empty")
	}
}

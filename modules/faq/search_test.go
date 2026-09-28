package faq

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
)

func TestSearchIsCaseInsensitiveAndPublishedOnly(t *testing.T) {
	items := []Question{{Base: sectoral.Base{Status: "published"}, QuestionText: "İade nasıl yapılır?", Answer: "Ürünü gönderin"}, {Base: sectoral.Base{Status: "draft"}, QuestionText: "İade taslağı", Answer: "Gizli"}}
	got := Search(items, "  İADE  ")
	if len(got) != 1 || got[0].Status != "published" {
		t.Fatalf("Search()=%+v", got)
	}
}

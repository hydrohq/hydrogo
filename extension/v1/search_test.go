package v1

import "testing"

func TestSearchContractBoundsAndStableResults(t *testing.T) {
	if err := (SearchQuery{Text: "haber", Locale: "tr-TR", Limit: 20}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (SearchQuery{Text: "", Limit: 1}).Validate(); err == nil {
		t.Fatal("expected empty query rejection")
	}
	response := StableSearchResults(SearchResponse{Total: 1, Items: []SearchResult{{ID: "z", Score: 1}, {ID: "a", Score: 1}, {ID: "b", Score: 2}}}, 2)
	if len(response.Items) != 2 || response.Items[0].ID != "b" || response.Items[1].ID != "a" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestStableSearchResultsHardBound(t *testing.T) {
	items := make([]SearchResult, MaxSearchLimit+1)
	got := StableSearchResults(SearchResponse{Items: items}, 0)
	if len(got.Items) != MaxSearchLimit {
		t.Fatalf("expected bound %d, got %d", MaxSearchLimit, len(got.Items))
	}
}

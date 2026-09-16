package search

import "testing"

func TestQueryValidateBounds(t *testing.T) {
	for name, query := range map[string]Query{
		"empty":    {},
		"too long": {Text: string(make([]byte, MaxQueryLength+1))},
		"limit":    {Text: "news", Limit: MaxLimit + 1},
		"offset":   {Text: "news", Offset: MaxOffset + 1},
	} {
		t.Run(name, func(t *testing.T) {
			if err := query.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if err := (Query{Text: "haber", Locale: "tr-TR", Limit: 20}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestStableResultsOrdersAndBounds(t *testing.T) {
	response := StableResults(Response{Total: 1, Items: []Result{{ID: "z", Score: 1}, {ID: "a", Score: 1}, {ID: "b", Score: 2}}}, 2)
	if len(response.Items) != 2 || response.Items[0].ID != "b" || response.Items[1].ID != "a" || response.Total != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
}

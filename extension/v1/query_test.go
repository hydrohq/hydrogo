package v1

import (
	"net/url"
	"testing"
)

func TestParseResourceQueryBoundsAndAllowlist(t *testing.T) {
	fields := map[string]QueryField{"title": {Name: "title", Operators: map[string]bool{"contains": true}}}
	q, err := ParseResourceQuery(url.Values{"where[title][contains]": {"roof"}, "limit": {"10"}, "sort": {"-title"}}, fields)
	if err != nil {
		t.Fatal(err)
	}
	if q.Limit != 10 || len(q.Conditions) != 1 || !q.Sort[0].Desc {
		t.Fatalf("unexpected query: %#v", q)
	}
	if _, err := ParseResourceQuery(url.Values{"where[title][equals]": {"roof"}}, fields); err == nil {
		t.Fatal("expected operator rejection")
	}
}

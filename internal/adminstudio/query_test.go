package adminstudio

import (
	"net/url"
	"testing"
)

func TestParseListQueryBoundsAndFilters(t *testing.T) {
	q := ParseListQuery(url.Values{"page": {"0"}, "per_page": {"1000"}, "search": {"  shoe "}, "sort": {"-created_at"}, "filter[status]": {" draft "}, "filter[]": {"ignored"}})
	if q.Page != 1 || q.PerPage != MaxPageSize || q.Search != "shoe" || q.Sort != "-created_at" || q.Filters["status"] != "draft" {
		t.Fatalf("unexpected query: %+v", q)
	}
}

func TestParseListQueryDefaults(t *testing.T) {
	q := ParseListQuery(nil)
	if q.Page != 1 || q.PerPage != DefaultPageSize || len(q.Filters) != 0 {
		t.Fatalf("unexpected defaults: %+v", q)
	}
}

func TestEncodeListQueryRoundTrip(t *testing.T) {
	q := ListQuery{Search: " shoe ", Filters: map[string]string{"status": "draft", "locale": "tr"}, Sort: "-created_at", Page: 2, PerPage: 50}
	encoded := EncodeListQuery(q)
	parsed, err := url.ParseQuery(encoded)
	if err != nil || parsed.Get("search") != "shoe" || parsed.Get("filter[locale]") != "tr" || parsed.Get("page") != "2" {
		t.Fatalf("encoded query = %q", encoded)
	}
}

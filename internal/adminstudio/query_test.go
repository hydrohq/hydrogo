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

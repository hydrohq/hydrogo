package adminstudio

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ListQuery is the bounded, provider-neutral query contract for admin lists.
// Modules translate it to their own repository predicates.
type ListQuery struct {
	Search  string
	Filters map[string]string
	Sort    string
	Page    int
	PerPage int
}

const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// ParseListQuery normalizes untrusted query parameters and applies hard bounds
// before they reach a repository or database adapter.
func ParseListQuery(values url.Values) ListQuery {
	page, _ := strconv.Atoi(values.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(values.Get("per_page"))
	if perPage < 1 {
		perPage = DefaultPageSize
	}
	if perPage > MaxPageSize {
		perPage = MaxPageSize
	}
	filters := make(map[string]string)
	for key, vals := range values {
		if !strings.HasPrefix(key, "filter[") || !strings.HasSuffix(key, "]") || len(vals) == 0 {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(key, "filter["), "]")
		if name != "" && len(name) <= 64 && strings.TrimSpace(vals[0]) != "" {
			filters[name] = strings.TrimSpace(vals[0])
		}
	}
	return ListQuery{Search: strings.TrimSpace(values.Get("search")), Filters: filters, Sort: strings.TrimSpace(values.Get("sort")), Page: page, PerPage: perPage}
}

// EncodeListQuery produces a stable query string for links/forms. Filter
// names and values are escaped by net/url and empty values are omitted.
func EncodeListQuery(q ListQuery) string {
	v := url.Values{}
	if q.Search != "" {
		v.Set("search", strings.TrimSpace(q.Search))
	}
	if q.Sort != "" {
		v.Set("sort", strings.TrimSpace(q.Sort))
	}
	if q.Page > 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PerPage > 0 && q.PerPage != DefaultPageSize {
		v.Set("per_page", strconv.Itoa(minInt(q.PerPage, MaxPageSize)))
	}
	keys := make([]string, 0, len(q.Filters))
	for key := range q.Filters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(q.Filters[key]) != "" {
			v.Set("filter["+key+"]", strings.TrimSpace(q.Filters[key]))
		}
	}
	return v.Encode()
}

// ConstrainListQuery applies module-owned allowlists before repository
// translation. A leading '-' denotes descending sort.
func ConstrainListQuery(q ListQuery, allowedSorts, allowedFilters []string) ListQuery {
	sortSet := make(map[string]struct{}, len(allowedSorts))
	for _, field := range allowedSorts {
		sortSet[field] = struct{}{}
	}
	base := strings.TrimPrefix(q.Sort, "-")
	if _, ok := sortSet[base]; !ok {
		q.Sort = ""
	}
	filterSet := make(map[string]struct{}, len(allowedFilters))
	for _, field := range allowedFilters {
		filterSet[field] = struct{}{}
	}
	for field := range q.Filters {
		if _, ok := filterSet[field]; !ok {
			delete(q.Filters, field)
		}
	}
	return q
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

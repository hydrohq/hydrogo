package adminstudio

import (
	"net/url"
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

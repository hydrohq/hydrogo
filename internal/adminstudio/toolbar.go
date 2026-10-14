package adminstudio

import (
	"context"
	"io"
	"sort"
	"strings"
)

type QueryToolbarData struct {
	Action        string
	Query         ListQuery
	FilterOptions map[string][]string
	SortOptions   []string
}

func RenderQueryToolbar(w io.Writer, data QueryToolbarData) error {
	if strings.TrimSpace(data.Action) == "" || !strings.HasPrefix(data.Action, "/") || strings.HasPrefix(data.Action, "//") {
		return ErrInvalidForm
	}
	if data.Query.PerPage < 1 || data.Query.PerPage > MaxPageSize {
		data.Query.PerPage = DefaultPageSize
	}
	return queryToolbarView(data).Render(context.Background(), w)
}

func sortedFilterNames(options map[string][]string) []string {
	names := make([]string, 0, len(options))
	for name := range options {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

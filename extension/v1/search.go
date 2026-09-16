package v1

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

const (
	MaxSearchQueryLength = 256
	MaxSearchLimit       = 100
	MaxSearchOffset      = 10000
)

type SearchQuery struct {
	Text   string `json:"text"`
	Locale string `json:"locale,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
type SearchResult struct {
	ID      string  `json:"id"`
	Score   float64 `json:"score"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet,omitempty"`
}
type SearchResponse struct {
	Items []SearchResult `json:"items"`
	Total int            `json:"total"`
}

// SearchProvider is implemented by an optional module or infrastructure adapter.
type SearchProvider interface {
	Search(context.Context, SearchQuery) (SearchResponse, error)
}

func (q SearchQuery) Validate() error {
	text := strings.TrimSpace(q.Text)
	if text == "" || len(text) > MaxSearchQueryLength {
		return fmt.Errorf("search text must be between 1 and %d characters", MaxSearchQueryLength)
	}
	if q.Locale != "" && canonical.Locale(q.Locale) == "" {
		return fmt.Errorf("invalid search locale")
	}
	if q.Limit < 0 || q.Limit > MaxSearchLimit {
		return fmt.Errorf("search limit must be between 0 and %d", MaxSearchLimit)
	}
	if q.Offset < 0 || q.Offset > MaxSearchOffset {
		return fmt.Errorf("search offset must be between 0 and %d", MaxSearchOffset)
	}
	return nil
}

// StableSearchResults prevents an untrusted provider from returning an
// unbounded or nondeterministically ordered page.
func StableSearchResults(response SearchResponse, limit int) SearchResponse {
	items := append([]SearchResult(nil), response.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].ID < items[j].ID
	})
	if limit <= 0 || limit > MaxSearchLimit {
		limit = MaxSearchLimit
	}
	if len(items) > limit {
		items = items[:limit]
	}
	if response.Total < len(items) {
		response.Total = len(items)
	}
	response.Items = items
	return response
}

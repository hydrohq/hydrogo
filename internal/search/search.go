// Package search defines the provider-neutral search boundary.
package search

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

const (
	MaxQueryLength = 256
	MaxLimit       = 100
	MaxOffset      = 10000
)

type Query struct {
	Text   string `json:"text"`
	Locale string `json:"locale,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

type Result struct {
	ID      string  `json:"id"`
	Score   float64 `json:"score"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet,omitempty"`
}

type Response struct {
	Items []Result `json:"items"`
	Total int      `json:"total"`
}

// Provider is implemented by a module or optional infrastructure adapter.
type Provider interface {
	Search(context.Context, Query) (Response, error)
}

func (q Query) Validate() error {
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" || len(q.Text) > MaxQueryLength {
		return fmt.Errorf("search text must be between 1 and %d characters", MaxQueryLength)
	}
	if q.Locale != "" && canonical.Locale(q.Locale) == "" {
		return fmt.Errorf("invalid search locale")
	}
	if q.Limit < 0 || q.Limit > MaxLimit {
		return fmt.Errorf("search limit must be between 0 and %d", MaxLimit)
	}
	if q.Offset < 0 || q.Offset > MaxOffset {
		return fmt.Errorf("search offset must be between 0 and %d", MaxOffset)
	}
	return nil
}

// StableResults bounds and orders provider output so clients receive
// deterministic pagination even when scores tie.
func StableResults(response Response, limit int) Response {
	items := append([]Result(nil), response.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].ID < items[j].ID
	})
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	if response.Total < len(items) {
		response.Total = len(items)
	}
	response.Items = items
	return response
}

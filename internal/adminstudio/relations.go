package adminstudio

import (
	"errors"
	"sort"
	"strings"
)

const (
	DefaultRelationLimit = 50
	MaxRelationLimit     = 100
)

var ErrInvalidRelationQuery = errors.New("invalid relation query")

// RelationQuery is a provider-neutral async relation lookup envelope. The
// owning module supplies the data source and authorization policy.
type RelationQuery struct {
	Collection string `json:"collection"`
	Query      string `json:"query"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

func ValidateRelationQuery(query RelationQuery) (RelationQuery, error) {
	collection := strings.TrimSpace(query.Collection)
	if collection == "" || len(collection) > 64 || strings.ContainsAny(collection, "/\\") {
		return RelationQuery{}, ErrInvalidRelationQuery
	}
	term := strings.TrimSpace(query.Query)
	if len(term) > 128 {
		return RelationQuery{}, ErrInvalidRelationQuery
	}
	cursor := strings.TrimSpace(query.Cursor)
	if len(cursor) > 256 {
		return RelationQuery{}, ErrInvalidRelationQuery
	}
	limit := query.Limit
	if limit == 0 {
		limit = DefaultRelationLimit
	}
	if limit < 1 || limit > MaxRelationLimit {
		return RelationQuery{}, ErrInvalidRelationQuery
	}
	return RelationQuery{Collection: collection, Query: term, Cursor: cursor, Limit: limit}, nil
}

func NormalizeRelationOptions(options []RelationOption, limit int) ([]RelationOption, error) {
	if limit < 1 || limit > MaxRelationLimit || len(options) > limit {
		return nil, ErrInvalidRelationQuery
	}
	seen := make(map[string]struct{}, len(options))
	result := make([]RelationOption, 0, len(options))
	for _, option := range options {
		id, label := strings.TrimSpace(option.ID), strings.TrimSpace(option.Label)
		if id == "" || label == "" || len(id) > 128 || len(label) > 256 {
			return nil, ErrInvalidRelationQuery
		}
		if _, exists := seen[id]; exists {
			return nil, ErrInvalidRelationQuery
		}
		seen[id] = struct{}{}
		result = append(result, RelationOption{ID: id, Label: label})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

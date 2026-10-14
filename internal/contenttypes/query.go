package contenttypes

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// parseAdminQuery translates the list query adminstudio.GeneratedAdminHandler
// sends to the record API:
//
//	where[or][i][field][contains]=q  search; any of these must match
//	where[field][equals]=v           filter; all of these must match
//	where[field][contains]=v
//	sort=field | sort=-field
//	limit=n&offset=n
//
// Field IDs are checked against the schema later by
// collections.Schema.ValidateQuery; here only the shape is checked.
func parseAdminQuery(values url.Values) (collections.Query, error) {
	limit, offset, err := pagingValues(values)
	if err != nil {
		return collections.Query{}, err
	}
	query := collections.Query{Limit: limit, Offset: offset}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entries := values[key]
		if len(entries) != 1 {
			return collections.Query{}, fmt.Errorf("%w: repeated parameter %q", collections.ErrInvalidQuery, key)
		}
		value := entries[0]
		switch {
		case key == "limit" || key == "offset":
		case key == "sort":
			query.Sort = collections.ParseSort(value)
		case strings.HasPrefix(key, "where[") && strings.HasSuffix(key, "]"):
			segments := strings.Split(key[len("where["):len(key)-1], "][")
			switch {
			case len(segments) == 2:
				query.Where = append(query.Where, collections.Condition{Field: segments[0], Op: collections.Operator(segments[1]), Value: value})
			case len(segments) == 4 && segments[0] == "or" && validIndex(segments[1]):
				query.AnyOf = append(query.AnyOf, collections.Condition{Field: segments[2], Op: collections.Operator(segments[3]), Value: value})
			default:
				return collections.Query{}, fmt.Errorf("%w: unsupported condition %q", collections.ErrInvalidQuery, key)
			}
		default:
			return collections.Query{}, fmt.Errorf("%w: unsupported parameter %q", collections.ErrInvalidQuery, key)
		}
	}
	if len(query.Where)+len(query.AnyOf) > collections.MaxQueryConditions {
		return collections.Query{}, fmt.Errorf("%w: too many conditions", collections.ErrInvalidQuery)
	}
	return query, nil
}

// parsePublicQuery reads the public API's list parameters:
//
//	filter[field]=v  exact match on a field marked filterable
//	search=q         substring match on any field marked searchable
//	sort=field | sort=-field (also created_at, updated_at)
//	limit=n&offset=n
//
// Other parameters are ignored.
func parsePublicQuery(values url.Values, definition Definition) (collections.Query, error) {
	limit, offset, err := pagingValues(values)
	if err != nil {
		return collections.Query{}, err
	}
	query := collections.Query{Limit: limit, Offset: offset}
	filterable, searchable := map[string]bool{}, []string{}
	for _, field := range definition.Fields {
		filterable[field.ID] = field.Filterable
		if field.Searchable {
			searchable = append(searchable, field.ID)
		}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entries := values[key]
		switch {
		case key == "sort" || key == "search" || strings.HasPrefix(key, "filter["):
			if len(entries) != 1 {
				return collections.Query{}, fmt.Errorf("%w: repeated parameter %q", collections.ErrInvalidQuery, key)
			}
		default:
			continue
		}
		value := entries[0]
		switch {
		case key == "sort":
			query.Sort = collections.ParseSort(value)
		case key == "search":
			if strings.TrimSpace(value) == "" {
				continue
			}
			if len(searchable) == 0 {
				return collections.Query{}, fmt.Errorf("%w: search is not enabled", collections.ErrInvalidQuery)
			}
			for _, field := range searchable {
				query.AnyOf = append(query.AnyOf, collections.Condition{Field: field, Op: collections.Contains, Value: strings.TrimSpace(value)})
			}
		case strings.HasSuffix(key, "]"):
			field := key[len("filter[") : len(key)-1]
			if !filterable[field] {
				return collections.Query{}, fmt.Errorf("%w: %q is not filterable", collections.ErrInvalidQuery, field)
			}
			query.Where = append(query.Where, collections.Condition{Field: field, Op: collections.Equals, Value: value})
		default:
			return collections.Query{}, fmt.Errorf("%w: unsupported parameter %q", collections.ErrInvalidQuery, key)
		}
	}
	if len(query.Where)+len(query.AnyOf) > collections.MaxQueryConditions {
		return collections.Query{}, fmt.Errorf("%w: too many conditions", collections.ErrInvalidQuery)
	}
	return query, nil
}

func validIndex(raw string) bool {
	index, err := strconv.Atoi(raw)
	return err == nil && index >= 0 && index < collections.MaxQueryConditions && strconv.Itoa(index) == raw
}

func pagingValues(values url.Values) (int, int, error) {
	limit, offset := 25, 0
	if raw := values.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return 0, 0, fmt.Errorf("%w: invalid limit", collections.ErrInvalidQuery)
		}
		limit = value
	}
	if raw := values.Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || value > collections.MaxCollectionRecords {
			return 0, 0, fmt.Errorf("%w: invalid offset", collections.ErrInvalidQuery)
		}
		offset = value
	}
	return limit, offset, nil
}

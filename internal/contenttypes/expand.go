package contenttypes

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// maxExpandFields bounds ?expand= so one request cannot fan out widely.
const maxExpandFields = 5

// expansion loads the records that relation fields point to, so a client can
// show "company: Acme" without one request per row. The result is added to
// each item as "expanded": {"company": {...}} (an array for many relations);
// the relation field itself keeps the plain ID.
type expansion struct {
	service collections.Service
	// target returns the definition a relation may be expanded into, and
	// whether it may be. The public API only expands into public types.
	target func(typeID string) (Definition, bool)
	// publishedOnly hides related drafts (the public API).
	publishedOnly bool
	// present renders a related record for the response.
	present func(Definition, collections.Record) map[string]any
}

// parse reads ?expand=field,field and returns the relation fields of
// the definition it names. Unknown fields, non-relations and relations
// whose target may not be expanded fail with ErrInvalidQuery.
func (e expansion) parse(values url.Values, definition Definition) ([]Field, error) {
	raw := values["expand"]
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) != 1 {
		return nil, fmt.Errorf("%w: repeated parameter \"expand\"", collections.ErrInvalidQuery)
	}
	var fields []Field
	seen := map[string]bool{}
	for _, name := range strings.Split(raw[0], ",") {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		var found *Field
		for i := range definition.Fields {
			if definition.Fields[i].ID == name {
				found = &definition.Fields[i]
			}
		}
		if found == nil || found.Kind != "relation" {
			return nil, fmt.Errorf("%w: cannot expand %q", collections.ErrInvalidQuery, name)
		}
		if _, ok := e.target(found.To); !ok {
			return nil, fmt.Errorf("%w: cannot expand %q", collections.ErrInvalidQuery, name)
		}
		fields = append(fields, *found)
	}
	if len(fields) > maxExpandFields {
		return nil, fmt.Errorf("%w: too many expand fields", collections.ErrInvalidQuery)
	}
	return fields, nil
}

// load returns, per record ID, the expanded value of each field.
func (e expansion) load(ctx context.Context, fields []Field, records []collections.Record) (map[string]map[string]any, error) {
	result := map[string]map[string]any{}
	for _, field := range fields {
		target, ok := e.target(field.To)
		if !ok {
			continue
		}
		wanted := map[string]bool{}
		for _, record := range records {
			for _, id := range relationIDs(record.Data[field.ID]) {
				wanted[id] = true
			}
		}
		ids := make([]string, 0, len(wanted))
		for id := range wanted {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		related := map[string]map[string]any{}
		for start := 0; start < len(ids); start += collections.MaxQueryIDs {
			end := min(start+collections.MaxQueryIDs, len(ids))
			query := collections.Query{Limit: collections.MaxQueryIDs, IDs: ids[start:end]}
			if target.Drafts && e.publishedOnly {
				query.Where = append(query.Where, publishedCondition())
			}
			found, err := e.service.QueryRecords(ctx, target.ID, query)
			if err != nil {
				return nil, err
			}
			for _, record := range found {
				related[record.ID] = e.present(target, record)
			}
		}
		for _, record := range records {
			ids := relationIDs(record.Data[field.ID])
			if len(ids) == 0 {
				continue
			}
			if result[record.ID] == nil {
				result[record.ID] = map[string]any{}
			}
			if !field.Many {
				if item, ok := related[ids[0]]; ok {
					result[record.ID][field.ID] = item
				}
				continue
			}
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				if item, ok := related[id]; ok {
					items = append(items, item)
				}
			}
			result[record.ID][field.ID] = items
		}
	}
	return result, nil
}

// relationIDs returns the record IDs held by a stored relation value.
func relationIDs(value any) []string {
	switch typed := value.(type) {
	case string:
		if typed != "" {
			return []string{typed}
		}
	case []any:
		ids := make([]string, 0, len(typed))
		for _, item := range typed {
			if id, ok := item.(string); ok && id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	}
	return nil
}

package contenttypes

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// AggregatePath is appended to a type's admin path.
const AggregatePath = "/_aggregate"

// Limits of one aggregation. Sums are exact decimals computed on the server
// from the matching records, so a request may not match more than
// MaxAggregateRecords (narrow it with filters) and may not form more than
// MaxAggregateGroups groups.
const (
	MaxAggregateRecords = 20000
	MaxAggregateGroups  = 1000
	maxAggregateSums    = 10
)

var errAggregate = errors.New("invalid aggregate")

type aggregateGroup struct {
	Key   any               `json:"key,omitempty"`
	Count int               `json:"count"`
	Sums  map[string]string `json:"sums,omitempty"`
}

// aggregate counts records and totals number fields, optionally per value of
// one field. The same filters as the list apply (filter[...], search,
// status), and so do row access and field permissions.
//
//	GET /api/v1/admin/content/deals/_aggregate?group_by=stage&sum=amount,tax&filter[owner]=x
//
// The answer is {data: {total: {count, sums}, groups: [{key, count, sums}]}},
// groups sorted by key; without group_by there are no groups.
func (a AdminAPI) aggregate(w http.ResponseWriter, r *http.Request) {
	definition, _, ok := a.resolve(w, r, false)
	if !ok {
		return
	}
	values := r.URL.Query()
	groupBy := values.Get("group_by")
	var sums []Field
	for _, id := range strings.FieldsFunc(values.Get("sum"), func(c rune) bool { return c == ',' }) {
		field, found := fieldByID(definition, id)
		if !found || field.Kind != "number" || len(sums) >= maxAggregateSums || !a.fieldReadable(r, field) {
			api.WriteError(w, r, http.StatusBadRequest, "sum needs number fields of this type", nil)
			return
		}
		sums = append(sums, field)
	}
	var group Field
	if groupBy != "" {
		field, found := fieldByID(definition, groupBy)
		if !found || field.Many || !a.fieldReadable(r, field) || !groupable(field.Kind) {
			api.WriteError(w, r, http.StatusBadRequest, "group_by needs a text, select, boolean, date or single relation field of this type", nil)
			return
		}
		group = field
	}
	query, err := parsePublicQuery(values, definition)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	if status := values.Get("status"); status != "" {
		if !definition.Drafts || (status != StatusDraft && status != StatusPublished) {
			api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
			return
		}
		query.Where = append(query.Where, collections.Condition{Field: StatusField, Op: collections.Equals, Value: status})
	}
	if err := a.checkFilters(r, definition, query); err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	if subject, restricted := a.restriction(r, definition); restricted {
		if subject == "" {
			api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		query.Where = append(query.Where, collections.Condition{Field: definition.Access.Owner, Op: collections.Equals, Value: subject})
	}
	query.Sort, query.Offset, query.Limit = nil, 0, 100

	type tally struct {
		count int
		sums  []*big.Rat
	}
	newTally := func() *tally {
		t := &tally{sums: make([]*big.Rat, len(sums))}
		for i := range t.sums {
			t.sums[i] = new(big.Rat)
		}
		return t
	}
	total, groups := newTally(), map[any]*tally{}
	scanned := 0
	for {
		records, err := a.Service.QueryRecords(r.Context(), definition.ID, query)
		if err != nil {
			writeServiceError(w, r, err, collections.Schema{})
			return
		}
		for _, record := range records {
			if scanned++; scanned > MaxAggregateRecords {
				api.WriteError(w, r, http.StatusUnprocessableEntity, fmt.Sprintf("more than %d records match; narrow the filters", MaxAggregateRecords), nil)
				return
			}
			targets := []*tally{total}
			if groupBy != "" {
				key := groupKey(record.Data[group.ID])
				entry := groups[key]
				if entry == nil {
					if len(groups) >= MaxAggregateGroups {
						api.WriteError(w, r, http.StatusUnprocessableEntity, fmt.Sprintf("more than %d groups", MaxAggregateGroups), nil)
						return
					}
					entry = newTally()
					groups[key] = entry
				}
				targets = append(targets, entry)
			}
			for _, target := range targets {
				target.count++
				for i, field := range sums {
					if number, ok := collections.ExactNumber(record.Data[field.ID]); ok {
						target.sums[i].Add(target.sums[i], number)
					}
				}
			}
		}
		if len(records) < query.Limit {
			break
		}
		cursor := collections.NextCursor(query, records[len(records)-1])
		decoded, err := collections.DecodeCursor(cursor)
		if err != nil {
			writeServiceError(w, r, err, collections.Schema{})
			return
		}
		query.After = &decoded
	}

	render := func(t *tally) map[string]string {
		if len(sums) == 0 {
			return nil
		}
		out := make(map[string]string, len(sums))
		for i, field := range sums {
			out[field.ID] = formatExact(t.sums[i], field.Decimals)
		}
		return out
	}
	result := map[string]any{"total": aggregateGroup{Count: total.count, Sums: render(total)}}
	if groupBy != "" {
		keys := make([]any, 0, len(groups))
		for key := range groups {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		list := make([]aggregateGroup, 0, len(keys))
		for _, key := range keys {
			entry := groups[key]
			list = append(list, aggregateGroup{Key: key, Count: entry.count, Sums: render(entry)})
		}
		result["groups"] = list
	}
	api.WriteData(w, http.StatusOK, result, map[string]any{"scanned": scanned})
}

func fieldByID(definition Definition, id string) (Field, bool) {
	for _, field := range definition.Fields {
		if field.ID == id {
			return field, true
		}
	}
	return Field{}, false
}

func groupable(kind string) bool {
	switch kind {
	case "text", "select", "boolean", "date", "relation":
		return true
	}
	return false
}

// groupKey turns a stored value into a comparable key; unset values group
// under "".
func groupKey(value any) any {
	switch typed := value.(type) {
	case string, bool:
		return typed
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}

// formatExact renders an exact sum with the field's decimals, or, for plain
// numbers, with as many places as needed (at most collections.MaxDecimals).
func formatExact(value *big.Rat, decimals *int) string {
	if decimals != nil {
		return value.FloatString(*decimals)
	}
	text := value.FloatString(collections.MaxDecimals)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}
	return text
}

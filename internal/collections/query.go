package collections

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidQuery reports a list query that names an unknown field, uses an
// operator the field kind does not support, or exceeds the query bounds.
var ErrInvalidQuery = errors.New("invalid collection query")

// Operator is a comparison a Condition applies to one field.
type Operator string

const (
	// Equals matches the stored value exactly (numbers and booleans by value).
	Equals Operator = "equals"
	// Contains is a case-insensitive substring match on text values.
	Contains Operator = "contains"
	// NotEquals matches records whose value differs, including records
	// that have no value for the field.
	NotEquals Operator = "ne"
	// GreaterThan, GreaterOrEqual, LessThan and LessOrEqual compare numbers
	// and dates (RFC 3339 strings compare in time order).
	GreaterThan    Operator = "gt"
	GreaterOrEqual Operator = "gte"
	LessThan       Operator = "lt"
	LessOrEqual    Operator = "lte"
	// In matches any of a comma separated list of values.
	In Operator = "in"
	// IsNull matches records with no value for the field (Value "true") or
	// with a value (Value "false").
	IsNull Operator = "null"
)

// System sort keys are record columns rather than fields in the record data.
const (
	SortCreatedAt = "created_at"
	SortUpdatedAt = "updated_at"
)

const (
	MaxQueryConditions = 32
	MaxQueryValueBytes = 256
	MaxInValues        = 50
	MaxQueryIDs        = 100
	// MaxScanLimit is the page size server-side scans (aggregations) may ask
	// for. The HTTP API still caps clients at 100; a scan of 10,000 records in
	// pages of 100 meant 100 round trips, 1.9 s on MySQL.
	MaxScanLimit = 1000
)

// Condition compares one schema field with a value. Value is the raw string
// from the request; ValidateQuery converts it to the field's kind.
type Condition struct {
	Field string
	Op    Operator
	Value string
	// Typed is set by ValidateQuery: string, float64 or bool.
	Typed any
	// Kind is set by ValidateQuery so storage can pick the right expression.
	Kind FieldKind
}

// Sort orders the result by a schema field or a system column.
type Sort struct {
	Field string
	Desc  bool
	// Kind is set by ValidateQuery; empty for system columns.
	Kind FieldKind
}

// AuthContext holds caller identity attributes used during row-level rule compilation.
type AuthContext struct {
	ID       string
	Email    string
	Role     string
	IsAdmin  bool
	IsMember bool
}

// Query is a bounded list query: every condition in Where must match, and
// when AnyOf is not empty at least one of its conditions must match too
// (used for search across several fields).
type Query struct {
	Limit  int
	Offset int
	Where  []Condition
	AnyOf  []Condition
	Sort   *Sort
	// IDs restricts the result to these record IDs (at most MaxQueryIDs).
	// Related records are loaded this way (see contenttypes expand).
	IDs []string
	// After continues a list after a cursor from NextCursor; it replaces
	// Offset and only works with the default sort or a created_at/updated_at sort.
	After *Cursor
	// AccessRule is a declarative row-level access rule pushed into SQL WHERE.
	AccessRule  string
	AuthContext AuthContext
}

// ParseSort reads "field" or "-field".
func ParseSort(raw string) *Sort {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "-") {
		return &Sort{Field: raw[1:], Desc: true}
	}
	return &Sort{Field: strings.TrimPrefix(raw, "+")}
}

// ValidateQuery checks every field ID against the schema and converts the
// condition values. Storage adapters only accept validated queries, so field
// IDs reaching SQL are always known schema identifiers.
func (s Schema) ValidateQuery(q Query) (Query, error) {
	if q.Limit < 1 || q.Limit > MaxScanLimit || q.Offset < 0 || q.Offset > MaxListOffset {
		return Query{}, fmt.Errorf("%w: limit or offset out of range", ErrInvalidQuery)
	}
	if len(q.Where)+len(q.AnyOf) > MaxQueryConditions {
		return Query{}, fmt.Errorf("%w: too many conditions", ErrInvalidQuery)
	}
	fields := make(map[string]Field, len(s.Fields))
	for _, field := range s.Fields {
		fields[field.ID] = field
	}
	if len(q.IDs) > MaxQueryIDs {
		return Query{}, fmt.Errorf("%w: too many ids", ErrInvalidQuery)
	}
	for _, id := range q.IDs {
		// Generated record IDs have the identifier shape (sqlstore newID).
		if !identifier.MatchString(id) {
			return Query{}, fmt.Errorf("%w: invalid record id", ErrInvalidQuery)
		}
	}
	if q.After != nil {
		if err := validateCursor(q.After, q); err != nil {
			return Query{}, err
		}
	}
	result := Query{
		Limit:       q.Limit,
		Offset:      q.Offset,
		IDs:         q.IDs,
		After:       q.After,
		AccessRule:  q.AccessRule,
		AuthContext: q.AuthContext,
	}
	for _, group := range []struct {
		in  []Condition
		out *[]Condition
	}{{q.Where, &result.Where}, {q.AnyOf, &result.AnyOf}} {
		for _, condition := range group.in {
			checked, err := validateCondition(fields, condition)
			if err != nil {
				return Query{}, err
			}
			*group.out = append(*group.out, checked)
		}
	}
	if q.Sort != nil {
		sort := *q.Sort
		switch sort.Field {
		case SortCreatedAt, SortUpdatedAt:
			sort.Kind = ""
		default:
			field, ok := fields[sort.Field]
			if !ok || !identifier.MatchString(sort.Field) {
				return Query{}, fmt.Errorf("%w: cannot sort by %q", ErrInvalidQuery, sort.Field)
			}
			sort.Kind = field.Kind
		}
		result.Sort = &sort
	}
	return result, nil
}

func validateCondition(fields map[string]Field, condition Condition) (Condition, error) {
	field, ok := fields[condition.Field]
	if !ok || !identifier.MatchString(condition.Field) {
		return Condition{}, fmt.Errorf("%w: unknown field %q", ErrInvalidQuery, condition.Field)
	}
	if len(condition.Value) > MaxQueryValueBytes {
		return Condition{}, fmt.Errorf("%w: value too long", ErrInvalidQuery)
	}
	condition.Kind = field.Kind
	switch condition.Op {
	case Contains:
		switch field.Kind {
		case Text, RichText, Select:
			condition.Typed = condition.Value
		default:
			return Condition{}, fmt.Errorf("%w: %q does not support contains", ErrInvalidQuery, field.ID)
		}
	case Equals, NotEquals:
		typed, err := scalarValue(field, condition.Value)
		if err != nil {
			return Condition{}, err
		}
		condition.Typed = typed
	case GreaterThan, GreaterOrEqual, LessThan, LessOrEqual:
		if field.Kind != Number && field.Kind != DateTime {
			return Condition{}, fmt.Errorf("%w: %q does not support %s", ErrInvalidQuery, field.ID, condition.Op)
		}
		typed, err := scalarValue(field, condition.Value)
		if err != nil {
			return Condition{}, err
		}
		condition.Typed = typed
	case In:
		parts := strings.Split(condition.Value, ",")
		if len(parts) > MaxInValues {
			return Condition{}, fmt.Errorf("%w: too many values for %q", ErrInvalidQuery, field.ID)
		}
		values := make([]any, 0, len(parts))
		for _, part := range parts {
			typed, err := scalarValue(field, strings.TrimSpace(part))
			if err != nil {
				return Condition{}, err
			}
			values = append(values, typed)
		}
		condition.Typed = values
	case IsNull:
		if _, err := scalarValue(Field{ID: field.ID, Kind: Boolean}, condition.Value); err != nil {
			return Condition{}, err
		}
		if field.Kind == RelationMany || field.Kind == MultiSelect {
			return Condition{}, fmt.Errorf("%w: %q does not support null", ErrInvalidQuery, field.ID)
		}
		flag, _ := strconv.ParseBool(strings.TrimSpace(condition.Value))
		condition.Typed = flag
	default:
		return Condition{}, fmt.Errorf("%w: unknown operator %q", ErrInvalidQuery, condition.Op)
	}
	return condition, nil
}

// scalarValue converts a raw query value to the type the field's kind is
// compared as: string, float64 or bool.
func scalarValue(field Field, raw string) (any, error) {
	switch field.Kind {
	case Text, RichText, Select, Media, Relation:
		return raw, nil
	case DateTime:
		// Records store UTC RFC 3339 timestamps (see contenttypes.toStorage),
		// so a date or local time is converted the same way before comparing.
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
			if parsed, err := time.Parse(layout, strings.TrimSpace(raw)); err == nil {
				return parsed.UTC().Format(time.RFC3339), nil
			}
		}
		return nil, fmt.Errorf("%w: %q needs a date or RFC 3339 timestamp", ErrInvalidQuery, field.ID)
	case Number:
		number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %q needs a number", ErrInvalidQuery, field.ID)
		}
		return number, nil
	case Boolean:
		value, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("%w: %q needs true or false", ErrInvalidQuery, field.ID)
		}
		return value, nil
	}
	return nil, fmt.Errorf("%w: %q cannot be compared", ErrInvalidQuery, field.ID)
}

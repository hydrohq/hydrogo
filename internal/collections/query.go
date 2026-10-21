package collections

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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
)

// System sort keys are record columns rather than fields in the record data.
const (
	SortCreatedAt = "created_at"
	SortUpdatedAt = "updated_at"
)

const (
	MaxQueryConditions = 32
	MaxQueryValueBytes = 256
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

// Query is a bounded list query: every condition in Where must match, and
// when AnyOf is not empty at least one of its conditions must match too
// (used for search across several fields).
type Query struct {
	Limit  int
	Offset int
	Where  []Condition
	AnyOf  []Condition
	Sort   *Sort
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
	if q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > MaxListOffset {
		return Query{}, fmt.Errorf("%w: limit or offset out of range", ErrInvalidQuery)
	}
	if len(q.Where)+len(q.AnyOf) > MaxQueryConditions {
		return Query{}, fmt.Errorf("%w: too many conditions", ErrInvalidQuery)
	}
	fields := make(map[string]Field, len(s.Fields))
	for _, field := range s.Fields {
		fields[field.ID] = field
	}
	result := Query{Limit: q.Limit, Offset: q.Offset}
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
	case Equals:
		switch field.Kind {
		case Text, RichText, Select, DateTime, Media, Relation:
			condition.Typed = condition.Value
		case Number:
			number, err := strconv.ParseFloat(strings.TrimSpace(condition.Value), 64)
			if err != nil {
				return Condition{}, fmt.Errorf("%w: %q needs a number", ErrInvalidQuery, field.ID)
			}
			condition.Typed = number
		case Boolean:
			value, err := strconv.ParseBool(strings.TrimSpace(condition.Value))
			if err != nil {
				return Condition{}, fmt.Errorf("%w: %q needs true or false", ErrInvalidQuery, field.ID)
			}
			condition.Typed = value
		default:
			return Condition{}, fmt.Errorf("%w: %q does not support equals", ErrInvalidQuery, field.ID)
		}
	default:
		return Condition{}, fmt.Errorf("%w: unknown operator %q", ErrInvalidQuery, condition.Op)
	}
	return condition, nil
}

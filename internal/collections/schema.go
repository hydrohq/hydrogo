// Package collections stores user-defined collection schemas and records in
// bounded, pre-migrated tables. A schema never executes SQL or creates DDL.
package collections

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidSchema      = errors.New("invalid collection schema")
	ErrInvalidRecord      = errors.New("record does not match collection schema")
	ErrUniqueValue        = errors.New("a unique field value already exists")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrRecordNotFound     = errors.New("record not found")
	ErrVersionConflict    = errors.New("record was changed since it was loaded")
)

const (
	MaxCollections    = 100
	MaxFields         = 64
	MaxRecordBytes    = 64 << 10
	MaxFieldTextBytes = 16 << 10
	// DefaultMaxRecords is the per-type, per-tenant record cap used when the
	// store has no configured limit (limits.records_per_type).
	DefaultMaxRecords = 100000
	// MaxListOffset bounds ?offset= so deep paging stays cheap.
	MaxListOffset = 10000
	// MaxDecimals is the most decimal places a number field can fix.
	MaxDecimals = 8
	// maxExactDigits keeps decimal values within what a float64 holds exactly.
	maxExactDigits = 15
)

type FieldKind string

const (
	Text         FieldKind = "text"
	RichText     FieldKind = "richtext"
	Number       FieldKind = "number"
	Boolean      FieldKind = "boolean"
	DateTime     FieldKind = "datetime"
	Select       FieldKind = "select"
	MultiSelect  FieldKind = "multi_select"
	Media        FieldKind = "media"
	Relation     FieldKind = "relation"
	RelationMany FieldKind = "relation_many"
)

type Field struct {
	ID       string    `json:"id"`
	Label    string    `json:"label"`
	Kind     FieldKind `json:"kind"`
	Required bool      `json:"required,omitempty"`
	Unique   bool      `json:"unique,omitempty"`
	Min      *float64  `json:"min,omitempty"`
	Max      *float64  `json:"max,omitempty"`
	Options  []string  `json:"options,omitempty"`
	Relation string    `json:"relation,omitempty"`
	// Localized fields keep one value per locale: the default locale's value
	// in the record, the others as translations (see Service.SaveTranslation).
	// Only text, rich text, media and select fields can be localized.
	Localized bool `json:"localized,omitempty"`
	// Indexed asks the store for a database index that serves filters and
	// lookups on this field. Only PostgreSQL builds them today; the flag is
	// a hint and never changes results.
	Indexed bool `json:"indexed,omitempty"`
	// Sequence makes the server number new records (see Sequence). The value
	// is assigned on create, never changes, and any client value is ignored.
	Sequence *Sequence `json:"sequence,omitempty"`
	// Decimals fixes the number of decimal places of a number field (money:
	// 2). Values with more places, or too large to be held exactly, are
	// rejected instead of being rounded.
	Decimals *int `json:"decimals,omitempty"`
	// Computed makes the server calculate the field from other number fields
	// on every save (see Expr): the value is stored, a client value is
	// ignored. Needs Decimals, which defines the rounding.
	Computed string `json:"computed,omitempty"`
	// Rollup makes the server total the records of another type that point
	// at this one (see Rollup). Needs Decimals; read-only for clients.
	Rollup *Rollup `json:"rollup,omitempty"`
}

type Schema struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Description string  `json:"description,omitempty"`
	Fields      []Field `json:"fields"`
	// Revisions keeps a snapshot of every saved version of a record (see
	// MaxRevisionsPerRecord).
	Revisions bool `json:"revisions,omitempty"`
	// Rules validate and lock records; States restricts how a select field
	// changes. See Rule and StateMachine.
	Rules  []Rule        `json:"rules,omitempty"`
	States *StateMachine `json:"states,omitempty"`
	// RollupParents lists the types that total this type's records; set by
	// the loader of the types (contenttypes.Schemas), not written by hand.
	RollupParents []string `json:"rollup_parents,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func (s Schema) Validate() error {
	if !identifier.MatchString(s.ID) || strings.TrimSpace(s.Label) == "" || len(s.Label) > 128 || len(s.Description) > 1000 || len(s.Fields) == 0 || len(s.Fields) > MaxFields {
		return ErrInvalidSchema
	}
	seen := make(map[string]struct{}, len(s.Fields))
	for _, field := range s.Fields {
		if !identifier.MatchString(field.ID) || strings.TrimSpace(field.Label) == "" || len(field.Label) > 128 {
			return ErrInvalidSchema
		}
		if _, exists := seen[field.ID]; exists {
			return ErrInvalidSchema
		}
		seen[field.ID] = struct{}{}
		switch field.Kind {
		case Text, RichText, Number, Boolean, DateTime, Select, MultiSelect, Media:
			if field.Relation != "" {
				return ErrInvalidSchema
			}
		case Relation, RelationMany:
			if !identifier.MatchString(field.Relation) {
				return ErrInvalidSchema
			}
		default:
			return ErrInvalidSchema
		}
		if field.Decimals != nil && (field.Kind != Number || *field.Decimals < 0 || *field.Decimals > MaxDecimals) {
			return ErrInvalidSchema
		}
		if field.Computed != "" {
			if err := s.validateComputed(field); err != nil {
				return err
			}
		}
		if field.Rollup != nil && (field.Kind != Number || field.Decimals == nil || field.Computed != "" || field.Required || field.Unique || field.Localized || field.Sequence != nil || field.Rollup.validate() != nil) {
			return ErrInvalidSchema
		}
		if field.Sequence != nil && (field.Kind != Text || field.Localized || field.Sequence.validate() != nil) {
			return ErrInvalidSchema
		}
		if field.Localized && (field.Unique || !localizable(field.Kind)) {
			return ErrInvalidSchema
		}
		if field.Min != nil && math.IsNaN(*field.Min) || field.Max != nil && math.IsNaN(*field.Max) || field.Min != nil && field.Max != nil && *field.Min > *field.Max {
			return ErrInvalidSchema
		}
		if field.Kind == Select || field.Kind == MultiSelect {
			if len(field.Options) == 0 || len(field.Options) > 100 {
				return ErrInvalidSchema
			}
			options := make(map[string]struct{}, len(field.Options))
			for _, option := range field.Options {
				if strings.TrimSpace(option) == "" || len(option) > 256 {
					return ErrInvalidSchema
				}
				if _, exists := options[option]; exists {
					return ErrInvalidSchema
				}
				options[option] = struct{}{}
			}
		} else if len(field.Options) > 0 {
			return ErrInvalidSchema
		}
	}
	return s.validateRulesAndStates()
}

func (s Schema) ValidateRecord(data map[string]any) (map[string]any, error) {
	if err := s.Validate(); err != nil || len(data) > MaxFields {
		return nil, ErrInvalidRecord
	}
	allowed := make(map[string]Field, len(s.Fields))
	for _, field := range s.Fields {
		allowed[field.ID] = field
	}
	for id := range data {
		if _, ok := allowed[id]; !ok {
			return nil, ErrInvalidRecord
		}
	}
	clean := make(map[string]any, len(data))
	for _, field := range s.Fields {
		if field.Sequence != nil || field.Computed != "" || field.Rollup != nil || (s.States != nil && s.States.Field == field.ID) {
			// Numbered by the server on create (Service.saveRecord) or
			// calculated below.
			continue
		}
		value, present := data[field.ID]
		if !present || value == nil || value == "" {
			if field.Required {
				return nil, fieldError(field.ID)
			}
			continue
		}
		if err := validateValue(field, value); err != nil {
			return nil, fieldError(field.ID)
		}
		clean[field.ID] = value
	}
	if err := s.applyComputed(clean); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(clean)
	if err != nil || len(encoded) > MaxRecordBytes {
		return nil, ErrInvalidRecord
	}
	return clean, nil
}

func validateValue(field Field, value any) error {
	switch field.Kind {
	case Text, RichText, Media, Relation:
		text, ok := value.(string)
		if !ok || len(text) > MaxFieldTextBytes {
			return ErrInvalidRecord
		}
		if field.Kind == Relation && !identifier.MatchString(text) {
			return ErrInvalidRecord
		}
	case Number:
		number, ok := numeric(value)
		if !ok || field.Min != nil && number < *field.Min || field.Max != nil && number > *field.Max {
			return ErrInvalidRecord
		}
		if field.Decimals != nil && !fitsDecimals(number, *field.Decimals) {
			return ErrInvalidRecord
		}
	case Boolean:
		if _, ok := value.(bool); !ok {
			return ErrInvalidRecord
		}
	case DateTime:
		text, ok := value.(string)
		if !ok {
			return ErrInvalidRecord
		}
		if _, err := time.Parse(time.RFC3339, text); err != nil {
			return ErrInvalidRecord
		}
	case Select:
		text, ok := value.(string)
		if !ok || !contains(field.Options, text) {
			return ErrInvalidRecord
		}
	case MultiSelect:
		items, ok := value.([]any)
		if !ok || len(items) > len(field.Options) {
			return ErrInvalidRecord
		}
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			text, ok := item.(string)
			if !ok || !contains(field.Options, text) {
				return ErrInvalidRecord
			}
			if _, exists := seen[text]; exists {
				return ErrInvalidRecord
			}
			seen[text] = struct{}{}
		}
	case RelationMany:
		items, ok := value.([]any)
		if !ok || len(items) > 100 {
			return ErrInvalidRecord
		}
		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			id, ok := item.(string)
			if !ok || !identifier.MatchString(id) {
				return ErrInvalidRecord
			}
			if _, exists := seen[id]; exists {
				return ErrInvalidRecord
			}
			seen[id] = struct{}{}
		}
	default:
		return ErrInvalidRecord
	}
	return nil
}

// fitsDecimals reports whether n has at most places decimal places and is
// small enough to be held exactly (at most maxExactDigits significant digits
// once scaled), so money is never silently rounded.
func fitsDecimals(n float64, places int) bool {
	text := strconv.FormatFloat(math.Abs(n), 'f', -1, 64)
	whole, fraction, _ := strings.Cut(text, ".")
	return len(fraction) <= places && len(whole)+places <= maxExactDigits
}

func numeric(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	case float32:
		return float64(number), !math.IsNaN(float64(number)) && !math.IsInf(float64(number), 0)
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	default:
		return 0, false
	}
}

func fieldError(id string) error { return fmt.Errorf("%w: %s", ErrInvalidRecord, id) }
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

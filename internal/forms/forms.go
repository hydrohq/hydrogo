// Package forms defines a bounded, provider-neutral form contract.
package forms

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	MaxFields       = 64
	MaxOptions      = 100
	MaxIDLength     = 64
	MaxLabelLength  = 256
	MaxValueLength  = 4096
	MaxFormIDLength = 128
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type FieldType string

const (
	Text     FieldType = "text"
	Email    FieldType = "email"
	Textarea FieldType = "textarea"
	Number   FieldType = "number"
	Checkbox FieldType = "checkbox"
	Select   FieldType = "select"
)

type Field struct {
	ID       string    `json:"id"`
	Type     FieldType `json:"type"`
	Label    string    `json:"label"`
	Required bool      `json:"required"`
	Options  []string  `json:"options,omitempty"`
}

type Form struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Fields []Field `json:"fields"`
}

func (f Form) Validate() error {
	if !idPattern.MatchString(f.ID) || len(f.ID) > MaxFormIDLength || strings.TrimSpace(f.Title) == "" || len(f.Title) > MaxLabelLength {
		return fmt.Errorf("invalid form metadata")
	}
	if len(f.Fields) == 0 || len(f.Fields) > MaxFields {
		return fmt.Errorf("form fields must contain 1-%d fields", MaxFields)
	}
	seen := make(map[string]struct{}, len(f.Fields))
	for _, field := range f.Fields {
		if !idPattern.MatchString(field.ID) || len(field.ID) > MaxIDLength || strings.TrimSpace(field.Label) == "" || len(field.Label) > MaxLabelLength {
			return fmt.Errorf("invalid form field %q", field.ID)
		}
		if _, ok := seen[field.ID]; ok {
			return fmt.Errorf("duplicate form field %q", field.ID)
		}
		seen[field.ID] = struct{}{}
		switch field.Type {
		case Text, Email, Textarea, Number, Checkbox:
			if len(field.Options) != 0 {
				return fmt.Errorf("field %q cannot have options", field.ID)
			}
		case Select:
			if len(field.Options) == 0 || len(field.Options) > MaxOptions {
				return fmt.Errorf("field %q has invalid options", field.ID)
			}
			optionSeen := map[string]struct{}{}
			for _, option := range field.Options {
				if strings.TrimSpace(option) == "" || len(option) > MaxLabelLength {
					return fmt.Errorf("field %q has invalid option", field.ID)
				}
				if _, ok := optionSeen[option]; ok {
					return fmt.Errorf("field %q repeats option", field.ID)
				}
				optionSeen[option] = struct{}{}
			}
		default:
			return fmt.Errorf("field %q has unsupported type", field.ID)
		}
	}
	return nil
}

// NormalizeSubmission validates and trims values without logging or persisting
// them. The caller owns consent, retention, inbox, and delivery policy.
func (f Form) NormalizeSubmission(values map[string]string) (map[string]string, map[string]string, error) {
	if err := f.Validate(); err != nil {
		return nil, nil, err
	}
	allowed := make(map[string]Field, len(f.Fields))
	for _, field := range f.Fields {
		allowed[field.ID] = field
	}
	clean := make(map[string]string, len(values))
	errors := make(map[string]string)
	for key, value := range values {
		field, ok := allowed[key]
		if !ok {
			errors[key] = "unknown-field"
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) > MaxValueLength {
			errors[key] = "value-too-large"
			continue
		}
		if field.Type == Select && !contains(field.Options, value) {
			errors[key] = "invalid-option"
			continue
		}
		clean[key] = value
	}
	for _, field := range f.Fields {
		if field.Required && clean[field.ID] == "" {
			errors[field.ID] = "required"
		}
	}
	if len(errors) > 0 {
		return clean, errors, nil
	}
	return clean, nil, nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// StableFields returns a copy sorted by field ID for generated manifests.
func StableFields(fields []Field) []Field {
	result := append([]Field(nil), fields...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

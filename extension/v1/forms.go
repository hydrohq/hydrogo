package v1

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	MaxFormFields        = 64
	MaxFormOptions       = 100
	MaxFormIDLength      = 128
	MaxFormFieldIDLength = 64
	MaxFormLabelLength   = 256
	MaxFormValueLength   = 4096
)

var formIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

type FormFieldType string

const (
	FormText     FormFieldType = "text"
	FormEmail    FormFieldType = "email"
	FormTextarea FormFieldType = "textarea"
	FormNumber   FormFieldType = "number"
	FormCheckbox FormFieldType = "checkbox"
	FormSelect   FormFieldType = "select"
)

type FormField struct {
	ID       string        `json:"id"`
	Type     FormFieldType `json:"type"`
	Label    string        `json:"label"`
	Required bool          `json:"required"`
	Options  []string      `json:"options,omitempty"`
}
type FormDefinition struct {
	ID     string      `json:"id"`
	Title  string      `json:"title"`
	Fields []FormField `json:"fields"`
}

func (f FormDefinition) Validate() error {
	if !formIDPattern.MatchString(f.ID) || len(f.ID) > MaxFormIDLength || strings.TrimSpace(f.Title) == "" || len(f.Title) > MaxFormLabelLength {
		return fmt.Errorf("invalid form metadata")
	}
	if len(f.Fields) == 0 || len(f.Fields) > MaxFormFields {
		return fmt.Errorf("form fields must contain 1-%d fields", MaxFormFields)
	}
	seen := make(map[string]struct{}, len(f.Fields))
	for _, field := range f.Fields {
		if !formIDPattern.MatchString(field.ID) || len(field.ID) > MaxFormFieldIDLength || strings.TrimSpace(field.Label) == "" || len(field.Label) > MaxFormLabelLength {
			return fmt.Errorf("invalid form field %q", field.ID)
		}
		if _, ok := seen[field.ID]; ok {
			return fmt.Errorf("duplicate form field %q", field.ID)
		}
		seen[field.ID] = struct{}{}
		switch field.Type {
		case FormText, FormEmail, FormTextarea, FormNumber, FormCheckbox:
			if len(field.Options) != 0 {
				return fmt.Errorf("field %q cannot have options", field.ID)
			}
		case FormSelect:
			if len(field.Options) == 0 || len(field.Options) > MaxFormOptions {
				return fmt.Errorf("field %q has invalid options", field.ID)
			}
			options := map[string]struct{}{}
			for _, option := range field.Options {
				if strings.TrimSpace(option) == "" || len(option) > MaxFormLabelLength {
					return fmt.Errorf("field %q has invalid option", field.ID)
				}
				if _, ok := options[option]; ok {
					return fmt.Errorf("field %q repeats option", field.ID)
				}
				options[option] = struct{}{}
			}
		default:
			return fmt.Errorf("field %q has unsupported type", field.ID)
		}
	}
	return nil
}

// NormalizeFormSubmission validates and trims values. Persistence, consent,
// retention and inbox delivery remain Client Edition/module policies.
func (f FormDefinition) NormalizeFormSubmission(values map[string]string) (map[string]string, map[string]string, error) {
	if err := f.Validate(); err != nil {
		return nil, nil, err
	}
	allowed := make(map[string]FormField, len(f.Fields))
	for _, field := range f.Fields {
		allowed[field.ID] = field
	}
	clean := make(map[string]string, len(values))
	problems := map[string]string{}
	for key, value := range values {
		field, ok := allowed[key]
		if !ok {
			problems[key] = "unknown-field"
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) > MaxFormValueLength {
			problems[key] = "value-too-large"
			continue
		}
		if field.Type == FormSelect && !formContains(field.Options, value) {
			problems[key] = "invalid-option"
			continue
		}
		clean[key] = value
	}
	for _, field := range f.Fields {
		if field.Required && clean[field.ID] == "" {
			problems[field.ID] = "required"
		}
	}
	if len(problems) > 0 {
		return clean, problems, nil
	}
	return clean, nil, nil
}

func formContains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func StableFormFields(fields []FormField) []FormField {
	result := append([]FormField(nil), fields...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

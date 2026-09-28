// Package forms defines a bounded, provider-neutral form contract.
package forms

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
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
var googleAdsConversionIDPattern = regexp.MustCompile(`^AW-[0-9]{1,20}$`)
var metaPixelEventPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

type FieldType string

const (
	Text            FieldType = "text"
	Email           FieldType = "email"
	Textarea        FieldType = "textarea"
	Number          FieldType = "number"
	Checkbox        FieldType = "checkbox"
	Select          FieldType = "select"
	Document        FieldType = "document"
	Phone           FieldType = "phone"
	CascadingSelect FieldType = "cascading_select"
)

type Field struct {
	ID             string              `json:"id"`
	Type           FieldType           `json:"type"`
	Label          string              `json:"label"`
	Required       bool                `json:"required"`
	Options        []string            `json:"options,omitempty"`
	Step           int                 `json:"step,omitempty"`
	DependsOn      string              `json:"depends_on,omitempty"`
	CascadeOptions map[string][]string `json:"cascade_options,omitempty"`
}

// AutoresponderBodyFor adds the configured response-time notice without
// changing the normal-hours response.
func (f Form) AutoresponderBodyFor(now time.Time) (string, error) {
	body := f.AutoresponderBody
	if f.WorkingHours == nil {
		return body, nil
	}
	notice, after, err := f.WorkingHours.AfterHoursNotice(now)
	if err != nil {
		return "", err
	}
	if after {
		if body != "" {
			body += "\n\n"
		}
		body += notice
	}
	return body, nil
}

type Form struct {
	ID                       string        `json:"id"`
	Title                    string        `json:"title"`
	Fields                   []Field       `json:"fields"`
	NotificationRecipients   []string      `json:"notification_recipients,omitempty"`
	Autoresponder            bool          `json:"autoresponder,omitempty"`
	TurnstileEnabled         bool          `json:"turnstile_enabled,omitempty"`
	AutoresponderSubject     string        `json:"autoresponder_subject,omitempty"`
	AutoresponderBody        string        `json:"autoresponder_body,omitempty"`
	ConversionTracking       bool          `json:"conversion_tracking,omitempty"`
	GoogleAdsConversionID    string        `json:"google_ads_conversion_id,omitempty"`
	GoogleAdsConversionLabel string        `json:"google_ads_conversion_label,omitempty"`
	MetaPixelEventName       string        `json:"meta_pixel_event_name,omitempty"`
	WorkingHours             *WorkingHours `json:"working_hours,omitempty"`
	MultiStep                bool          `json:"multi_step,omitempty"`
}

func (f Form) AfterHoursNotice(now time.Time) (string, bool, error) {
	if f.WorkingHours == nil {
		return "", false, nil
	}
	return f.WorkingHours.AfterHoursNotice(now)
}

func (f Form) PublicWorkingHoursNotice(now time.Time) string {
	notice, after, err := f.AfterHoursNotice(now)
	if err != nil || !after {
		return ""
	}
	return notice
}

func (f Form) Validate() error {
	if !idPattern.MatchString(f.ID) || len(f.ID) > MaxFormIDLength || strings.TrimSpace(f.Title) == "" || len(f.Title) > MaxLabelLength {
		return fmt.Errorf("invalid form metadata")
	}
	if len(f.Fields) == 0 || len(f.Fields) > MaxFields {
		return fmt.Errorf("form fields must contain 1-%d fields", MaxFields)
	}
	if len(f.NotificationRecipients) > 10 || len(f.AutoresponderSubject) > MaxLabelLength || len(f.AutoresponderBody) > 16<<10 || strings.ContainsAny(f.AutoresponderSubject, "\r\n") {
		return fmt.Errorf("invalid form notification settings")
	}
	if f.WorkingHours != nil {
		if err := f.WorkingHours.Validate(); err != nil {
			return err
		}
	}
	if len(f.GoogleAdsConversionID) > 32 || (f.GoogleAdsConversionID != "" && !googleAdsConversionIDPattern.MatchString(f.GoogleAdsConversionID)) || len(f.GoogleAdsConversionLabel) > 128 || strings.ContainsAny(f.GoogleAdsConversionLabel, "\r\n") || len(f.MetaPixelEventName) > 64 || (f.MetaPixelEventName != "" && !metaPixelEventPattern.MatchString(f.MetaPixelEventName)) {
		return fmt.Errorf("invalid conversion tracking settings")
	}
	if !f.ConversionTracking && (f.GoogleAdsConversionID != "" || f.GoogleAdsConversionLabel != "" || f.MetaPixelEventName != "") {
		return fmt.Errorf("conversion destinations require conversion tracking to be enabled")
	}
	for _, recipient := range f.NotificationRecipients {
		parsed, err := mail.ParseAddress(strings.TrimSpace(recipient))
		if err != nil || parsed.Address != strings.TrimSpace(recipient) || len(recipient) > 320 {
			return fmt.Errorf("invalid form notification recipient")
		}
	}
	seen := make(map[string]struct{}, len(f.Fields))
	fieldTypes := make(map[string]FieldType, len(f.Fields))
	steps := make(map[int]struct{})
	hasEmail := false
	documentFields := 0
	for _, field := range f.Fields {
		if field.Step < 0 || field.Step > 12 {
			return fmt.Errorf("invalid form step")
		}
		if !idPattern.MatchString(field.ID) || len(field.ID) > MaxIDLength || strings.TrimSpace(field.Label) == "" || len(field.Label) > MaxLabelLength {
			return fmt.Errorf("invalid form field %q", field.ID)
		}
		if _, ok := seen[field.ID]; ok {
			return fmt.Errorf("duplicate form field %q", field.ID)
		}
		seen[field.ID] = struct{}{}
		fieldTypes[field.ID] = field.Type
		steps[publicFieldStep(field)] = struct{}{}
		switch field.Type {
		case Text, Email, Textarea, Number, Checkbox, Document, Phone:
			if field.Type == Document {
				documentFields++
			}
			if len(field.Options) != 0 {
				return fmt.Errorf("field %q cannot have options", field.ID)
			}
		case Select, CascadingSelect:
			optionLimit := MaxOptions
			if field.Type == CascadingSelect {
				optionLimit = 1000
			}
			if len(field.Options) == 0 || len(field.Options) > optionLimit {
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
			if field.Type == CascadingSelect {
				if !idPattern.MatchString(field.DependsOn) || len(field.CascadeOptions) > MaxOptions {
					return fmt.Errorf("field %q has invalid cascade rules", field.ID)
				}
				for parent, options := range field.CascadeOptions {
					if parent == "" || len(parent) > MaxLabelLength || len(options) == 0 || len(options) > MaxOptions {
						return fmt.Errorf("field %q has invalid cascade options", field.ID)
					}
					for _, option := range options {
						if strings.TrimSpace(option) == "" || len(option) > MaxLabelLength {
							return fmt.Errorf("field %q has invalid cascade option", field.ID)
						}
					}
				}
			} else if field.DependsOn != "" || len(field.CascadeOptions) != 0 {
				return fmt.Errorf("field %q cannot have cascade rules", field.ID)
			}
		default:
			return fmt.Errorf("field %q has unsupported type", field.ID)
		}
		if field.Type == Email {
			hasEmail = true
		}
	}
	if f.MultiStep {
		for step := 1; step <= maximumFormStep(f); step++ {
			if _, ok := steps[step]; !ok {
				return fmt.Errorf("form steps must be contiguous")
			}
		}
	}
	for _, field := range f.Fields {
		if field.Type != CascadingSelect {
			continue
		}
		if fieldTypes[field.DependsOn] != Select && fieldTypes[field.DependsOn] != CascadingSelect {
			return fmt.Errorf("field %q has an invalid cascade parent", field.ID)
		}
		parentOptions := map[string]struct{}{}
		for _, option := range optionValues(f.Fields, field.DependsOn) {
			parentOptions[option] = struct{}{}
		}
		for parent := range field.CascadeOptions {
			if _, ok := parentOptions[parent]; !ok {
				return fmt.Errorf("field %q has an unknown cascade parent option", field.ID)
			}
		}
	}
	if f.Autoresponder && !hasEmail {
		return fmt.Errorf("autoresponder requires an email field")
	}
	if documentFields > 2 {
		return fmt.Errorf("forms may contain at most two document fields")
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
		if field.Type == Phone && value != "" && !validPhone(value) {
			errors[key] = "invalid-phone"
			continue
		}
		if field.Type == Email && value != "" {
			parsed, err := mail.ParseAddress(value)
			if err != nil || parsed.Address != value {
				errors[key] = "invalid-email"
				continue
			}
		}
		if field.Type == CascadingSelect && value != "" {
			parent := strings.TrimSpace(values[field.DependsOn])
			if !contains(field.CascadeOptions[parent], value) {
				errors[key] = "invalid-option"
				continue
			}
		}
		clean[key] = value
	}
	for _, field := range f.Fields {
		if field.Required && strings.TrimSpace(values[field.ID]) == "" {
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

func validPhone(value string) bool {
	digits, plus := 0, false
	for i, r := range value {
		if r >= '0' && r <= '9' {
			digits++
			continue
		}
		if r == '+' && i == 0 && !plus {
			plus = true
			continue
		}
		if r == ' ' || r == '(' || r == ')' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return digits >= 7 && digits <= 15
}

func optionValues(fields []Field, id string) []string {
	for _, field := range fields {
		if field.ID == id {
			return field.Options
		}
	}
	return nil
}

// StableFields returns a copy sorted by field ID for generated manifests.
func StableFields(fields []Field) []Field {
	result := append([]Field(nil), fields...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func NewSubmissionToken() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func publicInputType(kind FieldType) string {
	if kind == Email {
		return "email"
	}
	if kind == Number {
		return "number"
	}
	if kind == Document {
		return "file"
	}
	if kind == Phone {
		return "tel"
	}
	return "text"
}

func publicFormSubmitLabel(label string) string {
	if strings.TrimSpace(label) == "" {
		return "Send message"
	}
	return label
}

func publicMetaPixelEvent(f Form) string {
	if !f.ConversionTracking {
		return ""
	}
	if f.MetaPixelEventName == "" {
		return "Lead"
	}
	return f.MetaPixelEventName
}

func publicFormHasSteps(f Form) bool { return f.MultiStep && maximumFormStep(f) > 1 }
func maximumFormStep(f Form) int {
	maxStep := 1
	for _, field := range f.Fields {
		if field.Step > maxStep {
			maxStep = field.Step
		}
	}
	return maxStep
}
func publicFieldStep(field Field) int {
	if field.Step < 1 {
		return 1
	}
	return field.Step
}
func publicCascadeOptions(field Field) string {
	data, _ := json.Marshal(field.CascadeOptions)
	return string(data)
}
func phoneInputMode(kind FieldType) string {
	if kind == Phone {
		return "tel"
	}
	return "text"
}
func phoneAutocomplete(kind FieldType) string {
	if kind == Phone {
		return "tel"
	}
	return "off"
}

func safeFormAction(action, formID string) string {
	action = strings.TrimSpace(action)
	u, err := url.Parse(action)
	if err == nil && strings.HasPrefix(action, "/") && !strings.HasPrefix(action, "//") && !u.IsAbs() && u.Host == "" && !strings.ContainsAny(action, "\r\n\\") {
		return action
	}
	return "/forms/" + formID + "/submit"
}

func publicSubmissionToken(token string) string {
	if ValidateIdempotencyToken(token) == nil {
		return token
	}
	token, _ = NewSubmissionToken()
	return token
}

func ValidateIdempotencyToken(token string) error {
	if len(token) < 16 || len(token) > 128 || strings.TrimSpace(token) != token {
		return fmt.Errorf("invalid submission token")
	}
	for i := 0; i < len(token); i++ {
		if token[i] < 0x21 || token[i] > 0x7e {
			return fmt.Errorf("invalid submission token")
		}
	}
	return nil
}

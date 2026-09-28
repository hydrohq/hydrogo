// Package forms defines a bounded, provider-neutral form contract.
package forms

import (
	"crypto/rand"
	"encoding/hex"
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
	Text     FieldType = "text"
	Email    FieldType = "email"
	Textarea FieldType = "textarea"
	Number   FieldType = "number"
	Checkbox FieldType = "checkbox"
	Select   FieldType = "select"
	Document FieldType = "document"
)

type Field struct {
	ID       string    `json:"id"`
	Type     FieldType `json:"type"`
	Label    string    `json:"label"`
	Required bool      `json:"required"`
	Options  []string  `json:"options,omitempty"`
}

// AutoresponderBodyFor adds the configured response-time notice without
// changing the normal-hours response.
func(f Form)AutoresponderBodyFor(now time.Time)(string,error){
	body:=f.AutoresponderBody;if f.WorkingHours==nil{return body,nil};notice,after,err:=f.WorkingHours.AfterHoursNotice(now);if err!=nil{return "",err};if after{if body!=""{body+="\n\n"};body+=notice};return body,nil
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
	hasEmail := false
	documentFields := 0
	for _, field := range f.Fields {
		if !idPattern.MatchString(field.ID) || len(field.ID) > MaxIDLength || strings.TrimSpace(field.Label) == "" || len(field.Label) > MaxLabelLength {
			return fmt.Errorf("invalid form field %q", field.ID)
		}
		if _, ok := seen[field.ID]; ok {
			return fmt.Errorf("duplicate form field %q", field.ID)
		}
		seen[field.ID] = struct{}{}
		switch field.Type {
		case Text, Email, Textarea, Number, Checkbox, Document:
			if field.Type == Document {
				documentFields++
			}
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
		if field.Type == Email {
			hasEmail = true
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
		if field.Type == Email && value != "" {
			parsed, err := mail.ParseAddress(value)
			if err != nil || parsed.Address != value {
				errors[key] = "invalid-email"
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

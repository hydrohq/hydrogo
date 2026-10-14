// Package components contains the small, typed server-rendered UI contract
// shared by generated Client Editions. It deliberately has no browser runtime
// dependency and can be consumed from Templ or another templ.Component host.
// Markup lives in the .templ files next to this one.
package components

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// Variant is a design-system button variant.
type Variant string

const (
	Primary Variant = "primary"
	Quiet   Variant = "quiet"
	Danger  Variant = "danger"
)

// ButtonProps is intentionally typed so generated views do not assemble
// arbitrary markup or class strings.
type ButtonProps struct {
	Label    string
	Type     string
	Variant  Variant
	Class    string
	Disabled bool
	HTMX     HTMX
}

type LinkProps struct{ Href, Label, Class string }

// Link renders a same-origin link. Hrefs with a query, fragment, scheme or
// host are rejected at render time; use ButtonLink for query strings.
func Link(p LinkProps) templ.Component {
	if !validLocalPath(p.Href) {
		return failing(errors.New("link must use a local path"))
	}
	return linkTemplate(p)
}

func validLocalPath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") && !strings.ContainsAny(value, "?#\r\n\x00")
}

// InputProps describes a safe, accessible form input.
type InputProps struct {
	ID           string
	Name         string
	Label        string
	Type         string
	Value        string
	Placeholder  string
	Required     bool
	ReadOnly     bool
	Checked      bool
	MinLength    int
	MaxLength    int
	HasMinLength bool
	HasMaxLength bool
	HTMX         HTMX
}

type HiddenInputProps struct{ Name, Value string }

// HiddenInput renders a hidden form value; a name is required.
func HiddenInput(p HiddenInputProps) templ.Component {
	if strings.TrimSpace(p.Name) == "" {
		return failing(errors.New("hidden input name is required"))
	}
	return hiddenInputTemplate(p)
}

// CardProps wraps generated content in a stable design-system surface.
type CardProps struct {
	Title string
	Class string
	Body  templ.Component
}

// FormProps provides the mutation boundary used by generated forms.
type FormProps struct {
	Action        string
	Method        string
	Class         string
	CSRF          string
	ConfirmAction string
	Body          templ.Component
}

type FieldValidationError struct {
	FieldID string
	Label   string
	Message string
}

type ValidationSummaryProps struct {
	Title   string
	Message string
	Fields  []FieldValidationError
}

// ValidationSummary renders safe, keyboard-navigable links to invalid fields.
func ValidationSummary(p ValidationSummaryProps) templ.Component {
	if p.Title == "" || p.Message == "" {
		return failing(errors.New("validation summary title and message are required"))
	}
	for _, field := range p.Fields {
		if strings.TrimSpace(field.FieldID) == "" || strings.TrimSpace(field.Label) == "" || strings.TrimSpace(field.Message) == "" {
			return failing(errors.New("validation summary field id, label, and message are required"))
		}
	}
	return validationSummaryTemplate(p)
}

// HTMX is the allowlisted server interaction contract. Empty fields emit no
// attribute. Unsupported HTMX attributes cannot be represented by this type.
type HTMX struct {
	Get     string
	Post    string
	Target  string
	Swap    string
	Trigger string
	Confirm string
}

func buttonType(typ string) string {
	if typ == "" {
		return "button"
	}
	return typ
}

func buttonClass(p ButtonProps) string {
	variant := p.Variant
	if variant == "" {
		variant = Primary
	}
	className := "button button-" + string(variant)
	if extra := strings.TrimSpace(p.Class); extra != "" {
		className += " " + extra
	}
	return className
}

func cardClass(class string) string {
	if extra := strings.TrimSpace(class); extra != "" {
		return "card " + extra
	}
	return "card"
}

func formMethod(method string) string {
	if method == "" {
		return "post"
	}
	return method
}

// failing is a component that renders nothing and reports err, so invalid
// props surface as a render error exactly where the component is used.
func failing(err error) templ.Component {
	return templ.ComponentFunc(func(context.Context, io.Writer) error { return err })
}

// Package components contains the small, typed server-rendered UI contract
// shared by generated Client Editions. It deliberately has no browser runtime
// dependency and can be consumed from Templ or another templ.Component host.
package components

import (
	"context"
	"errors"
	"fmt"
	"html/template"
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

func Link(p LinkProps) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		if !validLocalPath(p.Href) {
			return errors.New("link must use a local path")
		}
		class := ""
		if strings.TrimSpace(p.Class) != "" {
			class = ` class="` + esc(p.Class) + `"`
		}
		_, err := io.WriteString(w, `<a href="`+esc(p.Href)+`"`+class+`>`+esc(p.Label)+`</a>`)
		return err
	})
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

func HiddenInput(p HiddenInputProps) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		if strings.TrimSpace(p.Name) == "" {
			return errors.New("hidden input name is required")
		}
		_, err := io.WriteString(w, `<input type="hidden" name="`+esc(p.Name)+`" value="`+esc(p.Value)+`">`)
		return err
	})
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
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		if p.Title == "" || p.Message == "" {
			return errors.New("validation summary title and message are required")
		}
		if _, err := io.WriteString(w, `<section class="studio-alert" role="alert" aria-labelledby="generated-validation-title"><h2 id="generated-validation-title">`+esc(p.Title)+`</h2><p>`+esc(p.Message)+`</p>`); err != nil {
			return err
		}
		if len(p.Fields) > 0 {
			if _, err := io.WriteString(w, `<ul>`); err != nil {
				return err
			}
			for _, field := range p.Fields {
				if strings.TrimSpace(field.FieldID) == "" || strings.TrimSpace(field.Label) == "" || strings.TrimSpace(field.Message) == "" {
					return errors.New("validation summary field id, label, and message are required")
				}
				if _, err := io.WriteString(w, `<li><a href="#`+esc(field.FieldID)+`">`+esc(field.Label)+`: `+esc(field.Message)+`</a></li>`); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, `</ul>`); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</section>`)
		return err
	})
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

func Button(p ButtonProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		typ := p.Type
		if typ == "" {
			typ = "button"
		}
		variant := p.Variant
		if variant == "" {
			variant = Primary
		}
		className := "button button-" + string(variant)
		if strings.TrimSpace(p.Class) != "" {
			className += " " + strings.TrimSpace(p.Class)
		}
		attrs := htmxAttrs(p.HTMX)
		if p.Disabled {
			attrs += ` disabled="disabled"`
		}
		_, err := io.WriteString(w, `<button type="`+esc(typ)+`" class="`+esc(className)+`"`+attrs+`>`+esc(p.Label)+`</button>`)
		return err
	})
}

func Card(p CardProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		className := "card"
		if strings.TrimSpace(p.Class) != "" {
			className += " " + strings.TrimSpace(p.Class)
		}
		if _, err := io.WriteString(w, `<section class="`+esc(className)+`"><h2>`+esc(p.Title)+`</h2>`); err != nil {
			return err
		}
		if p.Body != nil {
			if err := p.Body.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</section>`)
		return err
	})
}

func Form(p FormProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		method := p.Method
		if method == "" {
			method = "post"
		}
		classAttr := ""
		if strings.TrimSpace(p.Class) != "" {
			classAttr = ` class="` + esc(strings.TrimSpace(p.Class)) + `"`
		}
		confirmAttr := ""
		if strings.TrimSpace(p.ConfirmAction) != "" {
			confirmAttr = ` data-confirm-action="` + esc(p.ConfirmAction) + `"`
		}
		if _, err := io.WriteString(w, `<form action="`+esc(p.Action)+`" method="`+esc(method)+`"`+classAttr+confirmAttr+`>`); err != nil {
			return err
		}
		if p.CSRF != "" {
			if _, err := io.WriteString(w, `<input type="hidden" name="_csrf" value="`+esc(p.CSRF)+`">`); err != nil {
				return err
			}
		}
		if p.Body != nil {
			if err := p.Body.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</form>`)
		return err
	})
}

func htmxAttrs(h HTMX) string {
	var b strings.Builder
	write := func(name, value string) {
		if strings.TrimSpace(value) != "" {
			fmt.Fprintf(&b, ` %s="%s"`, name, esc(value))
		}
	}
	write("hx-get", h.Get)
	write("hx-post", h.Post)
	write("hx-target", h.Target)
	write("hx-swap", h.Swap)
	write("hx-trigger", h.Trigger)
	write("hx-confirm", h.Confirm)
	return b.String()
}

func esc(value string) string { return template.HTMLEscapeString(value) }

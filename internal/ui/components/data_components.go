package components

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

// SelectProps describes a typed select control. Options are escaped by the
// component; generated views never need to assemble option markup.
type SelectProps struct {
	ID       string
	Name     string
	Label    string
	Value    string
	Options  []string
	Items    []SelectOption
	Selected []string
	Multiple bool
	Required bool
	HTMX     HTMX
}

type SelectOption struct{ Value, Label string }

func Select(p SelectProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		attrs := ` id="` + esc(p.ID) + `" name="` + esc(p.Name) + `"` + htmxAttrs(p.HTMX)
		if p.Required {
			attrs += ` required="required"`
		}
		if p.Multiple {
			attrs += ` multiple="multiple"`
		}
		if _, err := io.WriteString(w, `<label for="`+esc(p.ID)+`">`+esc(p.Label)+`</label><select`+attrs+`>`); err != nil {
			return err
		}
		items := p.Items
		if len(items) == 0 {
			items = make([]SelectOption, len(p.Options))
			for index, option := range p.Options {
				items[index] = SelectOption{Value: option, Label: option}
			}
		}
		for _, option := range items {
			selected := ""
			if option.Value == p.Value || slices.Contains(p.Selected, option.Value) {
				selected = ` selected="selected"`
			}
			if _, err := io.WriteString(w, `<option value="`+esc(option.Value)+`"`+selected+`>`+esc(option.Label)+`</option>`); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</select>`)
		return err
	})
}

// TextareaProps describes a typed multiline control.
type TextareaProps struct {
	ID           string
	Name         string
	Label        string
	Value        string
	Required     bool
	Rows         int
	MinLength    int
	MaxLength    int
	HasMinLength bool
	HasMaxLength bool
	HTMX         HTMX
}

func Textarea(p TextareaProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		rows := p.Rows
		if rows < 1 {
			rows = 5
		}
		attrs := ` id="` + esc(p.ID) + `" name="` + esc(p.Name) + `" rows="` + strconv.Itoa(rows) + `"` + htmxAttrs(p.HTMX)
		if p.Required {
			attrs += ` required="required"`
		}
		if p.HasMinLength && p.MinLength >= 0 {
			attrs += ` minlength="` + strconv.Itoa(p.MinLength) + `"`
		}
		if p.HasMaxLength && p.MaxLength >= 0 {
			attrs += ` maxlength="` + strconv.Itoa(p.MaxLength) + `"`
		}
		_, err := io.WriteString(w, `<label for="`+esc(p.ID)+`">`+esc(p.Label)+`</label><textarea`+attrs+`>`+esc(p.Value)+`</textarea>`)
		return err
	})
}

// TableProps is the safe data boundary used by generated list views.
type TableProps struct {
	Caption      string
	Columns      []string
	Rows         [][]string
	RowActions   [][]TableAction
	EmptyMessage string
}

// TableAction is a CSRF-protected POST form rendered in its row.
type TableAction struct {
	Label, Action, CSRF string
	Variant             Variant
}

func Table(p TableProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		empty := p.EmptyMessage
		if strings.TrimSpace(empty) == "" {
			empty = "Kayıt bulunamadı."
		}
		if _, err := io.WriteString(w, `<table><caption class="visually-hidden">`+esc(p.Caption)+`</caption><thead><tr>`); err != nil {
			return err
		}
		for _, column := range p.Columns {
			if _, err := io.WriteString(w, `<th scope="col">`+esc(column)+`</th>`); err != nil {
				return err
			}
		}
		hasActions := len(p.RowActions) > 0
		if hasActions {
			if _, err := io.WriteString(w, `<th scope="col">İşlem</th>`); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, `</tr></thead><tbody>`); err != nil {
			return err
		}
		if len(p.Rows) == 0 {
			columns := len(p.Columns)
			if hasActions {
				columns++
			}
			_, err := io.WriteString(w, `<tr><td colspan="`+strconv.Itoa(max(1, columns))+`" role="status">`+esc(empty)+`</td></tr>`)
			if err != nil {
				return err
			}
		} else {
			for index, row := range p.Rows {
				if _, err := io.WriteString(w, `<tr>`); err != nil {
					return err
				}
				for i := range p.Columns {
					value := ""
					if i < len(row) {
						value = row[i]
					}
					if _, err := io.WriteString(w, `<td>`+esc(value)+`</td>`); err != nil {
						return err
					}
				}
				if hasActions {
					if _, err := io.WriteString(w, `<td>`); err != nil {
						return err
					}
					if index < len(p.RowActions) {
						for _, action := range p.RowActions[index] {
							if !validTableAction(action.Action) {
								return errors.New("table action must be a local POST path")
							}
							form := Form(FormProps{Action: action.Action, Method: "post", CSRF: action.CSRF, Body: Button(ButtonProps{Label: action.Label, Type: "submit", Variant: action.Variant})})
							if err := form.Render(ctx, w); err != nil {
								return err
							}
						}
					}
					if _, err := io.WriteString(w, `</td>`); err != nil {
						return err
					}
				}
				if _, err := io.WriteString(w, `</tr>`); err != nil {
					return err
				}
			}
		}
		_, err := io.WriteString(w, `</tbody></table>`)
		return err
	})
}

func validTableAction(action string) bool {
	return validLocalPath(action)
}

// PaginationProps keeps page navigation bounded and exposes only the HTMX
// attributes supported by the shared component contract.
type PaginationProps struct {
	Page    int
	Pages   int
	BaseURL string
	Target  string
	Swap    string
}

func Pagination(p PaginationProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if p.Pages <= 1 {
			return nil
		}
		page := p.Page
		if page < 1 {
			page = 1
		}
		if page > p.Pages {
			page = p.Pages
		}
		if _, err := io.WriteString(w, `<nav aria-label="Sayfalama" class="pagination">`); err != nil {
			return err
		}
		for current := 1; current <= p.Pages; current++ {
			attrs := fmt.Sprintf(` href="%s?page=%d"`, esc(p.BaseURL), current)
			attrs += htmxAttrs(HTMX{Get: fmt.Sprintf("%s?page=%d", p.BaseURL, current), Target: p.Target, Swap: p.Swap})
			if current == page {
				attrs += ` aria-current="page"`
			}
			if _, err := io.WriteString(w, `<a`+attrs+`>`+strconv.Itoa(current)+`</a>`); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</nav>`)
		return err
	})
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

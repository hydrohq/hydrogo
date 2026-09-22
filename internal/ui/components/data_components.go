package components

import (
	"context"
	"fmt"
	"io"
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
	Required bool
	HTMX     HTMX
}

func Select(p SelectProps) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		attrs := ` id="` + esc(p.ID) + `" name="` + esc(p.Name) + `"` + htmxAttrs(p.HTMX)
		if p.Required {
			attrs += ` required="required"`
		}
		if _, err := io.WriteString(w, `<label for="`+esc(p.ID)+`">`+esc(p.Label)+`</label><select`+attrs+`>`); err != nil {
			return err
		}
		for _, option := range p.Options {
			selected := ""
			if option == p.Value {
				selected = ` selected="selected"`
			}
			if _, err := io.WriteString(w, `<option value="`+esc(option)+`"`+selected+`>`+esc(option)+`</option>`); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</select>`)
		return err
	})
}

// TextareaProps describes a typed multiline control.
type TextareaProps struct {
	ID       string
	Name     string
	Label    string
	Value    string
	Required bool
	Rows     int
	HTMX     HTMX
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
		_, err := io.WriteString(w, `<label for="`+esc(p.ID)+`">`+esc(p.Label)+`</label><textarea`+attrs+`>`+esc(p.Value)+`</textarea>`)
		return err
	})
}

// TableProps is the safe data boundary used by generated list views.
type TableProps struct {
	Caption      string
	Columns      []string
	Rows         [][]string
	EmptyMessage string
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
		if _, err := io.WriteString(w, `</tr></thead><tbody>`); err != nil {
			return err
		}
		if len(p.Rows) == 0 {
			_, err := io.WriteString(w, `<tr><td colspan="`+strconv.Itoa(max(1, len(p.Columns)))+`" role="status">`+esc(empty)+`</td></tr>`)
			if err != nil {
				return err
			}
		} else {
			for _, row := range p.Rows {
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
				if _, err := io.WriteString(w, `</tr>`); err != nil {
					return err
				}
			}
		}
		_, err := io.WriteString(w, `</tbody></table>`)
		return err
	})
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

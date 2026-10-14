package components

import (
	"errors"
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

// ItemListEditor renders an accessible string-array editor. The shared Admin
// shell hydrates the rows and synchronizes them to the hidden newline value.
type ItemListEditorProps struct {
	ID, Name, Label, Value                                        string
	AddLabel, AddAriaLabel                                        string
	ItemPlaceholder, ItemAriaLabelPattern, RemoveAriaLabelPattern string
}

func ItemListEditor(p ItemListEditorProps) templ.Component {
	if strings.TrimSpace(p.ID) == "" {
		return failing(errors.New("item list id is required"))
	}
	if strings.TrimSpace(p.Name) == "" {
		p.Name = p.ID
	}
	defaults := []struct {
		target   *string
		fallback string
	}{
		{&p.AddLabel, "Madde ekle"},
		{&p.AddAriaLabel, p.Label + ": madde ekle"},
		{&p.ItemPlaceholder, "Önemli bir madde girin…"},
		{&p.ItemAriaLabelPattern, "{field} {index}"},
		{&p.RemoveAriaLabelPattern, "Maddeyi sil"},
	}
	for _, d := range defaults {
		if *d.target == "" {
			*d.target = d.fallback
		}
	}
	return itemListEditorTemplate(p)
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

// Table renders an escaped data table. Row actions must post to local paths;
// anything else fails the render.
func Table(p TableProps) templ.Component {
	if strings.TrimSpace(p.EmptyMessage) == "" {
		p.EmptyMessage = "Kayıt bulunamadı."
	}
	for index, actions := range p.RowActions {
		if index >= len(p.Rows) {
			break
		}
		for _, action := range actions {
			if !validTableAction(action.Action) {
				return failing(errors.New("table action must be a local POST path"))
			}
		}
	}
	return tableTemplate(p)
}

func validTableAction(action string) bool {
	return validLocalPath(action)
}

// tableCell returns the row value for column i, or "" for short rows.
func tableCell(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

// tableRowActions returns the actions for row index, or nil.
func tableRowActions(p TableProps, index int) []TableAction {
	if index < len(p.RowActions) {
		return p.RowActions[index]
	}
	return nil
}

// tableEmptyColspan spans every column, the action column included.
func tableEmptyColspan(p TableProps) string {
	columns := len(p.Columns)
	if len(p.RowActions) > 0 {
		columns++
	}
	return strconv.Itoa(max(1, columns))
}

func tableActionForm(action TableAction) templ.Component {
	return Form(FormProps{Action: action.Action, Method: "post", CSRF: action.CSRF, Body: Button(ButtonProps{Label: action.Label, Type: "submit", Variant: action.Variant})})
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

// Pagination renders one link per page; a single page renders nothing.
func Pagination(p PaginationProps) templ.Component {
	if p.Pages <= 1 {
		return templ.NopComponent
	}
	p.Page = min(max(p.Page, 1), p.Pages)
	return paginationTemplate(p)
}

// paginationPages lists the one-based page numbers.
func paginationPages(pages int) []int {
	numbers := make([]int, pages)
	for i := range numbers {
		numbers[i] = i + 1
	}
	return numbers
}

func paginationHref(p PaginationProps, page int) string {
	return p.BaseURL + "?page=" + strconv.Itoa(page)
}

func paginationHTMX(p PaginationProps, page int) templ.Attributes {
	return HTMX{Get: paginationHref(p, page), Target: p.Target, Swap: p.Swap}.Attributes()
}

func selectItems(p SelectProps) []SelectOption {
	if len(p.Items) > 0 {
		return p.Items
	}
	items := make([]SelectOption, len(p.Options))
	for index, option := range p.Options {
		items[index] = SelectOption{Value: option, Label: option}
	}
	return items
}

func containsValue(values []string, wanted string) bool { return slices.Contains(values, wanted) }

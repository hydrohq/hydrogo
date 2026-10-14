package adminstudio

import (
	"context"
	"io"
	"net/http"

	"github.com/a-h/templ"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

// View models rendered by generated_views.templ.

type generatedLocaleNav struct {
	Enabled bool
	Label   string
	Items   []generatedLocaleNavItem
}

type generatedLocaleNavItem struct {
	Label   string
	Current bool
	Switch  ui.ActionFormProps
}

type generatedListPage struct {
	Description string
	NewRecord   *ui.LinkProps
	DraftsForm  *ui.ActionFormProps
	DraftsLink  *ui.LinkProps
	LocaleNav   generatedLocaleNav
	Controls    *generatedListControls
	List        templ.Component
	Pager       []ui.ActionFormProps
}

type generatedListControls struct {
	Action  string
	Hidden  []ui.HiddenInputProps
	Search  *ui.InputProps
	Filters []generatedListFilter
	Apply   ui.ButtonProps
}

// generatedListFilter is either a select (fields with options) or a text input.
type generatedListFilter struct {
	Select *ui.SelectProps
	Input  *ui.InputProps
}

type generatedDetailRow struct{ Label, Value string }

type generatedFormPage struct {
	LocaleNav       generatedLocaleNav
	ConflictMessage string
	Form            ui.FormProps
	Delete          *ui.ActionFormProps
	Workflow        *generatedActionGroup
	Revisions       *generatedRevisionHistory
	CopyLocale      *generatedActionGroup
	Discard         *ui.ActionFormProps
}

// generatedActionGroup is a titled card of action forms.
type generatedActionGroup struct {
	Title string
	Forms []ui.ActionFormProps
}

type generatedRevisionHistory struct {
	Title string
	Empty string
	Items []generatedRevisionItem
}

type generatedRevisionItem struct {
	Open    ui.ActionFormProps
	Summary string
	Restore *ui.ActionFormProps
}

type generatedDiffRow struct{ Label, Before, After string }

type generatedDraftsPage struct {
	LocaleNav generatedLocaleNav
	Title     string
	List      templ.Component
	Pager     []ui.ActionFormProps
}

// renderListComponent adapts RenderList to a templ component.
func renderListComponent(data ListData) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		return RenderList(w, data)
	})
}

// getForm is a GET form that carries hidden query fields and a quiet button.
func getForm(action string, hidden []ui.HiddenInputProps, label string) ui.ActionFormProps {
	return ui.ActionFormProps{Action: action, Method: http.MethodGet, Hidden: hidden, Button: ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Quiet}}
}

// postForm is a CSRF-protected POST form with hidden fields and one button.
func postForm(action, csrf string, hidden []ui.HiddenInputProps, label string, variant ui.Variant) ui.ActionFormProps {
	return ui.ActionFormProps{Action: action, Method: "post", CSRF: csrf, Hidden: hidden, Button: ui.ButtonProps{Type: "submit", Label: label, Variant: variant}}
}

// localeHidden is the locale query field when the collection is localized.
func localeHidden(localized bool, locale string) []ui.HiddenInputProps {
	if !localized {
		return nil
	}
	return []ui.HiddenInputProps{{Name: "locale", Value: locale}}
}

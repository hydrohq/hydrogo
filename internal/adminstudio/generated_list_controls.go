package adminstudio

import (
	"context"
	"io"
	"net/url"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func generatedListControls(w io.Writer, collection Collection, query ListQuery, locale, uiLocale string) error {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if !hasListControls(collection) {
			return nil
		}
		if _, err := io.WriteString(out, `<form class="admin-list-toolbar studio-form" method="get" action="/admin/resources/`+url.PathEscape(collection.ID)+`">`); err != nil {
			return err
		}
		if query.Sort != "" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "sort", Value: query.Sort}).Render(ctx, out); err != nil {
				return err
			}
		}
		if collection.Localization && locale != "" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale", Value: locale}).Render(ctx, out); err != nil {
				return err
			}
		}
		searchFields := 0
		for _, field := range collection.Fields {
			if field.Searchable && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
				searchFields++
			}
		}
		if searchFields > 0 {
			if err := ui.Input(ui.InputProps{ID: "resource-search", Name: "search", Label: i18n.T(uiLocale, "admin.generated.list.search"), Type: "search", Value: query.Search}).Render(ctx, out); err != nil {
				return err
			}
		}
		for _, field := range collection.Fields {
			if !field.Filterable {
				continue
			}
			options := append([]string(nil), field.Options...)
			if len(options) > 0 {
				allLabel := i18n.T(uiLocale, "admin.generated.list.all")
				items := make([]ui.SelectOption, 0, len(options)+1)
				items = append(items, ui.SelectOption{Value: "", Label: allLabel})
				for _, option := range options {
					items = append(items, ui.SelectOption{Value: option, Label: option})
				}
				if err := ui.Select(ui.SelectProps{ID: "filter-" + field.ID, Name: "filter[" + field.ID + "]", Label: field.Label, Value: query.Filters[field.ID], Items: items}).Render(ctx, out); err != nil {
					return err
				}
			} else if err := ui.Input(ui.InputProps{ID: "filter-" + field.ID, Name: "filter[" + field.ID + "]", Label: field.Label, Value: query.Filters[field.ID]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `<div class="studio-actions">`); err != nil {
			return err
		}
		if err := ui.Button(ui.ButtonProps{Type: "submit", Label: i18n.T(uiLocale, "admin.generated.list.apply"), Variant: ui.Primary}).Render(ctx, out); err != nil {
			return err
		}
		if _, err := io.WriteString(out, `</div>`); err != nil {
			return err
		}
		_, err := io.WriteString(out, `</form>`)
		return err
	}).Render(context.Background(), w)
}

// hasListControls reports whether a collection has any searchable or
// filterable field; without one the toolbar would be a lone Apply button.
func hasListControls(collection Collection) bool {
	for _, field := range collection.Fields {
		if field.Filterable || field.Searchable && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
			return true
		}
	}
	return false
}

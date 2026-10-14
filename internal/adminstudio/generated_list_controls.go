package adminstudio

import (
	"net/url"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

// generatedListControlsFor builds the search and filter toolbar of a
// generated list, or nil when the collection has nothing to search or filter.
func generatedListControlsFor(collection Collection, query ListQuery, locale, uiLocale string) *generatedListControls {
	if !hasListControls(collection) {
		return nil
	}
	controls := &generatedListControls{
		Action: "/admin/resources/" + url.PathEscape(collection.ID),
		Apply:  ui.ButtonProps{Type: "submit", Label: i18n.T(uiLocale, "admin.generated.list.apply"), Variant: ui.Primary},
	}
	if query.Sort != "" {
		controls.Hidden = append(controls.Hidden, ui.HiddenInputProps{Name: "sort", Value: query.Sort})
	}
	if collection.Localization && locale != "" {
		controls.Hidden = append(controls.Hidden, ui.HiddenInputProps{Name: "locale", Value: locale})
	}
	for _, field := range collection.Fields {
		if field.Searchable && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
			controls.Search = &ui.InputProps{ID: "resource-search", Name: "search", Label: i18n.T(uiLocale, "admin.generated.list.search"), Type: "search", Value: query.Search}
			break
		}
	}
	for _, field := range collection.Fields {
		if !field.Filterable {
			continue
		}
		id, name := "filter-"+field.ID, "filter["+field.ID+"]"
		if len(field.Options) == 0 {
			controls.Filters = append(controls.Filters, generatedListFilter{Input: &ui.InputProps{ID: id, Name: name, Label: field.Label, Value: query.Filters[field.ID]}})
			continue
		}
		items := make([]ui.SelectOption, 0, len(field.Options)+1)
		items = append(items, ui.SelectOption{Value: "", Label: i18n.T(uiLocale, "admin.generated.list.all")})
		for _, option := range field.Options {
			items = append(items, ui.SelectOption{Value: option, Label: option})
		}
		controls.Filters = append(controls.Filters, generatedListFilter{Select: &ui.SelectProps{ID: id, Name: name, Label: field.Label, Value: query.Filters[field.ID], Items: items}})
	}
	return controls
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

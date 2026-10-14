package adminstudio

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func generatedFormFields(collection Collection, values map[string]string, relationOptions map[string][]ui.SelectOption, id string, editing bool, locale, defaultLocale string, localizedEdit, draftEdit, saveAllowed bool, interfaceLocale string, blockEditor BlockEditorConfig) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if editing && collection.Versioned && !localizedEdit && !draftEdit {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "version", Value: values["version"]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if editing && localizedEdit && !draftEdit {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "locale_version", Value: values["locale_version"]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if draftEdit {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "draft_version", Value: values["draft_version"]}).Render(ctx, out); err != nil {
				return err
			}
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "published_version", Value: values["published_version"]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if !editing && collection.Type != "global" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "id", Value: id}).Render(ctx, out); err != nil {
				return err
			}
		}
		for _, field := range collection.Fields {
			if collection.Workflow != nil && field.ID == collection.Workflow.StateField {
				value := values[field.ID]
				if value == "" && !editing {
					value = string(collection.Workflow.States[0])
				}
				if err := ui.Input(ui.InputProps{ID: field.ID, Name: field.ID, Label: field.Label, Type: "text", Value: value, ReadOnly: true}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.ReadOnly {
				continue
			}
			label := field.Label
			localizedOnly := collection.Localization && locale != defaultLocale && !field.Localized
			if field.Localized {
				label += " · " + locale
			}
			props := ui.InputProps{ID: field.ID, Name: field.ID, Label: label, Type: "text", Value: values[field.ID], Required: field.Required && (!collection.Localization || locale == defaultLocale), ReadOnly: localizedOnly}
			if localizedOnly {
				if err := ui.Input(props).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldArray {
				if err := ui.ItemListEditor(ui.ItemListEditorProps{
					ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID],
					AddLabel:               i18n.T(interfaceLocale, "admin.generated.widget.array.add"),
					AddAriaLabel:           fmt.Sprintf(i18n.T(interfaceLocale, "admin.generated.widget.array.add_aria"), field.Label),
					ItemPlaceholder:        i18n.T(interfaceLocale, "admin.generated.widget.array.item_placeholder"),
					ItemAriaLabelPattern:   i18n.T(interfaceLocale, "admin.generated.widget.array.item_aria"),
					RemoveAriaLabelPattern: i18n.T(interfaceLocale, "admin.generated.widget.array.remove_aria"),
				}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldBlocks || field.Widget == "block-editor" {
				if err := ui.BlockEditor(ui.BlockEditorProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], EditorScript: blockEditor.EditorScript, PreviewURL: blockEditor.PreviewURL, PreviewPending: i18n.T(interfaceLocale, "admin.generated.widget.blocks.preview_pending")}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldMedia || field.Widget == "media" {
				if err := ui.MediaPicker(ui.MediaPickerProps{
					ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID],
					SelectedImageAlt:  i18n.T(interfaceLocale, "admin.generated.widget.media.selected_image_alt"),
					EmptySelection:    i18n.T(interfaceLocale, "admin.generated.widget.media.empty"),
					ChooseFromLibrary: i18n.T(interfaceLocale, "admin.generated.widget.media.choose"),
					ClearSelection:    i18n.T(interfaceLocale, "admin.generated.widget.media.clear"),
					OpenLibrary:       i18n.T(interfaceLocale, "admin.generated.widget.media.open_library"),
				}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			if field.Kind == FieldRelation && field.Relation != nil {
				items := append([]ui.SelectOption(nil), relationOptions[field.ID]...)
				selected := relationIDsFromValue(values[field.ID], field.Relation.Cardinality)
				if field.Relation.Cardinality == "one" {
					items = append([]ui.SelectOption{{Value: "", Label: i18n.T(interfaceLocale, "admin.generated.form.choose_relation")}}, items...)
				}
				if err := ui.Select(ui.SelectProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], Items: items, Selected: selected, Multiple: field.Relation.Cardinality == "many", Required: props.Required}).Render(ctx, out); err != nil {
					return err
				}
				continue
			}
			switch field.Kind {
			case FieldNumber:
				props.Type = "number"
			case FieldBoolean:
				props.Type = "checkbox"
				props.Checked = values[field.ID] == "true"
			case FieldDate:
				props.Type = "date"
			}
			switch field.Widget {
			case "date":
				props.Type = "date"
			case "datetime", "datetime-local":
				props.Type = "datetime-local"
			case "checkbox":
				if field.Kind == FieldBoolean {
					props.Type = "checkbox"
					props.Checked = values[field.ID] == "true"
				}
			case "currency":
				if field.Kind == FieldNumber {
					props.Type = "number"
				}
			}
			var component templ.Component = ui.Input(props)
			if len(field.Options) > 0 {
				component = ui.Select(ui.SelectProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], Options: field.Options, Required: props.Required})
			} else if field.Widget == "richtext" || field.Kind == FieldRichText {
				component = ui.RichTextEditor(ui.RichTextEditorProps{
					ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID],
					ToolbarLabel:     i18n.T(interfaceLocale, "admin.generated.widget.richtext.toolbar"),
					BoldLabel:        i18n.T(interfaceLocale, "admin.generated.widget.richtext.bold"),
					ItalicLabel:      i18n.T(interfaceLocale, "admin.generated.widget.richtext.italic"),
					UnderlineLabel:   i18n.T(interfaceLocale, "admin.generated.widget.richtext.underline"),
					LinkLabel:        i18n.T(interfaceLocale, "admin.generated.widget.richtext.link"),
					ImageLabel:       i18n.T(interfaceLocale, "admin.generated.widget.richtext.image"),
					BlockQuoteLabel:  i18n.T(interfaceLocale, "admin.generated.widget.richtext.quote"),
					OrderedListLabel: i18n.T(interfaceLocale, "admin.generated.widget.richtext.ordered_list"),
					BulletListLabel:  i18n.T(interfaceLocale, "admin.generated.widget.richtext.bullet_list"),
				})
			} else if field.Widget == "textarea" {
				component = ui.Textarea(ui.TextareaProps{ID: field.ID, Name: field.ID, Label: label, Value: values[field.ID], Required: props.Required, Rows: 8})
			}
			if err := component.Render(ctx, out); err != nil {
				return err
			}
		}
		label := i18n.T(interfaceLocale, "admin.generated.form.save")
		if draftEdit {
			label = i18n.T(interfaceLocale, "admin.generated.form.save_draft")
		}
		return ui.Button(ui.ButtonProps{Type: "submit", Label: label, Variant: ui.Primary, Disabled: !saveAllowed}).Render(ctx, out)
	})
}

func relationIDsFromValue(value, cardinality string) []string {
	if value == "" || cardinality != "many" {
		return nil
	}
	return strings.Split(value, ",")
}

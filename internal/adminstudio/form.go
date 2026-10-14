package adminstudio

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/a-h/templ"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
)

var ErrInvalidForm = errors.New("invalid admin form")

type FormData struct {
	Collection      Collection
	Action          string
	CSRFToken       string
	Values          map[string]string
	RelationOptions map[string][]RelationOption
	Locales         []string
	CurrentLocale   string
	SubmitLabel     string
	HideSubmit      bool
	Nonce           string
	UILocale        string
}

func (d FormData) T(key string) string {
	return i18n.T(d.UILocale, key)
}

type RelationOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RenderCreateForm renders metadata-driven controls only. The surrounding
// module handler remains responsible for authorization, validation, service
// invocation, audit, and response navigation.
func RenderCreateForm(w io.Writer, data FormData) error {
	if _, err := NewRegistry([]Collection{data.Collection}); err != nil || data.Action == "" || data.Action[0] != '/' || (len(data.Action) > 1 && data.Action[1] == '/') || data.CSRFToken == "" {
		return ErrInvalidForm
	}
	if data.Values == nil {
		data.Values = map[string]string{}
	}
	return createFormView(data, newCreateFormViewModel(data)).Render(context.Background(), w)
}

// createFormViewModel holds the values createFormView derives from FormData.
type createFormViewModel struct {
	SubmitLabel    string
	RichTextLabel  string
	ChooseRelation string
	LocaleLabel    string
	Locales        []string
	Relations      []createFormRelation
}

type createFormRelation struct {
	Field   Field
	Options []RelationOption
}

func newCreateFormViewModel(data FormData) createFormViewModel {
	model := createFormViewModel{SubmitLabel: data.SubmitLabel, RichTextLabel: "Zengin metin alanı"}
	if model.SubmitLabel == "" {
		model.SubmitLabel = "Create"
	}
	for _, field := range data.Collection.Fields {
		if field.Widget == "richtext" || field.ID == "body" || field.ID == "description" {
			// Quill turns the editor into a contenteditable textbox at runtime;
			// the server-rendered shell carries the first rich text label.
			model.RichTextLabel = field.Label
			break
		}
	}
	model.ChooseRelation = i18n.T(data.UILocale, "admin.generated.form.choose_relation")
	if model.ChooseRelation == "" || model.ChooseRelation == "admin.generated.form.choose_relation" {
		model.ChooseRelation = "Seçin"
	}
	for _, field := range data.Collection.Fields {
		if field.Kind == FieldRelation && !field.ReadOnly {
			model.Relations = append(model.Relations, createFormRelation{Field: field, Options: data.RelationOptions[field.ID]})
		}
	}
	model.LocaleLabel = i18n.T(data.UILocale, "admin.generated.locale.navigation")
	if model.LocaleLabel == "" || model.LocaleLabel == "admin.generated.locale.navigation" {
		model.LocaleLabel = "İçerik dili"
	}
	for _, item := range data.Locales {
		item = strings.TrimSpace(item)
		if item != "" && len(item) <= 16 {
			model.Locales = append(model.Locales, item)
		}
	}
	return model
}

func isCreateFormRichText(field Field) bool {
	return field.Widget == "richtext" || field.ID == "body" || field.ID == "description"
}

// createFormLabel is a field label with the required-field marker.
func createFormLabel(field Field) string {
	if field.Required {
		return field.Label + " *"
	}
	return field.Label
}

func createFormChecked(value string) bool {
	return value == "true" || value == "on" || value == "1"
}

func createFormInputType(field Field) string {
	switch {
	case field.Widget == "datetime-local":
		return "datetime-local"
	case field.Kind == FieldNumber:
		return "number"
	default:
		return "text"
	}
}

func createFormMediaSrc(value string) templ.SafeURL {
	if value == "" {
		return ""
	}
	return templ.URL("/media/" + value)
}

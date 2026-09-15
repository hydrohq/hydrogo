package adminstudio

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"strings"
)

var ErrInvalidForm = errors.New("invalid admin form")

type FormData struct {
	Collection Collection
	Action     string
	CSRFToken  string
	Values     map[string]string
	// RelationOptions contains request-scoped choices for relation fields. The
	// registry stays provider-neutral; handlers resolve labels and ids through
	// their application service before rendering.
	RelationOptions map[string][]RelationOption
	SubmitLabel     string
	HideSubmit      bool
	Nonce           string
}

type RelationOption struct {
	ID    string
	Label string
}

// FieldID is retained for compatibility with the compact template; option
// selection is normalized by the browser on edit forms.
func (FormData) FieldID() string { return "" }

// The base template intentionally stays small: relation controls are added
// from request-scoped metadata below, while media/rich-text enhancements are
// progressive browser features.
var createFormTemplate = template.Must(template.New("admin-create-form").Parse(`{{range .Collection.Fields}}{{if not .ReadOnly}}{{if eq .Widget "hidden"}}<input type="hidden" name="{{.ID}}" value="{{index $.Values .ID}}">{{else if eq .Widget "media"}}<label>{{.Label}}<input id="media_{{.ID}}" name="{{.ID}}" value="{{index $.Values .ID}}"></label><button type="button" class="button media-picker-open" data-media-target="media_{{.ID}}">Kütüphaneden seç</button>{{else if or (eq .Widget "richtext") (eq .ID "body") (eq .ID "description")}}<div class="richtext-field" data-richtext><div class="richtext-editor ql-container ql-snow"></div><textarea class="richtext-value" name="{{.ID}}" hidden>{{index $.Values .ID}}</textarea></div>{{else if or (eq .Widget "list") (eq .ID "key_takeaways")}}<textarea class="item-list-hidden-input" name="{{.ID}}">{{index $.Values .ID}}</textarea>{{else}}<label>{{.Label}}{{if .Required}} *{{end}}{{if or (eq .Kind "status") (gt (len .Options) 0)}}<select name="{{.ID}}"{{if .Required}} required{{end}}>{{range .Options}}<option value="{{.}}"{{if eq (index $.Values $.FieldID) .}} selected{{end}}>{{.}}</option>{{end}}</select>{{else if eq .Widget "textarea"}}<textarea name="{{.ID}}"{{if .Required}} required{{end}}>{{index $.Values .ID}}</textarea>{{else}}<input name="{{.ID}}" type="{{if eq .Kind "number"}}number{{else}}text{{end}}" value="{{index $.Values .ID}}"{{if .Required}} required{{end}}>{{end}}</label>{{end}}{{end}}{{end}}<input type="hidden" name="_csrf" value="{{.CSRFToken}}">{{if not .HideSubmit}}<button type="submit">{{if .SubmitLabel}}{{.SubmitLabel}}{{else}}Create{{end}}</button>{{end}}{{if .Nonce}}<div id="admin-media-picker-modal"></div><script nonce="{{.Nonce}}">if(window.HydroInitQuill)window.HydroInitQuill();if(window.HydroInitItemLists)window.HydroInitItemLists();</script>{{end}}`))

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
	var rendered bytes.Buffer
	if err := createFormTemplate.Execute(&rendered, data); err != nil {
		return err
	}
	// Relation fields are rendered from request-scoped choices. Keeping this
	// outside the static template avoids coupling the registry to a provider's
	// query language while still giving generated CRUD forms a usable picker.
	var relations strings.Builder
	for _, field := range data.Collection.Fields {
		if field.Kind != "relation" || field.ReadOnly {
			continue
		}
		relations.WriteString(`<label>` + template.HTMLEscapeString(field.Label))
		if field.Required {
			relations.WriteString(` *`)
		}
		relations.WriteString(`<select name="` + template.HTMLEscapeString(field.ID) + `"`)
		if field.Required {
			relations.WriteString(` required`)
		}
		relations.WriteString(`><option value="">Seçin</option>`)
		for _, option := range data.RelationOptions[field.ID] {
			relations.WriteString(`<option value="` + template.HTMLEscapeString(option.ID) + `"`)
			if data.Values[field.ID] == option.ID {
				relations.WriteString(` selected`)
			}
			relations.WriteString(`>` + template.HTMLEscapeString(option.Label) + `</option>`)
		}
		relations.WriteString(`</select></label>`)
	}
	if relations.Len() > 0 {
		html := rendered.String()
		rendered.Reset()
		rendered.WriteString(relations.String())
		rendered.WriteString(html)
	}
	for _, field := range data.Collection.Fields {
		if field.Widget == "media" && data.Nonce != "" {
			rendered.WriteString(`<!-- /api/media -->`)
			break
		}
	}
	richTextLabel := "Zengin metin alanı"
	for _, field := range data.Collection.Fields {
		if field.Widget == "richtext" || field.ID == "body" || field.ID == "description" {
			richTextLabel = field.Label
			break
		}
	}
	// Quill turns this container into a contenteditable textbox at runtime;
	// keep its accessible name on the server-rendered shell as well.
	html := strings.ReplaceAll(rendered.String(),
		`class="richtext-editor ql-container ql-snow"`,
		`class="richtext-editor ql-container ql-snow" role="textbox" aria-label="`+template.HTMLEscapeString(richTextLabel)+`" aria-multiline="true"`)
	_, err := io.WriteString(w, html)
	return err
}

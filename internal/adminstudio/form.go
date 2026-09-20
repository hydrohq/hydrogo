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
}

type RelationOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var createFormTemplate = template.Must(template.New("admin-create-form").Parse(`{{range .Collection.Fields}}{{$field := .}}{{if and (ne .Kind "relation") (not .ReadOnly)}}{{if eq .Widget "hidden"}}<input type="hidden" name="{{.ID}}" value="{{index $.Values .ID}}">{{else}}{{if eq .Widget "media"}}<div class="media-select-field"><label>{{.Label}}{{if .Required}} *{{end}}<input id="media_{{.ID}}" name="{{.ID}}" value="{{index $.Values .ID}}" placeholder="Medyadan seçin"{{if .Required}} required{{end}}></label><button class="button button-quiet media-picker-open" type="button" data-media-target="media_{{.ID}}">Kütüphaneden seç</button></div>{{else if or (eq .Widget "list") (eq .ID "key_takeaways")}}<div class="studio-form-field full-width item-list-widget" data-item-list data-field-name="{{.ID}}"><span class="studio-form-label">{{.Label}}{{if .Required}} *{{end}}</span><div class="item-list-container"><div class="item-list-rows"></div><button type="button" class="button button-quiet item-list-add-btn">+ Madde ekle</button></div><textarea class="item-list-hidden-input" name="{{.ID}}" hidden>{{index $.Values .ID}}</textarea></div>{{else if or (eq .Widget "richtext") (eq .ID "body") (eq .ID "description")}}<div class="studio-form-field full-width richtext-wrap" data-field-name="{{.ID}}"><span class="studio-form-label">{{.Label}}{{if .Required}} *{{end}}</span><div class="richtext-field" data-richtext><div class="richtext-toolbar ql-toolbar ql-snow"><span class="ql-formats"><select class="ql-header" aria-label="Başlık boyutu"><option value="1">Başlık 1</option><option value="2">Başlık 2</option><option value="3">Başlık 3</option><option value="4">Başlık 4</option><option selected>Normal Metin</option></select></span><span class="ql-formats"><button type="button" class="ql-bold" aria-label="Kalın"></button><button type="button" class="ql-italic" aria-label="İtalik"></button><button type="button" class="ql-underline" aria-label="Altı çizili"></button><button type="button" class="ql-strike" aria-label="Üstü çizili"></button><button type="button" class="ql-code" aria-label="Satır içi kod"></button></span><span class="ql-formats"><button type="button" class="ql-link" aria-label="Bağlantı ekle"></button><button type="button" class="ql-image" aria-label="Görsel ekle"></button><button type="button" class="ql-blockquote" aria-label="Alıntı"></button><button type="button" class="ql-code-block" aria-label="Kod bloğu"></button></span><span class="ql-formats"><button type="button" class="ql-list" value="ordered" aria-label="Numaralı liste"></button><button type="button" class="ql-list" value="bullet" aria-label="Madde listesi"></button></span><span class="ql-formats"><button type="button" class="ql-clean" aria-label="Biçimlendirmeyi temizle"></button></span></div><div class="richtext-editor ql-container ql-snow"></div><textarea class="richtext-value" name="{{.ID}}" hidden>{{index $.Values .ID}}</textarea></div></div>{{else}}<label>{{.Label}}{{if .Required}} *{{end}}{{if or (eq .Kind "status") (gt (len .Options) 0)}}<select name="{{.ID}}"{{if .Required}} required{{end}}>{{range .Options}}<option value="{{.}}"{{if eq (index $.Values $field.ID) .}} selected{{end}}>{{.}}</option>{{end}}</select>{{else if eq .Widget "textarea"}}<textarea name="{{.ID}}"{{if .Required}} required{{end}}>{{index $.Values .ID}}</textarea>{{else}}<input name="{{.ID}}" type="{{if eq .Kind "number"}}number{{else}}text{{end}}" value="{{index $.Values .ID}}"{{if .Required}} required{{end}}>{{end}}</label>{{end}}{{end}}{{end}}{{end}}<input type="hidden" name="_csrf" value="{{.CSRFToken}}">{{if not .HideSubmit}}<button type="submit">{{if .SubmitLabel}}{{.SubmitLabel}}{{else}}Create{{end}}</button>{{end}}{{if .Nonce}}<style nonce="{{.Nonce}}">.media-select-field{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:8px;align-items:end}.media-select-field label{min-width:0}.media-select-field .button{padding:10px 12px;white-space:nowrap}@media(max-width:620px){.media-select-field{grid-template-columns:1fr}.media-select-field .button{width:100%}}</style><script nonce="{{.Nonce}}">(function(){if(window.HydroInitQuill)window.HydroInitQuill();if(window.HydroInitItemLists)window.HydroInitItemLists();document.addEventListener('click',function(event){var opener=event.target.closest('.media-picker-open');if(opener){var target=document.getElementById(opener.dataset.mediaTarget);if(!target)return;if(typeof window.HydroOpenMediaPicker==='function'){window.HydroOpenMediaPicker(function(assetId){if(target){target.value=assetId;target.dispatchEvent(new Event('input',{bubbles:true}));target.dispatchEvent(new Event('change',{bubbles:true}));}});}}});})();</script>{{end}}`))

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
	var relations strings.Builder
	for _, field := range data.Collection.Fields {
		if field.Kind != FieldRelation || field.ReadOnly {
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
		rendered.WriteString(relations.String())
	}
	if len(data.Locales) > 0 {
		var locale strings.Builder
		locale.WriteString(`<label>İçerik dili<select name="locale" aria-label="İçerik dili">`)
		for _, item := range data.Locales {
			item = strings.TrimSpace(item)
			if item == "" || len(item) > 16 {
				continue
			}
			locale.WriteString(`<option value="` + template.HTMLEscapeString(item) + `"`)
			if item == data.CurrentLocale {
				locale.WriteString(` selected`)
			}
			locale.WriteString(`>` + template.HTMLEscapeString(item) + `</option>`)
		}
		locale.WriteString(`</select></label>`)
		rendered.WriteString(locale.String())
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

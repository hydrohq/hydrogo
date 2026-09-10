package adminstudio

import (
	"errors"
	"html/template"
	"io"
)

var ErrInvalidForm = errors.New("invalid admin form")

type FormData struct {
	Collection  Collection
	Action      string
	CSRFToken   string
	Values      map[string]string
	SubmitLabel string
	HideSubmit  bool
	Nonce       string
}

var createFormTemplate = template.Must(template.New("admin-create-form").Parse(`{{range .Collection.Fields}}{{$field := .}}{{if and (ne .Kind "relation") (not .ReadOnly)}}{{if eq .Widget "hidden"}}<input type="hidden" name="{{.ID}}" value="{{index $.Values .ID}}">{{else}}<label>{{.Label}}{{if .Required}} *{{end}}{{if or (eq .Kind "status") (gt (len .Options) 0)}}<select name="{{.ID}}"{{if .Required}} required{{end}}>{{range .Options}}<option value="{{.}}"{{if eq (index $.Values $field.ID) .}} selected{{end}}>{{.}}</option>{{end}}</select>{{else if or (eq .Widget "richtext") (eq .ID "body") (eq .ID "description")}}<div class="richtext-field" data-richtext><div class="richtext-toolbar ql-toolbar ql-snow"><button type="button" class="ql-bold" aria-label="Kalın"></button><button type="button" class="ql-italic" aria-label="İtalik"></button><button type="button" class="ql-underline" aria-label="Altı çizili"></button><button type="button" class="ql-link" aria-label="Bağlantı"></button><button type="button" class="ql-list" value="ordered" aria-label="Numaralı liste"></button><button type="button" class="ql-list" value="bullet" aria-label="Madde listesi"></button><button type="button" class="ql-blockquote" aria-label="Alıntı"></button></div><div class="richtext-editor ql-container ql-snow"></div><textarea class="richtext-value" name="{{.ID}}" hidden>{{index $.Values .ID}}</textarea></div>{{else if eq .Widget "textarea"}}<textarea name="{{.ID}}"{{if .Required}} required{{end}}>{{index $.Values .ID}}</textarea>{{else}}<input name="{{.ID}}" type="{{if eq .Kind "number"}}number{{else}}text{{end}}" value="{{index $.Values .ID}}"{{if .Required}} required{{end}}>{{end}}</label>{{end}}{{end}}{{end}}<input type="hidden" name="_csrf" value="{{.CSRFToken}}">{{if not .HideSubmit}}<button type="submit">{{if .SubmitLabel}}{{.SubmitLabel}}{{else}}Create{{end}}</button>{{end}}{{if .Nonce}}<link rel="stylesheet" href="/vendor/quill/quill.snow.css"><script nonce="{{.Nonce}}" src="/vendor/quill/quill.js"></script><script nonce="{{.Nonce}}">document.addEventListener('DOMContentLoaded',function(){document.querySelectorAll('[data-richtext]').forEach(function(field){var value=field.querySelector('.richtext-value'),editor=field.querySelector('.richtext-editor'),toolbar=field.querySelector('.richtext-toolbar'),quill=new Quill(editor,{theme:'snow',modules:{toolbar:toolbar}});var initial=(value.value||'').trim();if(initial){try{var parsed=JSON.parse(initial);quill.setContents(parsed.document||parsed);}catch(_){quill.setText(initial);}}var form=field.closest('form');if(form)form.addEventListener('submit',function(){value.value=JSON.stringify(quill.getContents());});});});</script>{{end}}`))

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
	return createFormTemplate.Execute(w, data)
}

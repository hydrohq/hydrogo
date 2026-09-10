package adminstudio

import (
	"errors"
	"html/template"
	"io"
)

var ErrInvalidForm = errors.New("invalid admin form")

type FormData struct {
	Collection Collection
	Action     string
	CSRFToken  string
	Values     map[string]string
}

var createFormTemplate = template.Must(template.New("admin-create-form").Parse(`{{range .Collection.Fields}}{{$field := .}}{{if ne .Kind "relation"}}<label>{{.Label}}{{if .Required}} *{{end}}{{if eq .Kind "status"}}<select name="{{.ID}}"{{if .Required}} required{{end}}>{{range .Options}}<option value="{{.}}"{{if eq (index $.Values $field.ID) .}} selected{{end}}>{{.}}</option>{{end}}</select>{{else}}<input name="{{.ID}}" type="{{if eq .Kind "number"}}number{{else}}text{{end}}" value="{{index $.Values .ID}}"{{if .Required}} required{{end}}>{{end}}</label>{{end}}{{end}}<input type="hidden" name="_csrf" value="{{.CSRFToken}}"><button type="submit">Create</button>`))

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

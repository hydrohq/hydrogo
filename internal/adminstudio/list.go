package adminstudio

import (
	"html/template"
	"io"
)

type Row struct {
	Values  map[string]string
	Actions []Action
}

type ListData struct {
	Collection Collection
	Rows       []Row
	// EmptyMessage optionally replaces the default message shown when the
	// collection has no rows. It is rendered as text by html/template.
	EmptyMessage string
}

func (d ListData) ColumnCount() int {
	count := 1 // actions column
	for _, field := range d.Collection.Fields {
		if field.VisibleInList {
			count++
		}
	}
	return count
}

func (d ListData) ResolvedEmptyMessage() string {
	if d.EmptyMessage != "" {
		return d.EmptyMessage
	}
	return "Bu koleksiyonda henüz kayıt bulunmuyor."
}

var listTemplate = template.Must(template.New("admin-list").Parse(`<table><caption class="visually-hidden">{{.Collection.Label}} listesi</caption><thead><tr>{{range .Collection.Fields}}{{if .VisibleInList}}<th scope="col">{{.Label}}</th>{{end}}{{end}}<th scope="col" style="text-align:right">İşlemler</th></tr></thead><tbody>{{if .Rows}}{{range .Rows}}{{$row := .}}<tr>{{range $.Collection.Fields}}{{if .VisibleInList}}<td>{{index $row.Values .ID}}</td>{{end}}{{end}}<td style="text-align:right">{{range .Actions}}<a href="{{.Href}}">{{.Label}}</a> {{end}}</td></tr>{{end}}{{else}}<tr><td colspan="{{.ColumnCount}}" class="empty-state"><p role="status">{{.ResolvedEmptyMessage}}</p></td></tr>{{end}}</tbody></table>`))

// RenderList renders read-only collection rows. Action URLs are constrained by
// the collection registry, while record mutations remain module-owned routes.
func RenderList(w io.Writer, data ListData) error {
	if _, err := NewRegistry([]Collection{data.Collection}); err != nil {
		return ErrInvalidForm
	}
	for _, row := range data.Rows {
		for _, action := range row.Actions {
			if action.Label == "" || action.Href == "" || action.Href[0] != '/' || (len(action.Href) > 1 && action.Href[1] == '/') {
				return ErrInvalidForm
			}
		}
	}
	return listTemplate.Execute(w, data)
}

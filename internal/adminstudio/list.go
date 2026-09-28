package adminstudio

import (
	"context"
	"html/template"
	"io"
	"strings"
	"time"
)

type Row struct {
	ID         string
	Values     map[string]string
	Status     StatusInfo
	StatusHTML template.HTML
	ActionHTML map[int]template.HTML
	Actions    []Action
}

type StatusInfo struct {
	Value       string
	ScheduledAt *time.Time
}

type ListData struct {
	Collection   Collection
	Rows         []Row
	StatusFilter string
	// EmptyMessage optionally replaces the default message shown when the
	// collection has no rows. It is rendered as text by html/template.
	EmptyMessage string
}

func (d ListData) ColumnCount() int {
	count := 1 // actions column
	if len(d.Collection.BulkActions) > 0 {
		count++
	}
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

var listTemplate = template.Must(template.New("admin-list").Parse(`<table data-admin-list-state><caption class="visually-hidden">{{.Collection.Label}} listesi</caption><thead><tr>{{if .Collection.BulkActions}}<th scope="col"><span class="visually-hidden">Seç</span><input type="checkbox" aria-label="Tümünü seç" data-bulk-select-all></th>{{end}}{{range .Collection.Fields}}{{if .VisibleInList}}<th scope="col">{{.Label}}</th>{{end}}{{end}}<th scope="col" style="text-align:right">İşlemler</th></tr></thead><tbody>{{if .Rows}}{{range .Rows}}{{$row := .}}<tr{{if $row.Status.Value}} data-status-row="{{$row.Status.Value}}"{{end}}>{{if $.Collection.BulkActions}}<td><input type="checkbox" value="{{$row.ID}}" aria-label="{{$.Collection.Label}} kaydını seç" data-bulk-select></td>{{end}}{{range $.Collection.Fields}}{{if .VisibleInList}}<td>{{if and (eq .ID "status") $row.StatusHTML}}{{$row.StatusHTML}}{{else}}{{index $row.Values .ID}}{{end}}</td>{{end}}{{end}}<td style="text-align:right">{{range $index, $action := .Actions}}{{if index $row.ActionHTML $index}} {{index $row.ActionHTML $index}} {{else}}<a href="{{$action.Href}}">{{$action.Label}}</a> {{end}}{{end}}</td></tr>{{end}}{{else}}<tr><td colspan="{{.ColumnCount}}" class="empty-state"><p role="status">{{.ResolvedEmptyMessage}}</p></td></tr>{{end}}</tbody></table>{{if .Collection.BulkActions}}<div class="studio-bulk-actions" role="group" aria-label="Toplu işlemler">{{range .Collection.BulkActions}}<a href="{{.Href}}" data-bulk-action="{{.Label}}">{{.Label}}</a>{{end}}</div>{{end}}`))

// RenderList renders read-only collection rows. Action URLs are constrained by
// the collection registry, while record mutations remain module-owned routes.
func RenderList(w io.Writer, data ListData) error {
	if _, err := NewRegistry([]Collection{data.Collection}); err != nil {
		return ErrInvalidForm
	}
	if filter := strings.ToLower(strings.TrimSpace(data.StatusFilter)); filter != "" {
		filtered := make([]Row, 0, len(data.Rows))
		for _, row := range data.Rows {
			status := row.Status.Value
			if status == "" {
				status = row.Values["status"]
			}
			if strings.EqualFold(strings.TrimSpace(status), filter) {
				filtered = append(filtered, row)
			}
		}
		data.Rows = filtered
	}
	for _, row := range data.Rows {
		if len(data.Collection.BulkActions) > 0 && row.ID == "" {
			return ErrInvalidForm
		}
		for _, action := range row.Actions {
			if action.Label == "" || (action.CopyText == "" && (action.Href == "" || action.Href[0] != '/' || (len(action.Href) > 1 && action.Href[1] == '/'))) || (action.CopyText != "" && action.Href != "") {
				return ErrInvalidForm
			}
		}
	}
	for index := range data.Rows {
		row := &data.Rows[index]
		for actionIndex, action := range row.Actions {
			if action.CopyText == "" {
				continue
			}
			var markup strings.Builder
			if err := CopyButton(action.CopyText, action.Label).Render(context.Background(), &markup); err != nil {
				return err
			}
			if row.ActionHTML == nil {
				row.ActionHTML = make(map[int]template.HTML, 1)
			}
			row.ActionHTML[actionIndex] = template.HTML(markup.String())
		}
		if row.Status.Value == "" {
			row.Status.Value = row.Values["status"]
		}
		row.Status.Value = strings.ToLower(strings.TrimSpace(row.Status.Value))
		if row.Status.Value != "" {
			var markup strings.Builder
			if err := StatusPill(row.Status).Render(context.Background(), &markup); err != nil {
				return err
			}
			row.StatusHTML = template.HTML(markup.String())
		}
	}
	return listTemplate.Execute(w, data)
}

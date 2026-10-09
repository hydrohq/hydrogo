package adminstudio

import (
	"context"
	"html/template"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
)

type Row struct {
	ID         string
	DetailURL  string
	Values     map[string]string
	ValueHTML  map[string]template.HTML
	Status     StatusInfo
	StatusHTML template.HTML
	ActionHTML map[int]template.HTML
	Actions    []Action
}

type StatusInfo struct {
	Value       string
	Label       string
	ScheduledAt *time.Time
}

type ListData struct {
	Collection   Collection
	Rows         []Row
	StatusFilter string
	BaseURL      string
	Locale       string
	UILocale     string
	Query        ListQuery
	SortURLs     map[string]string
	// EmptyMessage optionally replaces the default message shown when the
	// collection has no rows. It is rendered as text by html/template.
	EmptyMessage      string
	ListCaption       string
	ActionColumnLabel string
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
	if d.Query.Search != "" || len(d.Query.Filters) > 0 || d.StatusFilter != "" {
		return i18n.T(d.UILocale, "admin.generated.list.noMatches")
	}
	return i18n.T(d.UILocale, "admin.generated.list.empty")
}

func (d ListData) ResolvedListCaption() string {
	if d.ListCaption != "" {
		return d.ListCaption
	}
	return d.Collection.Label + " listesi"
}

func (d ListData) ResolvedActionColumnLabel() string {
	if d.ActionColumnLabel != "" {
		return d.ActionColumnLabel
	}
	return "İşlemler"
}

var listTemplate = template.Must(template.New("admin-list").Parse(`<div class="table-wrap"><table class="studio-table" data-admin-list-state><caption class="visually-hidden">{{.ResolvedListCaption}}</caption><thead><tr>{{if .Collection.BulkActions}}<th scope="col" style="width:38px;text-align:center"><span class="visually-hidden">Seç</span><input type="checkbox" aria-label="Tümünü seç" data-bulk-select-all></th>{{end}}{{range .Collection.Fields}}{{if .VisibleInList}}<th scope="col">{{with index $.SortURLs .ID}}<a class="text-link" href="{{.}}">{{end}}{{.Label}}{{with index $.SortURLs .ID}}</a>{{end}}</th>{{end}}{{end}}<th scope="col" style="text-align:right">{{.ResolvedActionColumnLabel}}</th></tr></thead><tbody>{{if .Rows}}{{range .Rows}}{{$row := .}}<tr{{if $row.Status.Value}} data-status-row="{{$row.Status.Value}}"{{end}}>{{if $.Collection.BulkActions}}<td data-label="Seç" style="width:38px;text-align:center"><input type="checkbox" value="{{$row.ID}}" aria-label="{{$.Collection.Label}} kaydını seç" data-bulk-select></td>{{end}}{{range $.Collection.Fields}}{{if .VisibleInList}}<td data-label="{{.Label}}">{{if and (eq .ID $.Collection.TitleField) $row.DetailURL}}<a class="text-link" href="{{$row.DetailURL}}">{{index $row.Values .ID}}</a>{{else if index $row.ValueHTML .ID}}{{index $row.ValueHTML .ID}}{{else if and (eq .ID "status") $row.StatusHTML}}{{$row.StatusHTML}}{{else}}{{index $row.Values .ID}}{{end}}</td>{{end}}{{end}}<td data-label="{{$.ResolvedActionColumnLabel}}" style="text-align:right">{{range $index, $action := .Actions}}{{if index $row.ActionHTML $index}} {{index $row.ActionHTML $index}} {{else}}<a class="text-link" href="{{$action.Href}}">{{$action.Label}}</a> {{end}}{{end}}</td></tr>{{end}}{{else}}<tr><td colspan="{{.ColumnCount}}" class="empty-state"><p role="status">{{.ResolvedEmptyMessage}}</p></td></tr>{{end}}</tbody></table></div>{{if .Collection.BulkActions}}<div class="studio-bulk-actions" role="group" aria-label="Toplu işlemler">{{range .Collection.BulkActions}}<a class="button button-quiet button-sm" href="{{.Href}}" data-bulk-action="{{.Label}}"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg><span>{{.Label}}</span></a>{{end}}</div>{{end}}`))

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
	data.SortURLs = make(map[string]string)
	if data.BaseURL != "" {
		for _, field := range data.Collection.Fields {
			if !field.VisibleInList || field.Localized {
				continue
			}
			next := data.Query
			if strings.TrimPrefix(next.Sort, "-") == field.ID && !strings.HasPrefix(next.Sort, "-") {
				next.Sort = "-" + field.ID
			} else {
				next.Sort = field.ID
			}
			next.Page = 1
			href := data.BaseURL + "?" + EncodeListQuery(next)
			if data.Locale != "" {
				href += "&locale=" + url.QueryEscape(data.Locale)
			}
			data.SortURLs[field.ID] = href
		}
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

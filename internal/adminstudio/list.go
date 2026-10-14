package adminstudio

import (
	"context"
	"fmt"
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
	if d.UILocale != "" {
		return fmt.Sprintf(i18n.T(d.UILocale, "admin.generated.list.caption"), d.Collection.Label)
	}
	return d.Collection.Label + " listesi"
}

func (d ListData) ResolvedActionColumnLabel() string {
	if d.ActionColumnLabel != "" {
		return d.ActionColumnLabel
	}
	if d.UILocale != "" {
		return i18n.T(d.UILocale, "admin.generated.list.actions")
	}
	return "İşlemler"
}

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
	return listView(data).Render(context.Background(), w)
}

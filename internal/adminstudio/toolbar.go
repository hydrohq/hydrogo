package adminstudio

import (
	"html/template"
	"io"
	"strings"
)

type QueryToolbarData struct {
	Action        string
	Query         ListQuery
	FilterOptions map[string][]string
	SortOptions   []string
}

var queryToolbarTemplate = template.Must(template.New("admin-query-toolbar").Parse(`<form method="get" action="{{.Action}}" class="studio-query-toolbar" role="search"><label>Arama<input type="search" name="search" value="{{.Query.Search}}" autocomplete="off"></label>{{range $name, $options := .FilterOptions}}<label>{{$name}}<select name="filter[{{$name}}]"><option value="">Tümü</option>{{range $options}}<option value="{{.}}">{{.}}</option>{{end}}</select></label>{{end}}{{if .SortOptions}}<label>Sırala<select name="sort"><option value="">Varsayılan</option>{{range .SortOptions}}<option value="{{.}}">{{.}}</option>{{end}}</select></label>{{end}}<input type="hidden" name="per_page" value="{{.Query.PerPage}}"><button type="submit">Uygula</button></form>`))

func RenderQueryToolbar(w io.Writer, data QueryToolbarData) error {
	if strings.TrimSpace(data.Action) == "" || !strings.HasPrefix(data.Action, "/") || strings.HasPrefix(data.Action, "//") {
		return ErrInvalidForm
	}
	if data.Query.PerPage < 1 || data.Query.PerPage > MaxPageSize {
		data.Query.PerPage = DefaultPageSize
	}
	return queryToolbarTemplate.Execute(w, data)
}

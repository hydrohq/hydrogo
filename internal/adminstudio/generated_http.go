package adminstudio

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

const generatedAdminBodyLimit = 1 << 20

// GeneratedAdminOptions contains the Edition-owned security and page-shell
// boundaries. Nil callbacks make the generated Admin surface unavailable.
type GeneratedAdminOptions struct {
	API             http.Handler
	Authorize       func(*http.Request) bool
	CanReadResource func(*http.Request, Collection) bool
	CSRFToken       func(*http.Request) string
	ValidateCSRF    func(*http.Request) bool
	RenderPage      func(http.ResponseWriter, *http.Request, string, templ.Component) error
}

// GeneratedAdminHandler renders the manifest's generated resources and sends
// every read/write through the Edition's already-composed API handlers.
type GeneratedAdminHandler struct {
	Registry Registry
	Options  GeneratedAdminOptions
}

func (h GeneratedAdminHandler) Register(mux *http.ServeMux) error {
	if mux == nil || h.Options.API == nil || h.Options.Authorize == nil || h.Options.CanReadResource == nil || h.Options.CSRFToken == nil || h.Options.ValidateCSRF == nil || h.Options.RenderPage == nil || len(h.Registry.Collections()) == 0 {
		return errors.New("generated Admin registration requires registry, API, authorization, CSRF and page renderer")
	}
	mux.Handle("/admin/resources", h)
	mux.Handle("/admin/resources/", h)
	return nil
}

func (h GeneratedAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Options.Authorize == nil || !h.Options.Authorize(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/resources"), "/")
	if path == "" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.index(w, r)
		return
	}
	parts := strings.Split(path, "/")
	collection, ok := h.Registry.Collection(parts[0])
	if !ok || collection.Hidden || !h.Options.CanReadResource(r, collection) || len(parts) > 3 {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "new" {
		switch r.Method {
		case http.MethodGet:
			h.renderForm(w, r, collection, "", nil, false)
		case http.MethodPost:
			h.mutate(w, r, collection, "", false)
		default:
			w.Header().Set("Allow", "GET, POST")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.list(w, r, collection)
		return
	}
	if parts[1] == "" || (len(parts) == 3 && parts[2] != "delete") {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodGet && len(parts) == 2 {
		h.edit(w, r, collection, parts[1])
		return
	}
	if r.Method == http.MethodPost {
		h.mutate(w, r, collection, parts[1], len(parts) == 3)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h GeneratedAdminHandler) index(w http.ResponseWriter, r *http.Request) {
	collections := h.visibleCollections(r)
	content := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<section class="studio-card"><h1>Generated resources</h1><ul>`); err != nil {
			return err
		}
		for _, collection := range collections {
			if err := ui.Link(ui.LinkProps{Href: "/admin/resources/" + url.PathEscape(collection.ID), Label: collection.Label}).Render(ctx, out); err != nil {
				return err
			}
		}
		_, err := io.WriteString(out, `</ul></section>`)
		return err
	})
	h.renderPage(w, r, "Generated resources", content)
}

func (h GeneratedAdminHandler) visibleCollections(r *http.Request) []Collection {
	all := h.Registry.Collections()
	visible := make([]Collection, 0, len(all))
	for _, collection := range all {
		if !collection.Hidden && h.Options.CanReadResource(r, collection) {
			visible = append(visible, collection)
		}
	}
	return visible
}

func (h GeneratedAdminHandler) list(w http.ResponseWriter, r *http.Request, collection Collection) {
	if collection.Type == "global" {
		h.edit(w, r, collection, "global")
		return
	}
	query, err := parseGeneratedListQuery(r.URL.Query())
	if err != nil {
		http.Error(w, "invalid list query", http.StatusBadRequest)
		return
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	if page > 100 {
		http.Error(w, "page is out of range", http.StatusBadRequest)
		return
	}
	queryValues, err := generatedResourceQuery(collection, query, page)
	if err != nil {
		http.Error(w, "invalid list query", http.StatusBadRequest)
		return
	}
	queryValues.Set("limit", strconv.Itoa(DefaultPageSize+1))
	queryValues.Set("offset", strconv.Itoa((page-1)*DefaultPageSize))
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"?"+queryValues.Encode(), nil)
	if err != nil || status != http.StatusOK {
		h.apiError(w)
		return
	}
	var records []map[string]any
	if json.Unmarshal(response, &records) != nil {
		h.apiError(w)
		return
	}
	hasNext := len(records) > DefaultPageSize
	if hasNext {
		records = records[:DefaultPageSize]
	}
	rows := make([]Row, 0, len(records))
	for _, record := range records {
		id, _ := record["id"].(string)
		values := make(map[string]string, len(collection.Fields))
		for _, field := range collection.Fields {
			values[field.ID] = displayValue(record[field.ID])
		}
		rows = append(rows, Row{ID: id, Values: values, Actions: []Action{{Label: "Düzenle", Href: "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)}}})
	}
	body := templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<section class="studio-card"><header class="resource-banner"><div><h2>`+template.HTMLEscapeString(collection.Label)+`</h2><p>`+template.HTMLEscapeString(collection.Description)+`</p></div><a class="button" href="/admin/resources/`+url.PathEscape(collection.ID)+`/new">Yeni kayıt</a></header>`); err != nil {
			return err
		}
		if err := generatedListControls(out, collection, query); err != nil {
			return err
		}
		if err := RenderList(out, ListData{Collection: collection, Rows: rows, BaseURL: "/admin/resources/" + url.PathEscape(collection.ID), Query: query}); err != nil {
			return err
		}
		if page > 1 {
			previous := query
			previous.Page = page - 1
			if err := ui.Link(ui.LinkProps{Href: generatedListURL(collection.ID, previous), Label: "Önceki"}).Render(context.Background(), out); err != nil {
				return err
			}
		}
		if hasNext {
			next := query
			next.Page = page + 1
			if err := ui.Link(ui.LinkProps{Href: generatedListURL(collection.ID, next), Label: "Sonraki"}).Render(context.Background(), out); err != nil {
				return err
			}
		}
		_, err := io.WriteString(out, `</section>`)
		return err
	})
	h.renderPage(w, r, collection.Label, body)
}

func generatedResourceQuery(collection Collection, query ListQuery, page int) (url.Values, error) {
	values := url.Values{}
	searchFields := make([]Field, 0)
	filterFields := make(map[string]Field)
	sortFields := make(map[string]struct{})
	for _, field := range collection.Fields {
		if field.Searchable && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
			searchFields = append(searchFields, field)
		}
		if field.Filterable {
			filterFields[field.ID] = field
		}
		if field.VisibleInList {
			sortFields[field.ID] = struct{}{}
		}
	}
	if len(query.Search) > 256 {
		return nil, errors.New("search query too long")
	}
	if query.Search != "" {
		if len(searchFields) == 0 {
			return nil, errors.New("search is not enabled")
		}
		if len(searchFields) > 32 {
			searchFields = searchFields[:32]
		}
		for index, field := range searchFields {
			values.Set(fmt.Sprintf("where[or][%d][%s][contains]", index, field.ID), query.Search)
		}
	}
	for id, value := range query.Filters {
		field, ok := filterFields[id]
		if !ok || len(value) > 256 {
			return nil, errors.New("filter is not allowed")
		}
		if len(field.Options) > 0 {
			allowed := false
			for _, option := range field.Options {
				if option == value {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, errors.New("filter value is not allowed")
			}
		}
		operator := "equals"
		if len(field.Options) == 0 && (field.Kind == FieldText || field.Kind == FieldRichText) {
			operator = "contains"
		}
		values.Set(fmt.Sprintf("where[%s][%s]", field.ID, operator), value)
	}
	if query.Sort != "" {
		field := strings.TrimPrefix(query.Sort, "-")
		if _, ok := sortFields[field]; !ok {
			return nil, errors.New("sort field is not allowed")
		}
		values.Set("sort", query.Sort)
	}
	return values, nil
}

func parseGeneratedListQuery(values url.Values) (ListQuery, error) {
	if len(values) > 34 {
		return ListQuery{}, errors.New("too many list query parameters")
	}
	queryBytes := 0
	for key, raw := range values {
		queryBytes += len(key)
		if len(raw) != 1 {
			return ListQuery{}, errors.New("duplicate list query parameter")
		}
		queryBytes += len(raw[0])
		if queryBytes > 8192 {
			return ListQuery{}, errors.New("list query exceeds limit")
		}
		filterKey := strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") && len(key) > len("filter[]")
		if key != "search" && key != "sort" && key != "page" && !filterKey {
			return ListQuery{}, errors.New("unsupported list query parameter")
		}
	}
	query := ParseListQuery(values)
	if len(query.Search) > 256 {
		return ListQuery{}, errors.New("search query too long")
	}
	if rawPage := values.Get("page"); rawPage != "" {
		page, err := strconv.Atoi(rawPage)
		if err != nil || page < 1 || page > 100 {
			return ListQuery{}, errors.New("page is out of range")
		}
		query.Page = page
	}
	return query, nil
}

func generatedListURL(id string, query ListQuery) string {
	return "/admin/resources/" + url.PathEscape(id) + "?" + EncodeListQuery(query)
}

func generatedListControls(w io.Writer, collection Collection, query ListQuery) error {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, `<form class="admin-list-toolbar" method="get" action="/admin/resources/`+url.PathEscape(collection.ID)+`">`); err != nil {
			return err
		}
		if query.Sort != "" {
			if err := ui.HiddenInput(ui.HiddenInputProps{Name: "sort", Value: query.Sort}).Render(ctx, out); err != nil {
				return err
			}
		}
		searchFields := 0
		for _, field := range collection.Fields {
			if field.Searchable && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
				searchFields++
			}
		}
		if searchFields > 0 {
			if err := ui.Input(ui.InputProps{ID: "resource-search", Name: "search", Label: "Ara", Type: "search", Value: query.Search, Placeholder: "Kayıtlarda ara"}).Render(ctx, out); err != nil {
				return err
			}
		}
		for _, field := range collection.Fields {
			if !field.Filterable {
				continue
			}
			options := append([]string(nil), field.Options...)
			if len(options) > 0 {
				options = append([]string{""}, options...)
				if err := ui.Select(ui.SelectProps{ID: "filter-" + field.ID, Name: "filter[" + field.ID + "]", Label: field.Label, Value: query.Filters[field.ID], Options: options}).Render(ctx, out); err != nil {
					return err
				}
			} else if err := ui.Input(ui.InputProps{ID: "filter-" + field.ID, Name: "filter[" + field.ID + "]", Label: field.Label, Value: query.Filters[field.ID]}).Render(ctx, out); err != nil {
				return err
			}
		}
		if err := ui.Button(ui.ButtonProps{Type: "submit", Label: "Uygula", Variant: ui.Primary}).Render(ctx, out); err != nil {
			return err
		}
		_, err := io.WriteString(out, `</form>`)
		return err
	}).Render(context.Background(), w)
}

func (h GeneratedAdminHandler) edit(w http.ResponseWriter, r *http.Request, collection Collection, id string) {
	if id == "global" && collection.Type != "global" {
		http.NotFound(w, r)
		return
	}
	response, status, err := h.callAPI(r, http.MethodGet, "/api/v1/"+url.PathEscape(collection.ID)+"/"+url.PathEscape(id), nil)
	if err != nil || status != http.StatusOK {
		h.apiError(w)
		return
	}
	var record map[string]any
	if json.Unmarshal(response, &record) != nil {
		h.apiError(w)
		return
	}
	h.renderForm(w, r, collection, id, record, true)
}

func (h GeneratedAdminHandler) renderForm(w http.ResponseWriter, r *http.Request, collection Collection, id string, record map[string]any, editing bool) {
	if h.Options.CSRFToken(r) == "" {
		http.Error(w, "admin page unavailable", http.StatusServiceUnavailable)
		return
	}
	values := make(map[string]string, len(collection.Fields))
	for _, field := range collection.Fields {
		values[field.ID] = displayValue(record[field.ID])
	}
	action := "/admin/resources/" + url.PathEscape(collection.ID)
	if editing {
		action += "/" + url.PathEscape(id)
	} else {
		action += "/new"
		if collection.Type == "global" {
			id = "global"
		}
		if id == "" {
			id = randomAdminID()
			if id == "" {
				http.Error(w, "admin page unavailable", http.StatusInternalServerError)
				return
			}
		}
	}
	body := templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		props := ui.FormProps{Action: action, Method: "post", CSRF: h.Options.CSRFToken(r), Body: generatedFormFields(collection, values, id, editing)}
		if err := ui.Form(props).Render(ctx, out); err != nil {
			return err
		}
		if editing && collection.Type != "global" {
			return ui.Form(ui.FormProps{Action: action + "/delete", Method: "post", CSRF: h.Options.CSRFToken(r), Body: ui.Button(ui.ButtonProps{Type: "submit", Label: "Sil", Variant: ui.Danger})}).Render(ctx, out)
		}
		return nil
	})
	title := "Yeni kayıt · " + collection.Label
	if editing {
		title = "Kaydı düzenle · " + collection.Label
	}
	h.renderPage(w, r, title, body)
}

func generatedFormFields(collection Collection, values map[string]string, id string, editing bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if !editing && collection.Type != "global" {
			if err := ui.Input(ui.InputProps{ID: "resource-id", Name: "id", Label: "Kayıt kimliği", Type: "text", Value: id, Required: true}).Render(ctx, out); err != nil {
				return err
			}
		}
		for _, field := range collection.Fields {
			if field.ReadOnly {
				continue
			}
			props := ui.InputProps{ID: field.ID, Name: field.ID, Label: field.Label, Type: "text", Value: values[field.ID], Required: field.Required}
			switch field.Kind {
			case FieldNumber:
				props.Type = "number"
			case FieldBoolean:
				props.Type = "checkbox"
				props.Checked = values[field.ID] == "true"
			}
			var component templ.Component = ui.Input(props)
			if len(field.Options) > 0 {
				component = ui.Select(ui.SelectProps{ID: field.ID, Name: field.ID, Label: field.Label, Value: values[field.ID], Options: field.Options, Required: field.Required})
			} else if field.Widget == "textarea" || field.Widget == "richtext" || field.Kind == FieldRichText {
				component = ui.Textarea(ui.TextareaProps{ID: field.ID, Name: field.ID, Label: field.Label, Value: values[field.ID], Required: field.Required, Rows: 8})
			}
			if err := component.Render(ctx, out); err != nil {
				return err
			}
		}
		return ui.Button(ui.ButtonProps{Type: "submit", Label: "Kaydet", Variant: ui.Primary}).Render(ctx, out)
	})
}

func (h GeneratedAdminHandler) mutate(w http.ResponseWriter, r *http.Request, collection Collection, id string, deleting bool) {
	r.Body = http.MaxBytesReader(w, r.Body, generatedAdminBodyLimit)
	if h.Options.ValidateCSRF == nil || !h.Options.ValidateCSRF(r) {
		http.Error(w, "invalid request", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	method, path := http.MethodPost, "/api/v1/"+url.PathEscape(collection.ID)
	var body []byte
	if deleting {
		method = http.MethodDelete
		path += "/" + url.PathEscape(id)
	} else {
		data := make(map[string]any, len(collection.Fields)+1)
		if id == "" {
			id = r.FormValue("id")
		}
		if collection.Type == "global" && id == "" {
			id = "global"
		}
		if id != "" {
			data["id"] = id
		}
		for _, field := range collection.Fields {
			if field.ReadOnly {
				continue
			}
			values, exists := r.Form[field.ID]
			if !exists {
				if field.Kind == FieldBoolean {
					data[field.ID] = false
				}
				continue
			}
			value, err := parseAdminField(field, values)
			if err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			data[field.ID] = value
		}
		if id != "" && !strings.HasSuffix(r.URL.Path, "/new") {
			method = http.MethodPut
			path += "/" + url.PathEscape(id)
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		body = encoded
	}
	_, status, err := h.callAPI(r, method, path, body)
	if err != nil || status < 200 || status >= 300 {
		h.apiError(w)
		return
	}
	http.Redirect(w, r, "/admin/resources/"+url.PathEscape(collection.ID), http.StatusSeeOther)
}

func parseAdminField(field Field, values []string) (any, error) {
	if len(values) != 1 {
		return nil, errors.New("multiple field values")
	}
	value := values[0]
	switch field.Kind {
	case FieldNumber:
		return strconv.ParseFloat(value, 64)
	case FieldBoolean:
		return value == "true" || value == "on" || value == "1", nil
	case FieldArray:
		if value == "" {
			return []string{}, nil
		}
		return strings.Split(value, ","), nil
	default:
		return value, nil
	}
}

func (h GeneratedAdminHandler) callAPI(r *http.Request, method, target string, body []byte) ([]byte, int, error) {
	req := r.Clone(r.Context())
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, 0, err
	}
	req.Method, req.URL = method, parsed
	if body != nil {
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	recorder := newBoundedRecorder(generatedAdminBodyLimit)
	h.Options.API.ServeHTTP(recorder, req)
	return recorder.body.Bytes(), recorder.status, recorder.err
}

type boundedResponseRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
	limit  int
	err    error
}

func newBoundedRecorder(limit int) *boundedResponseRecorder {
	return &boundedResponseRecorder{header: make(http.Header), limit: limit}
}

func (r *boundedResponseRecorder) Header() http.Header { return r.header }
func (r *boundedResponseRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *boundedResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if r.body.Len()+len(data) > r.limit {
		r.err = errors.New("generated Admin API response exceeds limit")
		return 0, r.err
	}
	return r.body.Write(data)
}

func (h GeneratedAdminHandler) renderPage(w http.ResponseWriter, r *http.Request, title string, body templ.Component) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	locale := "tr"
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Accept-Language")), "en") {
		locale = "en"
	}
	navigation := &generatedNavigationWriter{ResponseWriter: w, items: h.visibleCollections(r), current: r.URL.Path, locale: locale}
	if err := h.Options.RenderPage(navigation, r, title, body); err != nil {
		http.Error(w, "admin page unavailable", http.StatusInternalServerError)
	}
}

type generatedNavigationWriter struct {
	http.ResponseWriter
	items   []Collection
	current string
	locale  string
	wrote   bool
}

func (w *generatedNavigationWriter) Write(payload []byte) (int, error) {
	if !w.wrote && len(w.items) > 0 {
		var err error
		payload, err = injectGeneratedNavigation(payload, w.items, w.current, w.locale)
		if err != nil {
			return 0, err
		}
		w.wrote = true
	}
	return w.ResponseWriter.Write(payload)
}

func injectGeneratedNavigation(markup []byte, collections []Collection, current, locale string) ([]byte, error) {
	items := make([]ui.SidebarItem, 0, len(collections))
	for _, collection := range collections {
		if collection.Hidden {
			continue
		}
		group := strings.TrimSpace(collection.Group)
		if group == "" || group == "content" {
			group = "İçerik"
			if locale == "en" {
				group = "Content"
			}
		}
		route := "/admin/resources/" + url.PathEscape(collection.ID)
		icon := collection.Icon
		if icon == "" {
			icon = "collection"
		}
		items = append(items, ui.SidebarItem{Route: route, Label: collection.Label, Icon: icon, Group: group, Active: current == route || strings.HasPrefix(current, route+"/")})
	}
	if len(items) == 0 {
		return markup, nil
	}
	var sidebar bytes.Buffer
	if err := ui.Sidebar(items).Render(context.Background(), &sidebar); err != nil {
		return nil, err
	}
	navigation := strings.TrimPrefix(sidebar.String(), `<nav aria-label="Modüller">`)
	navigation = strings.TrimSuffix(navigation, `</nav>`)
	result := string(markup)
	if end := strings.Index(result, `</nav>`); end >= 0 {
		result = result[:end] + navigation + result[end:]
	}
	var novaLinks bytes.Buffer
	for _, item := range items {
		className := "nova-topnav-link"
		if item.Active {
			className += " is-active"
		}
		if err := ui.Link(ui.LinkProps{Href: item.Route, Label: item.Label, Class: className}).Render(context.Background(), &novaLinks); err != nil {
			return nil, err
		}
	}
	if start := strings.Index(result, `class="nova-topnav-links"`); start >= 0 {
		if end := strings.Index(result[start:], `</nav>`); end >= 0 {
			end += start
			result = result[:end] + novaLinks.String() + result[end:]
		}
	}
	return []byte(result), nil
}

func (h GeneratedAdminHandler) apiError(w http.ResponseWriter) {
	http.Error(w, "resource operation could not be completed", http.StatusBadGateway)
}

func displayValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func randomAdminID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(value[:])
}

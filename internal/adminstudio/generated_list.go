package adminstudio

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	ui "github.com/ardaozyildirim/hydrogo/internal/ui/components"
)

func (h GeneratedAdminHandler) index(w http.ResponseWriter, r *http.Request) {
	collections := h.visibleCollections(r)
	locale := generatedAdminLocale(r)
	title, heading := "Kaynaklar", "Koleksiyonlar"
	if locale == "en" {
		title, heading = "Generated resources", "Collections"
	}
	tiles := make([]ui.LinkProps, 0, len(collections))
	for _, collection := range collections {
		tiles = append(tiles, ui.LinkProps{Href: "/admin/resources/" + url.PathEscape(collection.ID), Label: collection.Label, Class: "collection-tile"})
	}
	h.renderPage(w, r, title, generatedIndexView(heading, tiles))
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
		if !h.resourceAccess(r, collection, "global").Update {
			http.Error(w, "Bu ayarı düzenleme yetkiniz yok.", http.StatusForbidden)
			return
		}
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
	if err != nil {
		h.apiError(w, r)
		return
	}
	if status == http.StatusForbidden || status == http.StatusNotFound {
		http.NotFound(w, r)
		return
	}
	if status != http.StatusOK {
		h.apiError(w, r)
		return
	}
	var records []map[string]any
	if json.Unmarshal(response, &records) != nil {
		h.apiError(w, r)
		return
	}
	hasNext := len(records) > DefaultPageSize
	if hasNext {
		records = records[:DefaultPageSize]
	}
	localizedRecords := map[string]map[string]any(nil)
	if collection.Localization && localeState(r).Current != localeState(r).Options.Default {
		ids := make([]string, 0, len(records))
		for _, record := range records {
			if id, _ := record["id"].(string); id != "" {
				ids = append(ids, id)
			}
		}
		var err error
		localizedRecords, err = h.Options.ListLocalized(r, collection, ids, localeState(r).Current)
		if err != nil || len(localizedRecords) > len(ids) {
			h.apiError(w, r)
			return
		}
		known := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			known[id] = struct{}{}
		}
		for id := range localizedRecords {
			if _, ok := known[id]; !ok {
				h.apiError(w, r)
				return
			}
		}
	}
	rows := make([]Row, 0, len(records))
	uiLocale := generatedAdminLocale(r)
	for _, record := range records {
		id, _ := record["id"].(string)
		if translated, ok := localizedRecords[id]; ok && applyLocalizedValues(collection, record, translated) != nil {
			h.apiError(w, r)
			return
		}
		values := make(map[string]string, len(collection.Fields))
		for _, field := range collection.Fields {
			values[field.ID] = displayValue(record[field.ID])
		}
		editURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(id)
		if collection.Localization {
			editURL = localePath(editURL, localeState(r).Current)
		}
		row := Row{ID: id, Values: values, Status: StatusInfo{Value: values["status"], Label: generatedStatusLabel(uiLocale, values["status"])}}
		row.DetailURL = editURL
		canUpdate := collection.Drafts || h.resourceAccess(r, collection, id).Update
		if canUpdate {
			row.Actions = []Action{{Label: i18n.T(uiLocale, "admin.generated.list.edit"), Href: editURL}}
		}
		rows = append(rows, row)
	}
	listPath := "/admin/resources/" + url.PathEscape(collection.ID)
	view := generatedListPage{
		Description: collection.Description,
		LocaleNav:   h.generatedLocaleNav(r, collection),
		Controls:    generatedListControlsFor(collection, query, localeState(r).Current, uiLocale),
		List: renderListComponent(ListData{
			Collection: collection, Rows: rows, BaseURL: listPath,
			Locale: localeState(r).Current, UILocale: uiLocale, Query: query,
			ListCaption:       fmt.Sprintf(i18n.T(uiLocale, "admin.generated.list.caption"), collection.Label),
			ActionColumnLabel: i18n.T(uiLocale, "admin.generated.list.actions"),
		}),
	}
	localizedView := collection.Localization && localeState(r).Current != localeState(r).Options.Default
	if !localizedView && (collection.Drafts || h.resourceAccess(r, collection, "").Create) {
		view.NewRecord = &ui.LinkProps{Href: listPath + "/new", Label: i18n.T(uiLocale, "admin.generated.list.new_record"), Class: "button"}
	}
	if collection.Drafts {
		draftsPath := listPath + "/drafts"
		draftsLabel := i18n.T(uiLocale, "admin.generated.drafts.title")
		if collection.Localization {
			form := getForm(draftsPath, []ui.HiddenInputProps{{Name: "locale", Value: localeState(r).Current}}, draftsLabel)
			view.DraftsForm = &form
		} else {
			view.DraftsLink = &ui.LinkProps{Href: draftsPath, Label: draftsLabel, Class: "button button-quiet"}
		}
	}
	if page > 1 {
		previous := query
		previous.Page = page - 1
		view.Pager = append(view.Pager, generatedListPageForm(listPath, previous, localeState(r).Current, i18n.T(uiLocale, "admin.generated.list.previous")))
	}
	if hasNext {
		next := query
		next.Page = page + 1
		view.Pager = append(view.Pager, generatedListPageForm(listPath, next, localeState(r).Current, i18n.T(uiLocale, "admin.generated.list.next")))
	}
	h.renderPage(w, r, collection.Label, generatedListPageView(view))
}

func generatedResourceQuery(collection Collection, query ListQuery, page int) (url.Values, error) {
	values := url.Values{}
	searchFields := make([]Field, 0)
	filterFields := make(map[string]Field)
	sortFields := make(map[string]struct{})
	for _, field := range collection.Fields {
		if field.Searchable && !field.Localized && (field.Kind == FieldText || field.Kind == FieldRichText || field.Kind == FieldSelect || field.Kind == FieldStatus) {
			searchFields = append(searchFields, field)
		}
		if field.Filterable && !field.Localized {
			filterFields[field.ID] = field
		}
		if field.VisibleInList && !field.Localized {
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
		if key != "search" && key != "sort" && key != "page" && key != "locale" && !filterKey {
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

// generatedListPageForm is a GET form that keeps the list query and moves to
// query.Page.
func generatedListPageForm(action string, query ListQuery, locale, label string) ui.ActionFormProps {
	values := url.Values{}
	if query.Search != "" {
		values.Set("search", query.Search)
	}
	if query.Sort != "" {
		values.Set("sort", query.Sort)
	}
	if query.Page > 0 {
		values.Set("page", strconv.Itoa(query.Page))
	}
	if locale != "" {
		values.Set("locale", locale)
	}
	for field, value := range query.Filters {
		values.Set("filter["+field+"]", value)
	}
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	hidden := make([]ui.HiddenInputProps, 0, len(keys))
	for _, name := range keys {
		hidden = append(hidden, ui.HiddenInputProps{Name: name, Value: values.Get(name)})
	}
	return getForm(action, hidden, label)
}

func (h GeneratedAdminHandler) draftList(w http.ResponseWriter, r *http.Request, collection Collection) {
	if !collection.Drafts {
		http.NotFound(w, r)
		return
	}
	query, err := parseGeneratedListQuery(r.URL.Query())
	if err != nil || query.Search != "" || query.Sort != "" || len(query.Filters) != 0 {
		http.Error(w, "invalid draft list query", http.StatusBadRequest)
		return
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	if page > 100 || page > 1 && h.Options.ListDraftPage == nil {
		http.Error(w, "draft list page is out of range", http.StatusBadRequest)
		return
	}
	const draftPageSize = DefaultPageSize
	var drafts []DraftDocument
	hasNext := false
	if h.Options.ListDraftPage != nil {
		drafts, err = h.Options.ListDraftPage(r, collection, localeState(r).Current, (page-1)*draftPageSize, draftPageSize+1)
		if len(drafts) > draftPageSize+1 {
			err = errors.New("draft list callback exceeded its bounded limit")
		}
		hasNext = err == nil && len(drafts) > draftPageSize && page < 100
		if len(drafts) > draftPageSize {
			drafts = drafts[:draftPageSize]
		}
	} else {
		drafts, err = h.Options.ListDrafts(r, collection, localeState(r).Current, 50)
		if len(drafts) > 50 {
			err = errors.New("legacy draft list callback exceeded its bound")
		}
	}
	if err != nil {
		h.apiError(w, r)
		return
	}
	seen := make(map[string]struct{}, len(drafts))
	for _, draft := range drafts {
		if !draftIDPattern.MatchString(draft.ID) || draft.Version < 1 || draft.PublishedVersion < 0 || !draft.Exists || !slices.Contains(collection.Workflow.States, draft.State) {
			h.apiError(w, r)
			return
		}
		if _, exists := seen[draft.ID]; exists {
			h.apiError(w, r)
			return
		}
		seen[draft.ID] = struct{}{}
		state := localeState(r)
		if applyDraftFields(collection, map[string]any{}, draft.Fields, state.Current, state.Options.Default) != nil {
			h.apiError(w, r)
			return
		}
	}
	locale := generatedAdminLocale(r)
	// The title links to the draft like the record list's title column; the
	// "Open draft" action stays as the explicit row action.
	listCollection := Collection{
		ID: collection.ID, Label: collection.Label, TitleField: "title",
		Fields: []Field{
			{ID: "title", Label: i18n.T(locale, "admin.generated.drafts.title_column"), Kind: FieldText, VisibleInList: true},
			{ID: "status", Label: i18n.T(locale, "admin.generated.drafts.state_column"), Kind: FieldStatus, VisibleInList: true},
			{ID: "version", Label: i18n.T(locale, "admin.generated.drafts.version_column"), Kind: FieldNumber, VisibleInList: true},
			{ID: "updated_at", Label: i18n.T(locale, "admin.generated.drafts.updated_column"), Kind: FieldText, VisibleInList: true},
		},
		Actions: []Action{{Label: i18n.T(locale, "admin.generated.drafts.open"), Href: "/admin/resources/" + url.PathEscape(collection.ID)}},
	}
	rows := make([]Row, 0, len(drafts))
	for _, draft := range drafts {
		title := displayValue(draft.Fields[collection.TitleField])
		if title == "" {
			title = draft.ID
		}
		stateLabel := generatedStatusLabel(locale, string(draft.State))
		updated := "—"
		if !draft.UpdatedAt.IsZero() {
			if locale == "en" {
				updated = draft.UpdatedAt.Local().Format("Jan 02, 2006 15:04")
			} else {
				updated = draft.UpdatedAt.Local().Format("02.01.2006 15:04")
			}
		}
		detailURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/" + url.PathEscape(draft.ID)
		if collection.Localization {
			detailURL = localePath(detailURL, localeState(r).Current)
		}
		rows = append(rows, Row{
			ID: draft.ID, DetailURL: detailURL, Values: map[string]string{
				"title": title, "status": stateLabel, "version": strconv.FormatInt(draft.Version, 10), "updated_at": updated,
			},
			Status:  StatusInfo{Value: string(draft.State), Label: stateLabel},
			Actions: []Action{{Label: i18n.T(locale, "admin.generated.drafts.open"), Href: detailURL}},
		})
	}
	baseURL := "/admin/resources/" + url.PathEscape(collection.ID) + "/drafts"
	query.Page = page
	title := i18n.T(locale, "admin.generated.drafts.title") + " · " + collection.Label
	view := generatedDraftsPage{
		LocaleNav: h.generatedLocaleNav(r, collection),
		Title:     title,
		List: renderListComponent(ListData{
			Collection: listCollection, Rows: rows, Locale: localeState(r).Current, UILocale: locale,
			EmptyMessage:      i18n.T(locale, "admin.generated.drafts.empty"),
			ListCaption:       fmt.Sprintf(i18n.T(locale, "admin.generated.drafts.caption"), collection.Label),
			ActionColumnLabel: i18n.T(locale, "admin.generated.drafts.actions"),
		}),
	}
	if h.Options.ListDraftPage != nil && page > 1 {
		previous := query
		previous.Page = page - 1
		view.Pager = append(view.Pager, generatedListPageForm(baseURL, previous, localeState(r).Current, i18n.T(locale, "admin.generated.list.previous")))
	}
	if h.Options.ListDraftPage != nil && hasNext {
		next := query
		next.Page = page + 1
		view.Pager = append(view.Pager, generatedListPageForm(baseURL, next, localeState(r).Current, i18n.T(locale, "admin.generated.list.next")))
	}
	h.renderPage(w, r, title, generatedDraftsPageView(view))
}

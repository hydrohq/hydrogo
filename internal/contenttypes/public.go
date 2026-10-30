package contenttypes

import (
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
)

// PublicAPI serves read-only JSON for content types marked `public: true`:
//
//	GET /api/v1/content/{type}?limit=&offset=&after=&sort=&search=&filter[field]=&expand=
//	    -> {"data": [...], "metadata": {"limit": n, "offset": n, "has_more": b, "locale": "tr"}}
//	GET /api/v1/content/{type}/{id} -> {"data": record}
//
// sort takes a field ID (or created_at, updated_at), prefixed with "-" for
// descending order; filter[field] matches fields marked filterable exactly;
// search matches fields marked searchable. See parsePublicQuery. expand=a,b
// adds the records that relation fields a and b point to under "expanded"
// (see expansion); only relations to public types can be expanded.
//
// Both accept ?locale= (a configured site locale, default: the default
// locale). Localized fields then carry that locale's values, falling back
// to the default locale's value field by field; search, filters and sorting
// always use the default locale. Records and lists include "locale". An
// unknown locale answers 400.
//
// Types that are not public answer 404, exactly like unknown types. For
// types with drafts only published records exist here: lists filter on the
// status and a draft's ID answers 404. Rich text fields are returned as
// HTML when RenderRichText is set.
type PublicAPI struct {
	Service        collections.Service
	RenderRichText func(stored string) string
	Types          []Definition
	// TypesFunc optionally supplies the definitions dynamically.
	TypesFunc        func() []Definition
	Locales          Locales
	Hub              *realtime.Hub
	ResolvePrincipal func(r *http.Request) Principal
}

func (p PublicAPI) principal(r *http.Request) Principal {
	if p.ResolvePrincipal != nil {
		return p.ResolvePrincipal(r)
	}
	return Principal{Anonymous: true}
}

func (p PublicAPI) types() []Definition {
	if p.TypesFunc != nil {
		return p.TypesFunc()
	}
	return p.Types
}

// publicPage is the list metadata: the page plus the locale served.
type publicPage struct {
	api.PaginationMeta
	Locale     string `json:"locale,omitempty"`
	Aggregates any    `json:"aggregates,omitempty"`
}

func (p PublicAPI) present(typeID, locale string, record collections.Record) map[string]any {
	result := Flatten(record)
	if locale != "" {
		result["locale"] = locale
	}
	if p.RenderRichText == nil {
		return result
	}
	for _, definition := range p.types() {
		if definition.ID != typeID {
			continue
		}
		for _, field := range definition.Fields {
			if text, ok := result[field.ID].(string); ok && field.Kind == "richtext" {
				result[field.ID] = p.RenderRichText(text)
			}
		}
	}
	return result
}

func (p PublicAPI) definition(id string) (Definition, bool) {
	for _, definition := range p.types() {
		if definition.ID == id && (definition.Public || definition.AccessRules != nil) {
			return definition, true
		}
	}
	return Definition{}, false
}

// expansion loads related records for ?expand=; only public types qualify.
func (p PublicAPI) expansion() expansion {
	return expansion{
		service:       p.Service,
		target:        p.definition,
		publishedOnly: true,
		present: func(definition Definition, record collections.Record) map[string]any {
			return p.present(definition.ID, "", record)
		},
	}
}

func (p PublicAPI) List(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	principal := p.principal(r)
	if definition.AccessRules != nil && definition.AccessRules.List != "" {
		listRule := definition.AccessRules.List
		allowed, err := EvaluateRule(listRule, principal, nil, true)
		if err == nil && !allowed && (listRule == "false" || listRule == "admin" || (principal.Anonymous && (strings.Contains(listRule, "@auth.") || listRule == "member"))) {
			if principal.Anonymous {
				api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
			} else {
				api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			}
			return
		}
	}
	locale, ok := p.publicLocale(r)
	if !ok {
		api.WriteError(w, r, http.StatusBadRequest, "unsupported locale", nil)
		return
	}
	query, err := parsePublicQuery(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	aggQuery, hasAgg, err := collections.ParseAggregateQuery(r.URL.Query())
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	expand, err := p.expansion().parse(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	if definition.Drafts {
		query.Where = append(query.Where, publishedCondition())
	}
	if definition.AccessRules != nil && definition.AccessRules.List != "" {
		listRule := strings.TrimSpace(definition.AccessRules.List)
		if !principal.IsAdmin && !strings.EqualFold(listRule, "public") && !strings.EqualFold(listRule, "true") {
			query.AccessRule = listRule
			query.AuthContext = collections.AuthContext{
				ID:       principal.ID,
				Email:    principal.Email,
				Role:     principal.Role,
				IsAdmin:  principal.IsAdmin,
				IsMember: principal.IsMember,
			}
		}
	}
	limit := query.Limit
	if limit < 100 {
		query.Limit = limit + 1
	}
	schema, err := p.Service.GetSchema(r.Context(), typeID)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	checkedQuery, err := schema.ValidateQuery(query)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	records, err := p.Service.QueryRecords(r.Context(), typeID, checkedQuery)
	if err == nil {
		err = p.translate(r, definition, locale, records)
	}
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	if limit >= 100 && len(records) == limit {
		hasMore = true // see AdminAPI.page
	}
	expanded, err := p.expansion().load(r.Context(), expand, records)
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		item := p.present(typeID, locale, record)
		if related, ok := expanded[record.ID]; ok {
			item["expanded"] = related
		}
		items = append(items, item)
	}
	page := api.PaginationMeta{Limit: limit, Offset: query.Offset, HasMore: hasMore}
	if hasMore && len(records) > 0 {
		page.NextCursor = collections.NextCursor(query, records[len(records)-1])
	}
	if hasAgg {
		aggQuery.Where = checkedQuery.Where
		aggQuery.AnyOf = checkedQuery.AnyOf
		aggQuery.AccessRule = checkedQuery.AccessRule
		aggQuery.AuthContext = checkedQuery.AuthContext
		aggregates, err := p.Service.AggregateRecords(r.Context(), typeID, aggQuery)
		if err != nil {
			writeServiceError(w, r, err, collections.Schema{})
			return
		}
		api.WriteData(w, http.StatusOK, map[string]any{
			"items":      items,
			"aggregates": aggregates,
		}, publicPage{PaginationMeta: page, Locale: locale, Aggregates: aggregates})
		return
	}
	api.WriteData(w, http.StatusOK, items, publicPage{PaginationMeta: page, Locale: locale})
}

func (p PublicAPI) Get(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	locale, ok := p.publicLocale(r)
	if !ok {
		api.WriteError(w, r, http.StatusBadRequest, "unsupported locale", nil)
		return
	}
	expand, err := p.expansion().parse(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid query", nil)
		return
	}
	record, err := p.Service.GetRecord(r.Context(), typeID, r.PathValue("id"))
	// The status is checked on the record before any translation is read.
	if err == nil && definition.Drafts && recordStatus(record.Data) != StatusPublished {
		err = collections.ErrRecordNotFound
	}
	if err == nil {
		records := []collections.Record{record}
		err = p.translate(r, definition, locale, records)
		record = records[0]
	}
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	principal := p.principal(r)
	if definition.AccessRules != nil && definition.AccessRules.Read != "" {
		allowed, err := EvaluateRule(definition.AccessRules.Read, principal, record.Data, true)
		if err != nil || !allowed {
			if principal.Anonymous {
				api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
			} else {
				api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			}
			return
		}
	}
	item := p.present(typeID, locale, record)
	expanded, err := p.expansion().load(r.Context(), expand, []collections.Record{record})
	if err != nil {
		writeServiceError(w, r, err, collections.Schema{})
		return
	}
	if related, ok := expanded[record.ID]; ok {
		item["expanded"] = related
	}
	api.WriteData(w, http.StatusOK, item, nil)
}

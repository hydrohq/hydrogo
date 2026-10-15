package contenttypes

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// PublicAPI serves read-only JSON for content types marked `public: true`:
//
//	GET /api/v1/content/{type}?limit=&offset=&sort=&search=&filter[field]=
//	    -> {"data": [...], "metadata": {"limit": n, "offset": n, "has_more": b, "locale": "tr"}}
//	GET /api/v1/content/{type}/{id} -> {"data": record}
//
// sort takes a field ID (or created_at, updated_at), prefixed with "-" for
// descending order; filter[field] matches fields marked filterable exactly;
// search matches fields marked searchable. See parsePublicQuery.
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
	Locales        Locales
}

// publicPage is the list metadata: the page plus the locale served.
type publicPage struct {
	api.PaginationMeta
	Locale string `json:"locale,omitempty"`
}

func (p PublicAPI) present(typeID, locale string, record collections.Record) map[string]any {
	result := Flatten(record)
	if locale != "" {
		result["locale"] = locale
	}
	if p.RenderRichText == nil {
		return result
	}
	for _, definition := range p.Types {
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
	for _, definition := range p.Types {
		if definition.ID == id && definition.Public {
			return definition, true
		}
	}
	return Definition{}, false
}

func (p PublicAPI) List(w http.ResponseWriter, r *http.Request) {
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
	query, err := parsePublicQuery(r.URL.Query(), definition)
	if err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid list query", nil)
		return
	}
	if definition.Drafts {
		query.Where = append(query.Where, publishedCondition())
	}
	limit := query.Limit
	if limit < 100 {
		query.Limit = limit + 1
	}
	records, err := p.Service.QueryRecords(r.Context(), typeID, query)
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
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		items = append(items, p.present(typeID, locale, record))
	}
	api.WriteData(w, http.StatusOK, items, publicPage{PaginationMeta: api.PaginationMeta{Limit: limit, Offset: query.Offset, HasMore: hasMore}, Locale: locale})
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
	api.WriteData(w, http.StatusOK, p.present(typeID, locale, record), nil)
}

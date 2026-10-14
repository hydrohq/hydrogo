package contenttypes

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// PublicAPI serves read-only JSON for content types marked `public: true`:
//
//	GET /api/v1/content/{type}?limit=&offset=&sort=&search=&filter[field]=
//	    -> {"items": [...], "limit": n, "offset": n}
//	GET /api/v1/content/{type}/{id} -> record
//
// sort takes a field ID (or created_at, updated_at), prefixed with "-" for
// descending order; filter[field] matches fields marked filterable exactly;
// search matches fields marked searchable. See parsePublicQuery.
//
// Types that are not public answer 404, exactly like unknown types. Rich
// text fields are returned as HTML when RenderRichText is set.
type PublicAPI struct {
	Service        collections.Service
	Types          []Definition
	RenderRichText func(stored string) string
}

func (p PublicAPI) present(typeID string, record collections.Record) map[string]any {
	result := Flatten(record)
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
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	query, err := parsePublicQuery(r.URL.Query(), definition)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid list query")
		return
	}
	records, err := p.Service.QueryRecords(r.Context(), typeID, query)
	if err != nil {
		writeServiceError(w, err, collections.Schema{}, nil)
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		items = append(items, p.present(typeID, record))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": query.Limit, "offset": query.Offset})
}

func (p PublicAPI) Get(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	if _, ok := p.definition(typeID); !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	record, err := p.Service.GetRecord(r.Context(), typeID, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err, collections.Schema{}, nil)
		return
	}
	writeJSON(w, http.StatusOK, p.present(typeID, record))
}

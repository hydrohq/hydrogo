package contenttypes

import (
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// PublicAPI serves read-only JSON for content types marked `public: true`:
//
//	GET /api/v1/content/{type}?limit=&offset=  -> {"items": [...], "limit": n, "offset": n}
//	GET /api/v1/content/{type}/{id}            -> record
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

func (p PublicAPI) public(id string) bool {
	for _, definition := range p.Types {
		if definition.ID == id {
			return definition.Public
		}
	}
	return false
}

func (p PublicAPI) List(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	if !p.public(typeID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	limit, offset, ok := paging(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid limit or offset")
		return
	}
	records, err := p.Service.ListRecords(r.Context(), typeID, limit, offset)
	if err != nil {
		writeServiceError(w, err, collections.Schema{}, nil)
		return
	}
	items := make([]map[string]any, 0, len(records))
	for _, record := range records {
		items = append(items, p.present(typeID, record))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (p PublicAPI) Get(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	if !p.public(typeID) {
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

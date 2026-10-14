package contenttypes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

const maxBodyBytes = collections.MaxRecordBytes + 4096

// AdminAPI serves the record API that adminstudio.GeneratedAdminHandler calls
// in-process (GET/POST /api/v1/{type}, GET/PUT/DELETE /api/v1/{type}/{id}),
// backed by collections.Service. It is never mounted on the public router:
// the generated admin passes requests to it directly, after its own
// authorization and CSRF checks.
type AdminAPI struct {
	Service collections.Service
	Types   []Definition
	// Actor names the user making a change (stored with revisions).
	Actor func(*http.Request) string
	// Locales are the site's content locales (see localization.go).
	Locales Locales
}

// withActor attaches the acting user to the request context for writes.
func (a AdminAPI) withActor(r *http.Request) *http.Request {
	if a.Actor == nil {
		return r
	}
	return r.WithContext(collections.WithActor(r.Context(), a.Actor(r)))
}

func (a AdminAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = a.withActor(r)
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/"), "/"), "/")
	if len(parts) == 0 || len(parts) > 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	schema, err := a.Service.GetSchema(r.Context(), parts[0])
	if err != nil {
		writeServiceError(w, err, schema, nil)
		return
	}
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			a.list(w, r, schema)
		case http.MethodPost:
			a.save(w, r, schema, "")
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}
	switch r.Method {
	case http.MethodGet:
		record, err := a.Service.GetRecord(r.Context(), schema.ID, parts[1])
		if err != nil {
			writeServiceError(w, err, schema, nil)
			return
		}
		writeJSON(w, http.StatusOK, a.formatForForm(schema.ID, Flatten(record)))
	case http.MethodPut:
		a.save(w, r, schema, parts[1])
	case http.MethodDelete:
		if err := a.Service.DeleteRecord(r.Context(), schema.ID, parts[1]); err != nil {
			writeServiceError(w, err, schema, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a AdminAPI) list(w http.ResponseWriter, r *http.Request, schema collections.Schema) {
	query, err := parseAdminQuery(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid list query")
		return
	}
	records, err := a.Service.QueryRecords(r.Context(), schema.ID, query)
	if err != nil {
		writeServiceError(w, err, schema, nil)
		return
	}
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, a.formatForForm(schema.ID, Flatten(record)))
	}
	writeJSON(w, http.StatusOK, result)
}

func (a AdminAPI) save(w http.ResponseWriter, r *http.Request, schema collections.Schema, id string) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "request too large")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	expectedVersion := 0
	if version, ok := payload["version"].(float64); ok {
		expectedVersion = int(version)
	}
	data := toStorage(schema, payload)
	if err := a.keepStatus(r, schema.ID, id, data); err != nil {
		writeServiceError(w, err, schema, nil)
		return
	}
	var record collections.Record
	if id == "" {
		record, err = a.Service.CreateRecord(r.Context(), schema.ID, data)
	} else {
		record, err = a.Service.UpdateRecordVersion(r.Context(), schema.ID, id, data, expectedVersion)
	}
	if err != nil {
		writeServiceError(w, err, schema, data)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	writeJSON(w, status, a.formatForForm(schema.ID, Flatten(record)))
}

// Flatten returns a record as one JSON object with its fields next to the
// id, version and timestamps.
func Flatten(record collections.Record) map[string]any {
	result := make(map[string]any, len(record.Data)+4)
	for key, value := range record.Data {
		result[key] = value
	}
	result["id"] = record.ID
	result["version"] = record.Version
	result["created_at"] = record.CreatedAt
	result["updated_at"] = record.UpdatedAt
	return result
}

// formatForForm rewrites stored RFC 3339 dates into the value format of the
// matching HTML input (date or datetime-local).
func (a AdminAPI) formatForForm(typeID string, record map[string]any) map[string]any {
	for _, definition := range a.Types {
		if definition.ID != typeID {
			continue
		}
		if definition.Drafts {
			record[StatusField] = recordStatus(record)
		}
		for _, field := range definition.Fields {
			layout := ""
			switch field.Kind {
			case "date":
				layout = "2006-01-02"
			case "datetime":
				layout = "2006-01-02T15:04"
			}
			if text, ok := record[field.ID].(string); ok && layout != "" {
				if parsed, err := time.Parse(time.RFC3339, text); err == nil {
					record[field.ID] = parsed.UTC().Format(layout)
				}
			}
		}
	}
	return record
}

// toStorage keeps only schema fields and converts form date values
// ("2006-01-02" or "2006-01-02T15:04") to RFC 3339.
func toStorage(schema collections.Schema, payload map[string]any) map[string]any {
	data := make(map[string]any, len(schema.Fields))
	for _, field := range schema.Fields {
		value, ok := payload[field.ID]
		if !ok {
			continue
		}
		if text, isText := value.(string); isText && field.Kind == collections.DateTime && text != "" {
			for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
				if parsed, err := time.Parse(layout, text); err == nil {
					value = parsed.UTC().Format(time.RFC3339)
					break
				}
			}
		}
		data[field.ID] = value
	}
	return data
}

// writeServiceError maps collection errors onto the responses the generated
// admin understands: 404, 409 for a stale version, and 422 with per-field
// codes for validation and uniqueness failures.
func writeServiceError(w http.ResponseWriter, err error, schema collections.Schema, data map[string]any) {
	switch {
	case errors.Is(err, collections.ErrCollectionNotFound), errors.Is(err, collections.ErrRecordNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, collections.ErrInvalidQuery):
		writeError(w, http.StatusBadRequest, "invalid list query")
	case errors.Is(err, collections.ErrVersionConflict):
		writeError(w, http.StatusConflict, "the record was changed by someone else")
	case errors.Is(err, collections.ErrUniqueValue):
		fields := map[string]string{}
		for _, field := range schema.Fields {
			if _, present := data[field.ID]; field.Unique && present {
				fields[field.ID] = "taken"
			}
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "validation_failed", "fields": fields})
	case errors.Is(err, collections.ErrInvalidRecord):
		fields := map[string]string{}
		if _, id, found := strings.Cut(err.Error(), ": "); found {
			code := "invalid"
			if value, present := data[id]; !present || value == nil || value == "" {
				code = "required"
			}
			fields[id] = code
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "validation_failed", "fields": fields})
	default:
		writeError(w, http.StatusInternalServerError, "request could not be completed")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

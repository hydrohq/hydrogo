package transport

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
)

type Handler struct{ Service collections.Service }

//go:embed assets/schema-builder.js
var schemaBuilderJS []byte

func (h Handler) Routes() []router.Route {
	admin := router.Protected
	return []router.Route{
		{Pattern: "GET /admin/collections", Access: admin, Handler: http.HandlerFunc(h.adminIndex)},
		{Pattern: "POST /admin/collections", Access: admin, Handler: http.HandlerFunc(h.adminCreateSchema)},
		{Pattern: "GET /admin/collections/schema-builder.js", Access: admin, Handler: http.HandlerFunc(serveSchemaBuilder)},
		{Pattern: "POST /admin/collections/{collection}/delete", Access: admin, Handler: http.HandlerFunc(h.adminDeleteSchema)},
		{Pattern: "GET /admin/collections/{collection}", Access: admin, Handler: http.HandlerFunc(h.adminRecords)},
		{Pattern: "POST /admin/collections/{collection}/records", Access: admin, Handler: http.HandlerFunc(h.adminCreateRecord)},
		{Pattern: "GET /admin/collections/{collection}/records/{id}", Access: admin, Handler: http.HandlerFunc(h.adminEditRecord)},
		{Pattern: "POST /admin/collections/{collection}/records/{id}", Access: admin, Handler: http.HandlerFunc(h.adminUpdateRecord)},
		{Pattern: "POST /admin/collections/{collection}/records/{id}/delete", Access: admin, Handler: http.HandlerFunc(h.adminDeleteRecord)},
		{Pattern: "GET /api/v1/collections", Access: admin, Handler: http.HandlerFunc(h.apiSchemas)},
		{Pattern: "GET /api/v1/collections/{collection}", Access: admin, Handler: http.HandlerFunc(h.apiSchema)},
		{Pattern: "DELETE /api/v1/collections/{collection}", Access: admin, Handler: http.HandlerFunc(h.apiDeleteSchema)},
		{Pattern: "GET /api/v1/collections/{collection}/records", Access: admin, Handler: http.HandlerFunc(h.apiRecords)},
		{Pattern: "POST /api/v1/collections/{collection}/records", Access: admin, Handler: http.HandlerFunc(h.apiCreateRecord)},
		{Pattern: "GET /api/v1/collections/{collection}/records/{id}", Access: admin, Handler: http.HandlerFunc(h.apiGetRecord)},
		{Pattern: "PUT /api/v1/collections/{collection}/records/{id}", Access: admin, Handler: http.HandlerFunc(h.apiUpdateRecord)},
		{Pattern: "DELETE /api/v1/collections/{collection}/records/{id}", Access: admin, Handler: http.HandlerFunc(h.apiDeleteRecord)},
	}
}

func (h Handler) adminIndex(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	schemas, err := h.Service.ListSchemas(r.Context())
	if err != nil {
		http.Error(w, "collections unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Collections", "Schema builder", security.NonceFromContext(r.Context()), "collections") != nil {
		return
	}
	if err := ui.CollectionStudio(schemas, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		http.Error(w, "collection studio unavailable", http.StatusInternalServerError)
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) adminCreateSchema(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid schema", http.StatusBadRequest)
		return
	}
	schema := collections.Schema{ID: strings.TrimSpace(r.FormValue("id")), Label: strings.TrimSpace(r.FormValue("label")), Description: strings.TrimSpace(r.FormValue("description"))}
	for i := 0; i < collections.MaxFields; i++ {
		id := strings.TrimSpace(r.FormValue(field("id", i)))
		label := strings.TrimSpace(r.FormValue(field("label", i)))
		if id == "" && label == "" {
			continue
		}
		kind := collections.FieldKind(r.FormValue(field("kind", i)))
		setting := strings.TrimSpace(r.FormValue(field("options", i)))
		entry := collections.Field{ID: id, Label: label, Kind: kind, Required: r.FormValue(field("required", i)) == "true", Unique: r.FormValue(field("unique", i)) == "true"}
		var err error
		entry.Min, err = parseOptionalBound(r.FormValue(field("min", i)))
		if err != nil {
			http.Error(w, "invalid field limits", http.StatusBadRequest)
			return
		}
		entry.Max, err = parseOptionalBound(r.FormValue(field("max", i)))
		if err != nil {
			http.Error(w, "invalid field limits", http.StatusBadRequest)
			return
		}
		if kind == collections.Select || kind == collections.MultiSelect {
			for _, option := range strings.Split(setting, ",") {
				if option = strings.TrimSpace(option); option != "" {
					entry.Options = append(entry.Options, option)
				}
			}
		} else if kind == collections.Relation || kind == collections.RelationMany {
			entry.Relation = setting
		}
		schema.Fields = append(schema.Fields, entry)
	}
	if err := h.Service.CreateSchema(r.Context(), schema); err != nil {
		http.Error(w, "collection schema could not be saved", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/collections/"+url.PathEscape(schema.ID), http.StatusSeeOther)
}

func parseOptionalBound(value string) (*float64, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func serveSchemaBuilder(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(schemaBuilderJS)
}

func (h Handler) adminDeleteSchema(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid confirmation", http.StatusBadRequest)
		return
	}
	id := r.PathValue("collection")
	if r.FormValue("confirm_schema") != id {
		http.Error(w, "schema deletion must be confirmed", http.StatusBadRequest)
		return
	}
	if err := h.Service.DeleteSchema(r.Context(), id); err != nil {
		http.Error(w, "schema could not be deleted", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/admin/collections", http.StatusSeeOther)
}

func (h Handler) apiDeleteSchema(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.Service.DeleteSchema(r.Context(), r.PathValue("collection")); err != nil {
		apiError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) adminRecords(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := r.PathValue("collection")
	schema, err := h.Service.GetSchema(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	records, err := h.Service.ListRecords(r.Context(), id, 100, 0)
	if err != nil {
		http.Error(w, "records unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, schema.Label, "Collection records", security.NonceFromContext(r.Context()), "collections") != nil {
		return
	}
	if err := ui.CollectionRecordsPage(schema, records, security.CSRFTokenFromContext(r.Context()), ui.CollectionRecordFields(schema, nil), "/admin/collections/"+url.PathEscape(id)+"/records").Render(r.Context(), w); err != nil {
		http.Error(w, "collection records unavailable", http.StatusInternalServerError)
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) adminCreateRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, collections.MaxRecordBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid record", http.StatusBadRequest)
		return
	}
	data, err := formData(h, r)
	if err != nil {
		http.Error(w, "invalid record", http.StatusBadRequest)
		return
	}
	if _, err := h.Service.CreateRecord(r.Context(), r.PathValue("collection"), data); err != nil {
		http.Error(w, "record does not match this collection", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/collections/"+url.PathEscape(r.PathValue("collection")), http.StatusSeeOther)
}

func (h Handler) adminEditRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	schema, err := h.Service.GetSchema(r.Context(), r.PathValue("collection"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	record, err := h.Service.GetRecord(r.Context(), schema.ID, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ui.AdminStartForRequest(w, r, "Edit record", schema.Label, security.NonceFromContext(r.Context()), "collections") != nil {
		return
	}
	action := "/admin/collections/" + url.PathEscape(schema.ID) + "/records/" + url.PathEscape(record.ID)
	if err := ui.CollectionRecordEdit(schema, record, ui.CollectionRecordFields(schema, record.Data), action, security.CSRFTokenFromContext(r.Context())).Render(r.Context(), w); err != nil {
		http.Error(w, "record editor unavailable", http.StatusInternalServerError)
		return
	}
	_ = ui.AdminEnd(w)
}

func (h Handler) adminUpdateRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, collections.MaxRecordBytes)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid record", http.StatusBadRequest)
		return
	}
	data, err := formData(h, r)
	if err != nil {
		http.Error(w, "invalid record", http.StatusBadRequest)
		return
	}
	if _, err := h.Service.UpdateRecord(r.Context(), r.PathValue("collection"), r.PathValue("id"), data); err != nil {
		http.Error(w, "record does not match this collection", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/collections/"+url.PathEscape(r.PathValue("collection")), http.StatusSeeOther)
}

func (h Handler) adminDeleteRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.Service.DeleteRecord(r.Context(), r.PathValue("collection"), r.PathValue("id")); err != nil {
		http.Error(w, "record could not be deleted", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/admin/collections/"+url.PathEscape(r.PathValue("collection")), http.StatusSeeOther)
}

func (h Handler) apiSchemas(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	schemas, err := h.Service.ListSchemas(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, schemas)
}
func (h Handler) apiSchema(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	schema, err := h.Service.GetSchema(r.Context(), r.PathValue("collection"))
	if err != nil {
		apiError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, schema)
}
func (h Handler) apiRecords(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.Service.ListRecords(r.Context(), r.PathValue("collection"), limit, offset)
	if err != nil {
		apiError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"records": items, "limit": limit, "offset": offset})
}
func (h Handler) apiCreateRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	data, err := decodeRecord(w, r)
	if err != nil {
		apiError(w, err)
		return
	}
	record, err := h.Service.CreateRecord(r.Context(), r.PathValue("collection"), data)
	if err != nil {
		apiError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, record)
}
func (h Handler) apiGetRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	record, err := h.Service.GetRecord(r.Context(), r.PathValue("collection"), r.PathValue("id"))
	if err != nil {
		apiError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, record)
}
func (h Handler) apiUpdateRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	data, err := decodeRecord(w, r)
	if err != nil {
		apiError(w, err)
		return
	}
	record, err := h.Service.UpdateRecord(r.Context(), r.PathValue("collection"), r.PathValue("id"), data)
	if err != nil {
		apiError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, record)
}
func (h Handler) apiDeleteRecord(w http.ResponseWriter, r *http.Request) {
	if !admin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.Service.DeleteRecord(r.Context(), r.PathValue("collection"), r.PathValue("id")); err != nil {
		apiError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formData(h Handler, r *http.Request) (map[string]any, error) {
	schema, err := h.Service.GetSchema(r.Context(), r.PathValue("collection"))
	if err != nil {
		return nil, err
	}
	data := make(map[string]any, len(schema.Fields))
	for _, field := range schema.Fields {
		switch field.Kind {
		case collections.Boolean:
			data[field.ID] = r.FormValue(field.ID) == "true" || r.FormValue(field.ID) == "on"
		case collections.MultiSelect:
			items := r.Form[field.ID]
			values := make([]any, 0, len(items))
			for _, item := range items {
				values = append(values, item)
			}
			data[field.ID] = values
		case collections.RelationMany:
			values := make([]any, 0)
			for _, item := range strings.Split(r.FormValue(field.ID), ",") {
				if item = strings.TrimSpace(item); item != "" {
					values = append(values, item)
				}
			}
			data[field.ID] = values
		case collections.Number:
			raw := strings.TrimSpace(r.FormValue(field.ID))
			if raw != "" {
				value, err := strconv.ParseFloat(raw, 64)
				if err != nil {
					return nil, err
				}
				data[field.ID] = value
			}
		default:
			value := strings.TrimSpace(r.FormValue(field.ID))
			if field.Kind == collections.DateTime && value != "" {
				if parsed, err := time.Parse("2006-01-02T15:04", value); err == nil {
					value = parsed.UTC().Format(time.RFC3339)
				}
			}
			if value != "" {
				data[field.ID] = value
			}
		}
	}
	return data, nil
}
func decodeRecord(w http.ResponseWriter, r *http.Request) (map[string]any, error) {
	r.Body = http.MaxBytesReader(w, r.Body, collections.MaxRecordBytes)
	var data map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}
func admin(r *http.Request) bool {
	user := auth.UserFromContext(r.Context())
	return user != nil && user.Active && user.Role == auth.RoleAdmin
}
func field(prefix string, index int) string { return fmt.Sprintf("field_%s_%d", prefix, index) }
func apiError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, collections.ErrCollectionNotFound) || errors.Is(err, collections.ErrRecordNotFound) {
		status = http.StatusNotFound
	}
	if errors.Is(err, collections.ErrUniqueValue) {
		status = http.StatusConflict
	}
	message := "collection request could not be completed"
	if status == http.StatusNotFound {
		message = "collection or record not found"
	}
	if status == http.StatusConflict {
		message = "a field value conflicts with an existing record"
	}
	api.WriteJSON(w, status, map[string]string{"error": message})
}
func _unused(v any) string { return fmt.Sprint(v) }

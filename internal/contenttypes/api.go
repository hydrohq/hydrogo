package contenttypes

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
)

const maxBodyBytes = collections.MaxRecordBytes + 4096

// AdminAPI is the admin JSON API of the YAML content types, mounted under
// /api/v1/admin/content/{type} (see admin_http.go) and backed by
// collections.Service.
type AdminAPI struct {
	Service collections.Service
	Types   []Definition
	// TypesFunc optionally supplies the definitions dynamically.
	TypesFunc func() []Definition
	// Actor names the user making a change (stored with revisions).
	Actor func(*http.Request) string
	// Locales are the site's content locales (see localization.go).
	Locales Locales
	// Can reports whether the request's user holds a named permission, used by
	// state transitions that need one. Nil denies every permissioned step.
	Can func(r *http.Request, permission string) bool
	// Authorize reports whether the request may read (write false) or change
	// (write true) content. Nil denies every request.
	Authorize func(r *http.Request, write bool) bool
	// Subject names the signed-in user as record owners are written (their
	// e-mail address); used by Definition.Access.
	Subject func(r *http.Request) string
	// Hook calls a configured hook service for Definition.Hooks; nil means
	// none is configured.
	Hook HookCaller

	// hooksDone is set on the copy that runs a batch, whose hooks were
	// consulted before the transaction began.
	hooksDone bool
	// Presence tracks active editors in process memory.
	Presence *realtime.Presence
}

func (a AdminAPI) types() []Definition {
	if a.TypesFunc != nil {
		return a.TypesFunc()
	}
	return a.Types
}

// withActor attaches the acting user to the request context for writes.
func (a AdminAPI) withActor(r *http.Request) *http.Request {
	if a.Actor == nil {
		return r
	}
	return r.WithContext(collections.WithActor(r.Context(), a.Actor(r)))
}

// save creates (empty id) or updates a record from a JSON payload. A drafts
// type's status is kept server-side: new records are drafts and updates keep
// the stored status, whatever the payload says.
func (a AdminAPI) save(r *http.Request, schema collections.Schema, id string, payload map[string]any) (map[string]any, error) {
	r = a.withActor(r)
	expectedVersion := 0
	if version, ok := payload["version"].(float64); ok {
		expectedVersion = int(version)
	}
	if definition, ok := definitionByID(a.types(), schema.ID); ok {
		if err := a.scopePayload(r, definition, id, payload); err != nil {
			return nil, err
		}
		if err := a.keepHidden(r, definition, id, payload); err != nil {
			return nil, err
		}
		if !a.hooksDone {
			on := "create"
			if id != "" {
				on = "update"
			}
			if err := a.runHooks(r, definition, HookEvent{On: on, ID: id, Fields: toStorage(schema, payload)}); err != nil {
				return nil, err
			}
		}
	}
	data := toStorage(schema, payload)
	if err := a.keepStatus(r, schema.ID, id, data); err != nil {
		return nil, err
	}
	var record collections.Record
	var err error
	if id == "" {
		record, err = a.Service.CreateRecord(r.Context(), schema.ID, data)
	} else {
		record, err = a.Service.UpdateRecordVersion(r.Context(), schema.ID, id, data, expectedVersion)
	}
	if err != nil {
		return nil, &saveError{err: err, data: data}
	}
	return a.present(r, schema.ID, Flatten(record)), nil
}

// saveError carries the submitted data so validation failures can name the
// offending fields.
type saveError struct {
	err  error
	data map[string]any
}

func (e *saveError) Error() string { return e.err.Error() }
func (e *saveError) Unwrap() error { return e.err }

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

// present rewrites stored RFC 3339 dates into the value format of the
// matching input (date or datetime-local) and adds the status of a drafts
// type, so a record reads back exactly as a client submits it.
func (a AdminAPI) present(r *http.Request, typeID string, record map[string]any) map[string]any {
	for _, definition := range a.types() {
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
		for _, field := range a.hiddenFields(r, definition) {
			delete(record, field.ID)
		}
	}
	return record
}

// toStorage keeps only schema fields and converts date values
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

// writeServiceError maps collection errors onto API responses: 404 for
// unknown types, records, revisions and locales; 400 for bad queries and
// transitions; 409 for a stale version; 422 with per-field codes for
// validation and uniqueness failures.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error, schema collections.Schema) {
	status, message, fields := serviceFailure(err, schema)
	api.WriteError(w, r, status, message, fields)
}

// serviceFailure maps a service error to the HTTP status, message and field
// codes of its error response.
func serviceFailure(err error, schema collections.Schema) (int, string, map[string]string) {
	var data map[string]any
	var submitted *saveError
	if errors.As(err, &submitted) {
		data = submitted.data
	}
	var rejected *hookError
	if errors.As(err, &rejected) {
		return rejected.status, rejected.message, nil
	}
	switch {
	case errors.Is(err, collections.ErrCollectionNotFound), errors.Is(err, collections.ErrRecordNotFound),
		errors.Is(err, collections.ErrRevisionNotFound), errors.Is(err, collections.ErrNotLocalized),
		errors.Is(err, errNoRevisions), errors.Is(err, errLocale):
		return http.StatusNotFound, "not found", nil
	case errors.Is(err, errBatchForbidden):
		return http.StatusForbidden, "forbidden", nil
	case errors.Is(err, collections.ErrRuleViolated):
		_, text, _ := strings.Cut(err.Error(), ": ")
		return http.StatusUnprocessableEntity, text, map[string]string{"_rule": text}
	case errors.Is(err, collections.ErrTransitionForbidden):
		return http.StatusForbidden, "forbidden", nil
	case errors.Is(err, collections.ErrTransitionNotAllowed):
		_, text, _ := strings.Cut(err.Error(), ": ")
		return http.StatusUnprocessableEntity, "transition not allowed: " + text, map[string]string{"_state": text}
	case errors.Is(err, collections.ErrNoStates):
		return http.StatusNotFound, "not found", nil
	case errors.Is(err, errBatchOperation):
		return http.StatusBadRequest, "invalid operation", nil
	case errors.Is(err, collections.ErrInvalidQuery):
		return http.StatusBadRequest, "invalid list query", nil
	case errors.Is(err, collections.ErrInvalidLocale):
		return http.StatusBadRequest, "unsupported locale", nil
	case errors.Is(err, errTransition):
		return http.StatusBadRequest, "transition not allowed", nil
	case errors.Is(err, collections.ErrVersionConflict):
		return http.StatusConflict, "the record was changed by someone else", nil
	case errors.Is(err, collections.ErrUniqueValue):
		fields := map[string]string{}
		for _, field := range schema.Fields {
			if _, present := data[field.ID]; field.Unique && present {
				fields[field.ID] = "taken"
			}
		}
		return http.StatusUnprocessableEntity, "validation failed", fields
	case errors.Is(err, collections.ErrInvalidRecord):
		fields := map[string]string{}
		if _, id, found := strings.Cut(err.Error(), ": "); found {
			code := "invalid"
			if value, present := data[id]; !present || value == nil || value == "" {
				code = "required"
			}
			fields[id] = code
		}
		return http.StatusUnprocessableEntity, "validation failed", fields
	default:
		return http.StatusInternalServerError, "request could not be completed", nil
	}
}

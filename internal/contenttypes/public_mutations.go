package contenttypes

import (
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/api"
)

const maxPublicBodyBytes = 1 << 20 // 1MB

func (p PublicAPI) Create(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	principal := p.principal(r)
	rule := ""
	if definition.AccessRules != nil {
		rule = definition.AccessRules.Create
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPublicBodyBytes)
	var payload map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", nil)
		return
	}
	if payload == nil {
		payload = make(map[string]any)
	}

	allowed, err := EvaluateRule(rule, principal, payload, false)
	if err != nil || !allowed {
		if principal.Anonymous {
			api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
		} else {
			api.WriteError(w, r, http.StatusForbidden, "create not allowed by access rules", nil)
		}
		return
	}

	// If drafts is enabled and caller didn't specify status, default to draft
	if definition.Drafts {
		if _, ok := payload[StatusField]; !ok {
			payload[StatusField] = StatusDraft
		}
	}

	record, err := p.Service.CreateRecord(r.Context(), typeID, payload)
	if err != nil {
		schema, _ := definition.Schema()
		writeServiceError(w, r, err, schema)
		return
	}

	if definition.Realtime && p.Hub != nil {
		p.dispatchRealtime(r.Context(), typeID, "create", record)
	}

	api.WriteData(w, http.StatusCreated, p.present(typeID, "", record), nil)
}

func (p PublicAPI) Update(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	id := r.PathValue("id")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	principal := p.principal(r)
	rule := ""
	if definition.AccessRules != nil {
		rule = definition.AccessRules.Update
	}

	stored, err := p.Service.GetRecord(r.Context(), typeID, id)
	if err != nil {
		schema, _ := definition.Schema()
		writeServiceError(w, r, err, schema)
		return
	}

	// Check update rule on existing stored record
	allowed, err := EvaluateRule(rule, principal, stored.Data, false)
	if err != nil || !allowed {
		if principal.Anonymous {
			api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
		} else {
			api.WriteError(w, r, http.StatusForbidden, "update not allowed by access rules", nil)
		}
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPublicBodyBytes)
	var payload map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", nil)
		return
	}
	if payload == nil {
		payload = make(map[string]any)
	}

	// Check update rule on resulting proposed state
	merged := make(map[string]any, len(stored.Data)+len(payload))
	for k, v := range stored.Data {
		merged[k] = v
	}
	for k, v := range payload {
		merged[k] = v
	}
	allowed, err = EvaluateRule(rule, principal, merged, false)
	if err != nil || !allowed {
		api.WriteError(w, r, http.StatusForbidden, "updated record violates access rules", nil)
		return
	}

	record, err := p.Service.UpdateRecord(r.Context(), typeID, id, payload)
	if err != nil {
		schema, _ := definition.Schema()
		writeServiceError(w, r, err, schema)
		return
	}

	if definition.Realtime && p.Hub != nil {
		p.dispatchRealtime(r.Context(), typeID, "update", record)
	}

	api.WriteData(w, http.StatusOK, p.present(typeID, "", record), nil)
}

func (p PublicAPI) Delete(w http.ResponseWriter, r *http.Request) {
	typeID := r.PathValue("type")
	id := r.PathValue("id")
	definition, ok := p.definition(typeID)
	if !ok {
		api.WriteError(w, r, http.StatusNotFound, "not found", nil)
		return
	}
	principal := p.principal(r)
	rule := ""
	if definition.AccessRules != nil {
		rule = definition.AccessRules.Delete
	}

	stored, err := p.Service.GetRecord(r.Context(), typeID, id)
	if err != nil {
		schema, _ := definition.Schema()
		writeServiceError(w, r, err, schema)
		return
	}

	allowed, err := EvaluateRule(rule, principal, stored.Data, false)
	if err != nil || !allowed {
		if principal.Anonymous {
			api.WriteError(w, r, http.StatusUnauthorized, "authentication required", nil)
		} else {
			api.WriteError(w, r, http.StatusForbidden, "delete not allowed by access rules", nil)
		}
		return
	}

	if err := p.Service.DeleteRecord(r.Context(), typeID, id); err != nil {
		schema, _ := definition.Schema()
		writeServiceError(w, r, err, schema)
		return
	}

	if definition.Realtime && p.Hub != nil {
		p.dispatchRealtime(r.Context(), typeID, "delete", stored)
	}

	api.WriteData(w, http.StatusOK, map[string]any{"id": id, "deleted": true}, nil)
}

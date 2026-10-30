package contenttypes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

type PublicBatchOperation struct {
	Op      string         `json:"op"`
	Action  string         `json:"action"` // alias for op
	Type    string         `json:"type"`
	ID      string         `json:"id"`
	Fields  map[string]any `json:"fields"`
	Data    map[string]any `json:"data"` // alias for fields
	Version int            `json:"version"`
}

type PublicBatchRequest struct {
	Atomic     *bool                  `json:"atomic"`
	Operations []PublicBatchOperation `json:"operations"`
}

type PublicBatchItemResult struct {
	Index  int            `json:"index"`
	Action string         `json:"action"`
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Status int            `json:"status"`
	Record map[string]any `json:"record,omitempty"`
	Error  string         `json:"error,omitempty"`
}

func (p PublicAPI) Batch(w http.ResponseWriter, r *http.Request) {
	p = p.forRequest(r)
	defaultType := r.PathValue("type")
	r.Body = http.MaxBytesReader(w, r.Body, maxBatchBodyBytes)

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", nil)
		return
	}

	var req PublicBatchRequest
	trimmed := strings.TrimSpace(string(raw))
	if strings.HasPrefix(trimmed, "[") {
		var ops []PublicBatchOperation
		if err := json.Unmarshal(raw, &ops); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid batch operations array", nil)
			return
		}
		req.Operations = ops
	} else {
		if err := json.Unmarshal(raw, &req); err != nil {
			api.WriteError(w, r, http.StatusBadRequest, "invalid batch request object", nil)
			return
		}
	}

	if len(req.Operations) == 0 || len(req.Operations) > MaxBatchOperations {
		api.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("a batch needs 1 to %d operations", MaxBatchOperations), nil)
		return
	}

	isAtomic := true
	if req.Atomic != nil {
		isAtomic = *req.Atomic
	}

	principal := p.principal(r)

	for i := range req.Operations {
		op := &req.Operations[i]
		if op.Op == "" && op.Action != "" {
			op.Op = op.Action
		}
		op.Op = strings.ToLower(strings.TrimSpace(op.Op))
		if op.Op != "create" && op.Op != "update" && op.Op != "delete" {
			api.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("operations[%d].action must be create, update or delete", i), nil)
			return
		}
		if op.Type == "" {
			op.Type = defaultType
		}
		if op.Type == "" {
			api.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("operations[%d].type is required", i), nil)
			return
		}
		if op.Fields == nil && op.Data != nil {
			op.Fields = op.Data
		}
		if op.Fields == nil {
			op.Fields = make(map[string]any)
		}
		if (op.Op == "update" || op.Op == "delete") && strings.TrimSpace(op.ID) == "" {
			api.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("operations[%d].id is required for %s", i, op.Op), nil)
			return
		}
		if _, ok := p.definition(op.Type); !ok {
			api.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("content type %q not found", op.Type), nil)
			return
		}
	}

	executeOp := func(svc collections.Service, index int, op PublicBatchOperation) (PublicBatchItemResult, error) {
		def, _ := p.definition(op.Type)
		switch op.Op {
		case "create":
			rule := ""
			if def.AccessRules != nil {
				rule = def.AccessRules.Create
			}
			allowed, err := EvaluateRule(rule, principal, op.Fields, false)
			if err != nil || !allowed {
				return PublicBatchItemResult{Index: index, Action: "create", Type: op.Type, Status: http.StatusForbidden, Error: "create not allowed by access rules"},
					fmt.Errorf("operations[%d]: create not allowed by access rules", index)
			}
			if def.Drafts {
				if _, ok := op.Fields[StatusField]; !ok {
					op.Fields[StatusField] = StatusDraft
				}
			}
			record, err := svc.CreateRecord(r.Context(), op.Type, op.Fields)
			if err != nil {
				return PublicBatchItemResult{Index: index, Action: "create", Type: op.Type, Status: http.StatusUnprocessableEntity, Error: err.Error()},
					fmt.Errorf("operations[%d]: %w", index, err)
			}
			if def.Realtime && p.Hub != nil {
				p.dispatchRealtime(r.Context(), op.Type, "create", record)
			}
			return PublicBatchItemResult{Index: index, Action: "create", Type: op.Type, ID: record.ID, Status: http.StatusCreated, Record: p.present(op.Type, "", record)}, nil

		case "update":
			rule := ""
			if def.AccessRules != nil {
				rule = def.AccessRules.Update
			}
			stored, err := svc.GetRecord(r.Context(), op.Type, op.ID)
			if err != nil {
				return PublicBatchItemResult{Index: index, Action: "update", Type: op.Type, ID: op.ID, Status: http.StatusNotFound, Error: err.Error()},
					fmt.Errorf("operations[%d]: %w", index, err)
			}
			allowed, err := EvaluateRule(rule, principal, stored.Data, false)
			if err != nil || !allowed {
				return PublicBatchItemResult{Index: index, Action: "update", Type: op.Type, ID: op.ID, Status: http.StatusForbidden, Error: "update not allowed by access rules"},
					fmt.Errorf("operations[%d]: update not allowed by access rules", index)
			}
			merged := make(map[string]any, len(stored.Data)+len(op.Fields))
			for k, v := range stored.Data {
				merged[k] = v
			}
			for k, v := range op.Fields {
				merged[k] = v
			}
			allowed, err = EvaluateRule(rule, principal, merged, false)
			if err != nil || !allowed {
				return PublicBatchItemResult{Index: index, Action: "update", Type: op.Type, ID: op.ID, Status: http.StatusForbidden, Error: "updated record violates access rules"},
					fmt.Errorf("operations[%d]: updated record violates access rules", index)
			}
			record, err := svc.UpdateRecord(r.Context(), op.Type, op.ID, op.Fields)
			if err != nil {
				return PublicBatchItemResult{Index: index, Action: "update", Type: op.Type, ID: op.ID, Status: http.StatusUnprocessableEntity, Error: err.Error()},
					fmt.Errorf("operations[%d]: %w", index, err)
			}
			if def.Realtime && p.Hub != nil {
				p.dispatchRealtime(r.Context(), op.Type, "update", record)
			}
			return PublicBatchItemResult{Index: index, Action: "update", Type: op.Type, ID: record.ID, Status: http.StatusOK, Record: p.present(op.Type, "", record)}, nil

		case "delete":
			rule := ""
			if def.AccessRules != nil {
				rule = def.AccessRules.Delete
			}
			stored, err := svc.GetRecord(r.Context(), op.Type, op.ID)
			if err != nil {
				return PublicBatchItemResult{Index: index, Action: "delete", Type: op.Type, ID: op.ID, Status: http.StatusNotFound, Error: err.Error()},
					fmt.Errorf("operations[%d]: %w", index, err)
			}
			allowed, err := EvaluateRule(rule, principal, stored.Data, false)
			if err != nil || !allowed {
				return PublicBatchItemResult{Index: index, Action: "delete", Type: op.Type, ID: op.ID, Status: http.StatusForbidden, Error: "delete not allowed by access rules"},
					fmt.Errorf("operations[%d]: delete not allowed by access rules", index)
			}
			if err := svc.DeleteRecord(r.Context(), op.Type, op.ID); err != nil {
				return PublicBatchItemResult{Index: index, Action: "delete", Type: op.Type, ID: op.ID, Status: http.StatusUnprocessableEntity, Error: err.Error()},
					fmt.Errorf("operations[%d]: %w", index, err)
			}
			if def.Realtime && p.Hub != nil {
				p.dispatchRealtime(r.Context(), op.Type, "delete", stored)
			}
			return PublicBatchItemResult{Index: index, Action: "delete", Type: op.Type, ID: op.ID, Status: http.StatusOK}, nil
		}
		return PublicBatchItemResult{}, fmt.Errorf("unknown operation")
	}

	results := make([]PublicBatchItemResult, 0, len(req.Operations))

	if isAtomic {
		err := p.Service.InTransaction(r.Context(), func(tx collections.Service) error {
			for i, op := range req.Operations {
				res, err := executeOp(tx, i, op)
				if err != nil {
					return err
				}
				results = append(results, res)
			}
			return nil
		})
		if err != nil {
			api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), nil)
			return
		}
	} else {
		for i, op := range req.Operations {
			res, _ := executeOp(p.Service, i, op)
			results = append(results, res)
		}
	}

	api.WriteData(w, http.StatusOK, map[string]any{
		"results": results,
		"count":   len(results),
	}, nil)
}

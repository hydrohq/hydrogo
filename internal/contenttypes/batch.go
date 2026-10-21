package contenttypes

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

const (
	// MaxBatchOperations bounds one batch so a request cannot hold a
	// transaction open for long.
	MaxBatchOperations = 200
	maxBatchBodyBytes  = 1 << 20
	refPrefix          = "$ref:"
)

// BatchPath is the route of the atomic batch endpoint.
const BatchPath = AdminAPIPath + "/_batch"

type batchOperation struct {
	Op      string         `json:"op"`
	Type    string         `json:"type"`
	ID      string         `json:"id"`
	Version int            `json:"version"`
	Fields  map[string]any `json:"fields"`
}

type batchRequest struct {
	Operations []batchOperation `json:"operations"`
}

type batchResult struct {
	Index  int            `json:"index"`
	Op     string         `json:"op"`
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Record map[string]any `json:"record,omitempty"`
}

// batch applies create, update and delete operations in one transaction.
// Values of the form "$ref:N" (in fields or id) stand for the id of the record
// created by operation N, so a batch can create a company and its contacts in
// one request. Any failure rolls everything back and names the failing
// operation: fields are reported as "operations[i].<field>".
//
//	POST /api/v1/admin/content/_batch
//	{"operations":[{"op":"create","type":"companies","fields":{"name":"Acme"}},
//	               {"op":"create","type":"contacts","fields":{"full_name":"Al","company":"$ref:0"}}]}
func (a AdminAPI) batch(w http.ResponseWriter, r *http.Request) {
	if a.Authorize == nil || !a.Authorize(r, true) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	var input batchRequest
	if !api.DecodeJSON(w, r, &input, maxBatchBodyBytes) {
		return
	}
	if len(input.Operations) == 0 || len(input.Operations) > MaxBatchOperations {
		api.WriteError(w, r, http.StatusBadRequest, fmt.Sprintf("a batch needs 1 to %d operations", MaxBatchOperations), nil)
		return
	}
	r = a.withActor(r)
	hookRequest, cancelHooks := a.hookBudget(r)
	defer cancelHooks()
	for index, operation := range input.Operations {
		if definition, ok := definitionByID(a.types(), operation.Type); ok && len(definition.Hooks) > 0 && a.allowed(r, definition, true, true) {
			on := operation.Op
			if err := a.runHooks(hookRequest, definition, HookEvent{On: on, ID: operation.ID, Fields: operation.Fields}); err != nil {
				status, message, _ := serviceFailure(err, collections.Schema{})
				api.WriteError(w, r, status, fmt.Sprintf("operation %d failed: %s", index, message), map[string]string{fmt.Sprintf("operations[%d]", index): message})
				return
			}
		}
	}
	results := make([]batchResult, 0, len(input.Operations))
	err := a.Service.InTransaction(r.Context(), func(tx collections.Service) error {
		inTx := a
		inTx.hooksDone = true
		inTx.Service = tx
		for index, operation := range input.Operations {
			result, err := inTx.runBatchOperation(r, index, operation, results)
			if err != nil {
				return &batchError{index: index, err: err, schema: inTx.schemaOf(r, operation.Type)}
			}
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		var failed *batchError
		switch {
		case errors.As(err, &failed):
			status, message, fields := serviceFailure(failed.err, failed.schema)
			prefixed := map[string]string{}
			for field, code := range fields {
				prefixed[fmt.Sprintf("operations[%d].%s", failed.index, field)] = code
			}
			if len(prefixed) == 0 {
				prefixed[fmt.Sprintf("operations[%d]", failed.index)] = message
			}
			api.WriteError(w, r, status, fmt.Sprintf("operation %d failed: %s", failed.index, message), prefixed)
		case errors.Is(err, collections.ErrNoTransactions):
			api.WriteError(w, r, http.StatusNotImplemented, "batch is not available with this storage", nil)
		default:
			api.WriteError(w, r, http.StatusInternalServerError, "request could not be completed", nil)
		}
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"results": results}, nil)
}

type batchError struct {
	index  int
	err    error
	schema collections.Schema
}

func (e *batchError) Error() string { return fmt.Sprintf("operation %d: %v", e.index, e.err) }
func (e *batchError) Unwrap() error { return e.err }

func (a AdminAPI) schemaOf(r *http.Request, typeID string) collections.Schema {
	schema, _ := a.Service.GetSchema(r.Context(), typeID)
	return schema
}

var errBatchForbidden = errors.New("not allowed to change this type")

var errBatchOperation = errors.New("invalid batch operation")

func (a AdminAPI) runBatchOperation(r *http.Request, index int, operation batchOperation, done []batchResult) (batchResult, error) {
	definition, ok := definitionByID(a.types(), operation.Type)
	if !ok {
		return batchResult{}, collections.ErrCollectionNotFound
	}
	if !a.allowed(r, definition, true, true) {
		return batchResult{}, errBatchForbidden
	}
	id, err := resolveRef(operation.ID, done)
	if err != nil {
		return batchResult{}, err
	}
	result := batchResult{Index: index, Op: operation.Op, Type: operation.Type}
	switch operation.Op {
	case "delete":
		if err := a.guardPeriod(r, definition, id, nil); err != nil {
			return batchResult{}, err
		}
		if subject, restricted := a.restriction(r, definition); restricted && !a.owns(r.Context(), definition, subject, id) {
			return batchResult{}, collections.ErrRecordNotFound
		}
		if err := a.Service.DeleteRecord(r.Context(), operation.Type, id); err != nil {
			return batchResult{}, err
		}
		result.ID = id
		return result, nil
	case "create", "update":
		if (operation.Op == "create") != (id == "") {
			return batchResult{}, errBatchOperation
		}
		if operation.Fields == nil {
			return batchResult{}, errBatchOperation
		}
		payload := make(map[string]any, len(operation.Fields)+1)
		for field, value := range operation.Fields {
			resolved, err := resolveValue(value, done)
			if err != nil {
				return batchResult{}, err
			}
			payload[field] = resolved
		}
		if operation.Op == "update" {
			payload["version"] = float64(operation.Version)
		}
		schema, err := a.Service.GetSchema(r.Context(), operation.Type)
		if err != nil {
			return batchResult{}, err
		}
		record, err := a.save(r, schema, id, payload)
		if err != nil {
			return batchResult{}, err
		}
		result.Record = record
		result.ID, _ = record["id"].(string)
		return result, nil
	default:
		return batchResult{}, errBatchOperation
	}
}

// resolveRef replaces a "$ref:N" string with the id created by operation N.
// N must be an earlier operation that created a record.
func resolveRef(value string, done []batchResult) (string, error) {
	rest, ok := strings.CutPrefix(value, refPrefix)
	if !ok {
		return value, nil
	}
	index, err := strconv.Atoi(rest)
	if err != nil || index < 0 || index >= len(done) || done[index].Op != "create" {
		return "", fmt.Errorf("%w: %s", collections.ErrInvalidRecord, "$ref")
	}
	return done[index].ID, nil
}

func resolveValue(value any, done []batchResult) (any, error) {
	switch typed := value.(type) {
	case string:
		return resolveRef(typed, done)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			resolved, err := resolveValue(item, done)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	default:
		return value, nil
	}
}

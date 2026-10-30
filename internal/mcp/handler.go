package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/openapi"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"gopkg.in/yaml.v3"
)

// Config holds dependencies for the MCP server handler.
type Config struct {
	CollectionService      collections.Service
	ContentTypesDir        string
	RequireTenantScope     bool
	RequireAuth            bool
	ContentTypes           func() []contenttypes.Definition
	ContentTypesForContext func(ctx context.Context) []contenttypes.Definition
	SyncSchema             func(ctx context.Context, def contenttypes.Definition) error
	AuditStore             audit.DetailRecorder
	DB                     database.Executor
}

func (h *Handler) contentTypes(ctx context.Context) []contenttypes.Definition {
	if h.cfg.ContentTypesForContext != nil {
		return h.cfg.ContentTypesForContext(ctx)
	}
	if h.cfg.ContentTypes != nil {
		return h.cfg.ContentTypes()
	}
	return nil
}

// Handler handles Model Context Protocol requests over SSE and JSON-RPC 2.0.
type Handler struct {
	cfg Config
}

// NewHandler constructs a new MCP Handler.
func NewHandler(cfg Config) *Handler {
	return &Handler{cfg: cfg}
}

// ServeHTTP handles both GET (SSE / status) and POST (JSON-RPC) for MCP.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}

		sessionID := newSessionID()
		_, _ = fmt.Fprintf(w, "event: endpoint\ndata: /api/v1/mcp?session_id=%s\n\n", sessionID)
		if ok {
			flusher.Flush()
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name":            "hydrogo-mcp",
		"version":         "1.0.0",
		"protocolVersion": "2024-11-05",
		"status":          "ready",
	})
}

func (h *Handler) handlePost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		writeRPCError(w, nil, -32700, "Parse error: "+err.Error())
		return
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPCError(w, nil, -32700, "Parse error: invalid JSON")
		return
	}

	if req.JSONRPC != "2.0" && req.JSONRPC != "" {
		writeRPCError(w, req.ID, -32600, "Invalid Request: unsupported jsonrpc version")
		return
	}

	ctx := r.Context()
	switch req.Method {
	case "initialize":
		h.handleInitialize(w, req)
	case "notifications/initialized":
		writeRPCResult(w, req.ID, map[string]any{})
	case "ping":
		writeRPCResult(w, req.ID, map[string]any{})
	case "tools/list":
		h.handleToolsList(ctx, w, req)
	case "tools/call":
		h.handleToolsCall(ctx, w, req)
	case "resources/list":
		h.handleResourcesList(ctx, w, req)
	case "resources/read":
		h.handleResourcesRead(ctx, w, req)
	default:
		writeRPCError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (h *Handler) handleInitialize(w http.ResponseWriter, req JSONRPCRequest) {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "HydroGo AI Control Plane",
			"version": "1.0.0",
		},
	}
	writeRPCResult(w, req.ID, result)
}

func (h *Handler) handleToolsList(ctx context.Context, w http.ResponseWriter, req JSONRPCRequest) {
	if err := h.requireScope(ctx, "agent:read"); err != nil {
		writeRPCError(w, req.ID, -32001, err.Error())
		return
	}

	tools := []Tool{
		{
			Name:        "list_content_types",
			Description: "Introspect all active collection schemas, field validations, and declarative access rules.",
			InputSchema: map[string]any{
				"type": "object",
			},
		},
		{
			Name:        "schema_list",
			Description: "Alias for list_content_types. Introspect all active collection schemas.",
			InputSchema: map[string]any{
				"type": "object",
			},
		},
		{
			Name:        "define_content_type",
			Description: "Dynamically provision or update a collection schema in runtime without restarting the server.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":          map[string]any{"type": "string", "description": "Unique singular identifier of the collection, e.g. faq"},
					"name":        map[string]any{"type": "string", "description": "Human-readable singular name, e.g. FAQ"},
					"plural":      map[string]any{"type": "string", "description": "Human-readable plural name, e.g. FAQs"},
					"description": map[string]any{"type": "string", "description": "Brief description of the content type"},
					"public":      map[string]any{"type": "boolean", "description": "Whether the collection is publicly readable at /api/v1/content/<id>"},
					"drafts":      map[string]any{"type": "boolean", "description": "Enable draft / publish workflow"},
					"fields": map[string]any{
						"type":        "array",
						"description": "Field definitions",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":       map[string]any{"type": "string"},
								"name":     map[string]any{"type": "string"},
								"kind":     map[string]any{"type": "string"},
								"required": map[string]any{"type": "boolean"},
							},
							"required": []string{"id", "name", "kind"},
						},
					},
					"definition_yaml": map[string]any{"type": "string", "description": "Raw YAML definition string (alternative to structured fields)"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "schema_apply",
			Description: "Alias for define_content_type. Provision or update collection schema in runtime.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "query_collection",
			Description: "Query collection records with structured filtering, sorting, pagination, and search.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":   map[string]any{"type": "string", "description": "Content type / collection identifier (e.g. faq)"},
					"limit":  map[string]any{"type": "integer", "description": "Number of records to return (1-100, default 20)"},
					"offset": map[string]any{"type": "integer", "description": "Pagination offset"},
					"sort":   map[string]any{"type": "string", "description": "Field sort order, e.g. '-created_at' or 'title'"},
					"search": map[string]any{"type": "string", "description": "Search keyword query"},
					"status": map[string]any{"type": "string", "description": "Filter by status ('published', 'draft', or 'all')"},
				},
				"required": []string{"type"},
			},
		},
		{
			Name:        "content_query",
			Description: "Alias for query_collection. Query collection records.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type": map[string]any{"type": "string"},
				},
				"required": []string{"type"},
			},
		},
		{
			Name:        "mutate_record",
			Description: "Create, update, or delete records in dynamic collections with tamper-evident audit tagging.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":      map[string]any{"type": "string", "description": "Content type identifier"},
					"operation": map[string]any{"type": "string", "enum": []string{"create", "update", "delete"}, "description": "Mutation type"},
					"id":        map[string]any{"type": "string", "description": "Record ID (required for update and delete)"},
					"data":      map[string]any{"type": "object", "description": "Record field payload for create and update"},
				},
				"required": []string{"type", "operation"},
			},
		},
		{
			Name:        "content_mutate",
			Description: "Alias for mutate_record. Mutate records with audit tracking.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":      map[string]any{"type": "string"},
					"operation": map[string]any{"type": "string"},
				},
				"required": []string{"type", "operation"},
			},
		},
	}

	writeRPCResult(w, req.ID, map[string]any{"tools": tools})
}

func (h *Handler) handleToolsCall(ctx context.Context, w http.ResponseWriter, req JSONRPCRequest) {
	var callParams struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &callParams); err != nil {
		writeRPCError(w, req.ID, -32602, "Invalid params: failed to parse arguments")
		return
	}

	var res ToolCallResult
	var err error

	switch callParams.Name {
	case "list_content_types", "schema_list":
		if err = h.requireScope(ctx, "agent:read"); err == nil {
			res, err = h.toolListContentTypes(ctx)
		}
	case "define_content_type", "schema_apply":
		if err = h.requireScope(ctx, "agent:schema"); err == nil {
			res, err = h.toolDefineContentType(ctx, callParams.Arguments)
		}
	case "query_collection", "content_query":
		if err = h.requireScope(ctx, "agent:read"); err == nil {
			res, err = h.toolQueryCollection(ctx, callParams.Arguments)
		}
	case "mutate_record", "content_mutate":
		if err = h.requireScope(ctx, "agent:mutate"); err == nil {
			res, err = h.toolMutateRecord(ctx, callParams.Arguments)
		}
	default:
		writeRPCError(w, req.ID, -32601, fmt.Sprintf("Tool not found: %s", callParams.Name))
		return
	}

	if err != nil {
		writeRPCError(w, req.ID, -32000, err.Error())
		return
	}

	writeRPCResult(w, req.ID, res)
}

func (h *Handler) handleResourcesList(ctx context.Context, w http.ResponseWriter, req JSONRPCRequest) {
	if err := h.requireScope(ctx, "agent:read"); err != nil {
		writeRPCError(w, req.ID, -32001, err.Error())
		return
	}

	resources := []Resource{
		{
			URI:         "hydro://openapi.json",
			Name:        "HydroGo OpenAPI Specification",
			Description: "OpenAPI 3.1 specification for all active collection endpoints",
			MimeType:    "application/json",
		},
		{
			URI:         "hydro://schemas",
			Name:        "Collection Schemas",
			Description: "Full JSON definitions of all configured content types",
			MimeType:    "application/json",
		},
	}

	writeRPCResult(w, req.ID, map[string]any{"resources": resources})
}

func (h *Handler) handleResourcesRead(ctx context.Context, w http.ResponseWriter, req JSONRPCRequest) {
	if err := h.requireScope(ctx, "agent:read"); err != nil {
		writeRPCError(w, req.ID, -32001, err.Error())
		return
	}

	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeRPCError(w, req.ID, -32602, "Invalid params: uri is required")
		return
	}

	var contentText string
	switch params.URI {
	case "hydro://openapi.json":
		types := h.contentTypes(ctx)
		doc, err := openapi.BuildDocument(nil, types)
		if err != nil {
			writeRPCError(w, req.ID, -32000, fmt.Sprintf("failed to build OpenAPI spec: %v", err))
			return
		}
		contentText = string(doc)

	case "hydro://schemas":
		types := h.contentTypes(ctx)
		raw, _ := json.MarshalIndent(types, "", "  ")
		contentText = string(raw)

	default:
		writeRPCError(w, req.ID, -32002, fmt.Sprintf("Resource not found: %s", params.URI))
		return
	}

	writeRPCResult(w, req.ID, map[string]any{
		"contents": []map[string]any{
			{
				"uri":      params.URI,
				"mimeType": "application/json",
				"text":     contentText,
			},
		},
	})
}

// Tool Implementations

func (h *Handler) toolListContentTypes(ctx context.Context) (ToolCallResult, error) {
	types := h.contentTypes(ctx)

	data, err := json.MarshalIndent(types, "", "  ")
	if err != nil {
		return ToolCallResult{IsError: true}, err
	}

	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (h *Handler) toolDefineContentType(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		ID             string               `json:"id"`
		Name           string               `json:"name"`
		Label          string               `json:"label"`
		Description    string               `json:"description"`
		Public         bool                 `json:"public"`
		Drafts         bool                 `json:"drafts"`
		Fields         []contenttypes.Field `json:"fields"`
		DefinitionYAML string               `json:"definition_yaml"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "invalid JSON arguments"}}}, nil
	}

	id := strings.TrimSpace(strings.ToLower(input.ID))
	if id == "" {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "collection id is required"}}}, nil
	}

	var def contenttypes.Definition
	if strings.TrimSpace(input.DefinitionYAML) != "" {
		if err := yaml.Unmarshal([]byte(input.DefinitionYAML), &def); err != nil {
			return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("YAML parse error: %v", err)}}}, nil
		}
	} else {
		lbl := input.Label
		if lbl == "" {
			lbl = input.Name
		}
		def = contenttypes.Definition{
			ID:          id,
			Label:       lbl,
			Description: input.Description,
			Public:      input.Public,
			Drafts:      input.Drafts,
			Fields:      input.Fields,
		}
	}

	if def.ID == "" {
		def.ID = id
	}
	if def.Label == "" {
		def.Label = strings.ToUpper(def.ID[:1]) + def.ID[1:]
	}

	if err := def.Validate(); err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("schema validation failed: %v", err)}}}, nil
	}

	dir := h.cfg.ContentTypesDir
	if dir == "" {
		dir = "content-types"
	}
	if h.cfg.RequireTenantScope {
		if scope, err := tenant.FromContext(ctx); err == nil && scope.TenantID != "" {
			dir = filepath.Join(dir, "tenants", scope.TenantID)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("failed to create directory: %v", err)}}}, nil
	}

	yamlBytes, err := yaml.Marshal(def)
	if err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("failed to serialize YAML: %v", err)}}}, nil
	}

	filePath := filepath.Join(dir, def.ID+".yaml")
	if err := os.WriteFile(filePath, yamlBytes, 0o644); err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("failed to write definition file: %v", err)}}}, nil
	}

	if h.cfg.SyncSchema != nil {
		if err := h.cfg.SyncSchema(ctx, def); err != nil {
			return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("failed to sync runtime schema: %v", err)}}}, nil
		}
	}

	h.recordAudit(ctx, "content_type.defined", def.ID, nil, yamlBytes)

	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("Successfully provisioned content type '%s' at %s", def.ID, filePath),
			},
		},
	}, nil
}

func (h *Handler) toolQueryCollection(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Type   string `json:"type"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
		Sort   string `json:"sort"`
		Search string `json:"search"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "invalid JSON arguments"}}}, nil
	}

	if input.Type == "" {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "type is required"}}}, nil
	}
	if input.Limit <= 0 || input.Limit > 100 {
		input.Limit = 20
	}

	q := collections.Query{
		Limit:  input.Limit,
		Offset: input.Offset,
		Sort:   collections.ParseSort(input.Sort),
	}

	records, err := h.cfg.CollectionService.QueryRecords(ctx, input.Type, q)
	if err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("query error: %v", err)}}}, nil
	}

	resp := map[string]any{
		"items": records,
		"meta": map[string]any{
			"count":  len(records),
			"limit":  input.Limit,
			"offset": input.Offset,
		},
	}

	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(respJSON),
			},
		},
	}, nil
}

func (h *Handler) toolMutateRecord(ctx context.Context, args json.RawMessage) (ToolCallResult, error) {
	var input struct {
		Type      string         `json:"type"`
		Operation string         `json:"operation"`
		ID        string         `json:"id"`
		Data      map[string]any `json:"data"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "invalid JSON arguments"}}}, nil
	}

	if input.Type == "" || input.Operation == "" {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "type and operation are required"}}}, nil
	}

	var resultRecord any
	var resultErr error

	switch input.Operation {
	case "create":
		created, err := h.cfg.CollectionService.CreateRecord(ctx, input.Type, input.Data)
		if err != nil {
			resultErr = err
		} else {
			resultRecord = created
			beforeJSON, _ := json.Marshal(map[string]any{"actor_type": "ai_agent"})
			afterJSON, _ := json.Marshal(map[string]any{"actor_type": "ai_agent", "data": created})
			h.recordAudit(ctx, "content.create", fmt.Sprintf("%s:%s", input.Type, created.ID), beforeJSON, afterJSON)
		}

	case "update":
		if input.ID == "" {
			return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "id is required for update"}}}, nil
		}
		updated, err := h.cfg.CollectionService.UpdateRecord(ctx, input.Type, input.ID, input.Data)
		if err != nil {
			resultErr = err
		} else {
			resultRecord = updated
			beforeJSON, _ := json.Marshal(map[string]any{"actor_type": "ai_agent"})
			afterJSON, _ := json.Marshal(map[string]any{"actor_type": "ai_agent", "data": updated})
			h.recordAudit(ctx, "content.update", fmt.Sprintf("%s:%s", input.Type, input.ID), beforeJSON, afterJSON)
		}

	case "delete":
		if input.ID == "" {
			return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: "id is required for delete"}}}, nil
		}
		err := h.cfg.CollectionService.DeleteRecord(ctx, input.Type, input.ID)
		if err != nil {
			resultErr = err
		} else {
			resultRecord = map[string]any{"deleted": true, "id": input.ID}
			beforeJSON, _ := json.Marshal(map[string]any{"actor_type": "ai_agent", "id": input.ID})
			afterJSON, _ := json.Marshal(map[string]any{"actor_type": "ai_agent", "deleted": true})
			h.recordAudit(ctx, "content.delete", fmt.Sprintf("%s:%s", input.Type, input.ID), beforeJSON, afterJSON)
		}

	default:
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("unsupported operation: %s", input.Operation)}}}, nil
	}

	if resultErr != nil {
		return ToolCallResult{IsError: true, Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("mutation error: %v", resultErr)}}}, nil
	}

	respJSON, _ := json.MarshalIndent(resultRecord, "", "  ")
	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(respJSON),
			},
		},
	}, nil
}

func (h *Handler) requireScope(ctx context.Context, scope string) error {
	principal := auth.PrincipalFromContext(ctx)
	if len(principal.Scopes) > 0 {
		for _, s := range principal.Scopes {
			if s == auth.ScopeAdmin || s == "agent:admin" || s == scope {
				return nil
			}
		}
		return fmt.Errorf("insufficient agent permissions: requires %s scope", scope)
	}

	user := auth.UserFromContext(ctx)
	if user != nil {
		if !user.Active {
			return fmt.Errorf("unauthorized: inactive user")
		}
		if user.Role == auth.RoleAdmin {
			return nil
		}
		if scope == "agent:read" && auth.Can(user, auth.PermissionContentRead) {
			return nil
		}
		if (scope == "agent:mutate" || scope == "agent:schema") && auth.Can(user, auth.PermissionContentWrite) {
			return nil
		}
		return fmt.Errorf("insufficient user permissions: requires %s scope", scope)
	}

	if h.cfg.RequireAuth {
		return fmt.Errorf("authentication required for MCP endpoint")
	}

	return nil
}

func (h *Handler) recordAudit(ctx context.Context, action, targetID string, before, after []byte) {
	if h.cfg.AuditStore == nil {
		return
	}
	_ = h.cfg.AuditStore.RecordDetails(ctx, action, targetID, before, after)
}

func writeRPCResult(w http.ResponseWriter, id any, result any) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func writeRPCError(w http.ResponseWriter, id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

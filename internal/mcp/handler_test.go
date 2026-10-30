package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

type mockAuditStore struct {
	recorded []string
}

func (m *mockAuditStore) RecordDetails(_ context.Context, action, targetID string, _, after []byte) error {
	m.recorded = append(m.recorded, action+":"+targetID+":"+string(after))
	return nil
}

type mockRepo struct {
	records map[string][]collections.Record
	schemas map[string]collections.Schema
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		records: make(map[string][]collections.Record),
		schemas: map[string]collections.Schema{
			"faq": {
				ID:    "faq",
				Label: "FAQ",
				Fields: []collections.Field{
					{ID: "question", Label: "Question", Kind: collections.Text},
					{ID: "answer", Label: "Answer", Kind: collections.Text},
				},
			},
		},
	}
}

func (m *mockRepo) ListSchemas(_ context.Context) ([]collections.Schema, error) {
	var list []collections.Schema
	for _, s := range m.schemas {
		list = append(list, s)
	}
	return list, nil
}

func (m *mockRepo) GetSchema(_ context.Context, id string) (collections.Schema, error) {
	s, ok := m.schemas[id]
	if !ok {
		return collections.Schema{}, collections.ErrCollectionNotFound
	}
	return s, nil
}

func (m *mockRepo) CreateSchema(_ context.Context, s collections.Schema) error {
	m.schemas[s.ID] = s
	return nil
}

func (m *mockRepo) UpdateSchema(_ context.Context, s collections.Schema) error {
	m.schemas[s.ID] = s
	return nil
}

func (m *mockRepo) DeleteSchema(_ context.Context, id string) error {
	delete(m.schemas, id)
	return nil
}

func (m *mockRepo) QueryRecords(_ context.Context, collectionID string, _ collections.Query) ([]collections.Record, error) {
	return m.records[collectionID], nil
}

func (m *mockRepo) AggregateRecords(_ context.Context, _ string, _ collections.AggregateQuery) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockRepo) GetRecord(_ context.Context, collectionID, id string) (collections.Record, error) {
	for _, r := range m.records[collectionID] {
		if r.ID == id {
			return r, nil
		}
	}
	return collections.Record{}, collections.ErrRecordNotFound
}

func (m *mockRepo) SaveRecord(_ context.Context, collectionID, id string, data map[string]any, _ []collections.Field, _ int) (collections.Record, error) {
	if id == "" {
		id = "rec-1"
	}
	rec := collections.Record{
		ID:           id,
		CollectionID: collectionID,
		Data:         data,
		Version:      1,
	}
	m.records[collectionID] = append(m.records[collectionID], rec)
	return rec, nil
}

func (m *mockRepo) DeleteRecord(_ context.Context, collectionID, id string) error {
	var remaining []collections.Record
	for _, r := range m.records[collectionID] {
		if r.ID != id {
			remaining = append(remaining, r)
		}
	}
	m.records[collectionID] = remaining
	return nil
}

func (m *mockRepo) RecordExists(_ context.Context, collectionID, id string) (bool, error) {
	for _, r := range m.records[collectionID] {
		if r.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func setupTestMCP(t *testing.T) (*Handler, *mockAuditStore) {
	t.Helper()
	repo := newMockRepo()
	colService := collections.Service{Repository: repo}
	auditStore := &mockAuditStore{}
	tmpDir := t.TempDir()

	types := []contenttypes.Definition{
		{
			ID:    "faq",
			Label: "FAQ",
			Fields: []contenttypes.Field{
				{ID: "question", Label: "Question", Kind: "text", Required: true},
				{ID: "answer", Label: "Answer", Kind: "text", Required: true},
			},
		},
	}

	h := NewHandler(Config{
		CollectionService: colService,
		ContentTypesDir:   tmpDir,
		ContentTypes:      func() []contenttypes.Definition { return types },
		SyncSchema: func(_ context.Context, def contenttypes.Definition) error {
			repo.schemas[def.ID] = collections.Schema{ID: def.ID}
			return nil
		},
		AuditStore: auditStore,
	})

	return h, auditStore
}

func TestMCPGetHandshakeAndSSE(t *testing.T) {
	h, _ := setupTestMCP(t)

	// Plain GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mcp", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "hydrogo-mcp") {
		t.Fatalf("expected hydrogo-mcp in response, got %s", w.Body.String())
	}

	// SSE GET
	sseReq := httptest.NewRequest(http.MethodGet, "/api/v1/mcp", nil)
	sseReq.Header.Set("Accept", "text/event-stream")
	sseW := httptest.NewRecorder()
	h.ServeHTTP(sseW, sseReq)

	if sseW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", sseW.Code)
	}
	if !strings.Contains(sseW.Body.String(), "event: endpoint") {
		t.Fatalf("expected event: endpoint, got %s", sseW.Body.String())
	}
}

func TestMCPInitializeAndToolsList(t *testing.T) {
	h, _ := setupTestMCP(t)

	initPayload := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(initPayload))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var initResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &initResp); err != nil {
		t.Fatalf("unmarshal init response: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("unexpected error: %v", initResp.Error)
	}

	// Tools list
	toolsPayload := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(toolsPayload))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var toolsResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &toolsResp); err != nil {
		t.Fatalf("unmarshal tools response: %v", err)
	}
	if toolsResp.Error != nil {
		t.Fatalf("unexpected error: %v", toolsResp.Error)
	}
}

func TestMCPToolsCallDefineAndMutateWithAudit(t *testing.T) {
	h, auditStore := setupTestMCP(t)

	// Call define_content_type
	definePayload := `{
		"jsonrpc": "2.0",
		"id": 10,
		"method": "tools/call",
		"params": {
			"name": "define_content_type",
			"arguments": {
				"id": "articles",
				"label": "Articles",
				"fields": [
					{"id": "title", "label": "Title", "kind": "text", "required": true}
				]
			}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(definePayload))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var defineResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &defineResp); err != nil {
		t.Fatalf("unmarshal define response: %v", err)
	}
	if defineResp.Error != nil {
		t.Fatalf("unexpected error: %v", defineResp.Error)
	}

	// Call mutate_record (create)
	mutatePayload := `{
		"jsonrpc": "2.0",
		"id": 11,
		"method": "tools/call",
		"params": {
			"name": "mutate_record",
			"arguments": {
				"type": "faq",
				"operation": "create",
				"data": {
					"question": "What is HydroGo?",
					"answer": "Headless CMS and AI control plane."
				}
			}
		}
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(mutatePayload))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var mutateResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &mutateResp); err != nil {
		t.Fatalf("unmarshal mutate response: %v", err)
	}
	if mutateResp.Error != nil {
		t.Fatalf("unexpected error: %v", mutateResp.Error)
	}

	// Verify audit trail recorded with actor_type: ai_agent
	foundAudit := false
	for _, entry := range auditStore.recorded {
		if strings.Contains(entry, "content.create") && strings.Contains(entry, "actor_type") && strings.Contains(entry, "ai_agent") {
			foundAudit = true
			break
		}
	}
	if !foundAudit {
		t.Fatalf("expected audit log with actor_type: ai_agent, got: %v", auditStore.recorded)
	}

	// Query collection
	queryPayload := `{
		"jsonrpc": "2.0",
		"id": 12,
		"method": "tools/call",
		"params": {
			"name": "query_collection",
			"arguments": {
				"type": "faq",
				"limit": 10
			}
		}
	}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(queryPayload))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var queryResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &queryResp); err != nil {
		t.Fatalf("unmarshal query response: %v", err)
	}
	if queryResp.Error != nil {
		t.Fatalf("unexpected error: %v", queryResp.Error)
	}
}

func TestMCPResources(t *testing.T) {
	h, _ := setupTestMCP(t)

	// List resources
	listPayload := `{"jsonrpc":"2.0","id":20,"method":"resources/list","params":{}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(listPayload))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var listResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if listResp.Error != nil {
		t.Fatalf("unexpected error: %v", listResp.Error)
	}

	// Read resource
	readPayload := `{"jsonrpc":"2.0","id":21,"method":"resources/read","params":{"uri":"hydro://schemas"}}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(readPayload))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var readResp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &readResp); err != nil {
		t.Fatalf("unmarshal read response: %v", err)
	}
	if readResp.Error != nil {
		t.Fatalf("unexpected error: %v", readResp.Error)
	}
}

func TestMCPScopeEnforcement(t *testing.T) {
	h, _ := setupTestMCP(t)

	// Token with only agent:read scope
	readTokenPrincipal := auth.Principal{
		Kind:   auth.PrincipalToken,
		Scopes: []string{"agent:read"},
	}

	ctx := auth.WithPrincipal(context.Background(), readTokenPrincipal)

	// Calling mutate_record should be rejected
	mutatePayload := `{
		"jsonrpc": "2.0",
		"id": 30,
		"method": "tools/call",
		"params": {
			"name": "mutate_record",
			"arguments": {
				"type": "faq",
				"operation": "create",
				"data": {"question": "Q", "answer": "A"}
			}
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(mutatePayload)).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil {
		t.Fatalf("expected permission error for agent:read trying to mutate, got success: %v", resp.Result)
	}
	if resp.Error.Code != -32000 && resp.Error.Code != -32001 {
		t.Fatalf("expected error code -32000 or -32001, got %d", resp.Error.Code)
	}

	// Viewer user cannot mutate records
	viewerCtx := auth.WithUser(context.Background(), &auth.User{
		ID:     "viewer-1",
		Role:   auth.RoleViewer,
		Active: true,
	})
	reqViewer := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(mutatePayload)).WithContext(viewerCtx)
	wViewer := httptest.NewRecorder()
	h.ServeHTTP(wViewer, reqViewer)
	var viewerResp JSONRPCResponse
	if err := json.Unmarshal(wViewer.Body.Bytes(), &viewerResp); err != nil {
		t.Fatal(err)
	}
	if viewerResp.Error == nil {
		t.Fatalf("expected viewer mutation to be rejected, got: %v", viewerResp.Result)
	}

	// RequireAuth rejects unauthenticated requests
	hAuth := NewHandler(Config{
		CollectionService: h.cfg.CollectionService,
		RequireAuth:       true,
	})
	reqAnon := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", strings.NewReader(mutatePayload))
	wAnon := httptest.NewRecorder()
	hAuth.ServeHTTP(wAnon, reqAnon)
	var anonResp JSONRPCResponse
	if err := json.Unmarshal(wAnon.Body.Bytes(), &anonResp); err != nil {
		t.Fatal(err)
	}
	if anonResp.Error == nil {
		t.Fatalf("expected unauthenticated request to be rejected when RequireAuth is true")
	}
}

var _ = filepath.Join
var _ = bytes.NewReader

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mcp"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestMCPEndpointWiring(t *testing.T) {
	root := t.TempDir()
	contentTypesDir := filepath.Join(root, "content-types")
	if err := os.MkdirAll(contentTypesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	faqYaml := `id: faq
label: Frequently Asked Questions
public: true
fields:
  - id: question
    label: Question
    kind: text
    required: true
  - id: answer
    label: Answer
    kind: richtext
    required: true
`
	if err := os.WriteFile(filepath.Join(contentTypesDir, "faq.yaml"), []byte(faqYaml), 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "mcp.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	colService := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite}}
	app := &serveApp{
		db:                 db,
		collectionService:  colService,
		requireTenantScope: false,
	}
	app.cfg.ContentTypesDir = contentTypesDir
	app.cfg.Database.Dialect = database.SQLite

	if err := app.syncContentTypes(context.Background()); err != nil {
		t.Fatalf("syncContentTypes: %v", err)
	}

	routes := app.mcpRoutes()
	var getHandler, postHandler http.Handler
	for _, route := range routes {
		if route.Pattern == "GET /api/v1/mcp" {
			getHandler = route.Handler
		} else if route.Pattern == "POST /api/v1/mcp" {
			postHandler = route.Handler
		}
	}

	if getHandler == nil || postHandler == nil {
		t.Fatal("MCP routes not registered properly")
	}

	// 1. Handshake GET
	req := httptest.NewRequest("GET", "/api/v1/mcp", nil)
	w := httptest.NewRecorder()
	getHandler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 2. Introspect list_content_types
	listReq := httptest.NewRequest("POST", "/api/v1/mcp", strings.NewReader(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {
			"name": "list_content_types",
			"arguments": {}
		}
	}`))
	w = httptest.NewRecorder()
	postHandler.ServeHTTP(w, listReq)

	var listResp mcp.JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if listResp.Error != nil {
		t.Fatalf("unexpected error: %v", listResp.Error)
	}

	// 3. Mutate record via MCP
	createReq := httptest.NewRequest("POST", "/api/v1/mcp", strings.NewReader(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "tools/call",
		"params": {
			"name": "mutate_record",
			"arguments": {
				"type": "faq",
				"operation": "create",
				"data": {
					"question": "What is MCP?",
					"answer": "Model Context Protocol."
				}
			}
		}
	}`))
	w = httptest.NewRecorder()
	postHandler.ServeHTTP(w, createReq)

	var createResp mcp.JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	if createResp.Error != nil {
		t.Fatalf("unexpected error: %v", createResp.Error)
	}
}

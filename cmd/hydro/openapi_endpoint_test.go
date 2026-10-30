package main

import (
	"bytes"
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
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSDKGenerateWithContentTypes(t *testing.T) {
	root := t.TempDir()
	contentTypesDir := filepath.Join(root, "content-types")
	if err := os.MkdirAll(contentTypesDir, 0o750); err != nil {
		t.Fatal(err)
	}

	productYaml := `id: product
label: Product
public: true
drafts: true
fields:
  - id: title
    label: Title
    kind: text
    required: true
  - id: price
    label: Price
    kind: number
`
	if err := os.WriteFile(filepath.Join(contentTypesDir, "product.yaml"), []byte(productYaml), 0o600); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(root, "sdk")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"sdk", "generate", "--out=" + out, "--content-types-dir=" + contentTypesDir}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("sdk generate with content-types failed: %v (%s)", err, stderr.String())
	}

	tsClientPath := filepath.Join(out, "typescript", "client.ts")
	tsContent, err := os.ReadFile(tsClientPath)
	if err != nil {
		t.Fatalf("failed to read generated client.ts: %v", err)
	}

	tsStr := string(tsContent)
	if !strings.Contains(tsStr, "ProductRecord") {
		t.Fatalf("expected client.ts to contain ProductRecord type, got:\n%s", tsStr)
	}
	if !strings.Contains(tsStr, "listProduct") {
		t.Fatalf("expected client.ts to contain listProduct method, got:\n%s", tsStr)
	}
}

func TestOpenAPIEndpoint(t *testing.T) {
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
`
	if err := os.WriteFile(filepath.Join(contentTypesDir, "faq.yaml"), []byte(faqYaml), 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "openapi.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	colService := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite}}
	_, routes, err := contentTypeRoutes(context.Background(), contentTypesDir, colService, nil, contenttypes.Locales{}, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("contentTypeRoutes failed: %v", err)
	}

	var openAPIRoute http.Handler
	for _, route := range routes {
		if route.Pattern == "GET /api/v1/openapi.json" {
			openAPIRoute = route.Handler
			break
		}
	}
	if openAPIRoute == nil {
		t.Fatal("route GET /api/v1/openapi.json not registered")
	}

	req := httptest.NewRequest("GET", "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	openAPIRoute.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var doc struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]any `json:"paths"`
		Components struct {
			Schemas map[string]any `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal openapi.json: %v", err)
	}

	if doc.OpenAPI != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %s", doc.OpenAPI)
	}
	if _, ok := doc.Components.Schemas["FaqRecord"]; !ok {
		t.Fatalf("expected FaqRecord in components.schemas, got: %+v", doc.Components.Schemas)
	}
	if _, ok := doc.Paths["/api/v1/content/faq"]; !ok {
		t.Fatalf("expected /api/v1/content/faq in paths, got: %+v", doc.Paths)
	}
}

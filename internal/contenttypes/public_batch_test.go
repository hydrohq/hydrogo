package contenttypes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

const batchItemsYaml = `id: items
label: Items
public: true
fields:
  - id: title
    label: Title
    kind: text
    required: true
  - id: value
    label: Value
    kind: number
    required: true
`

func setupPublicBatchAPI(t *testing.T) (PublicAPI, collections.Service) {
	t.Helper()
	service, defs := newServiceFor(t, batchItemsYaml)
	api := PublicAPI{
		Service: service,
		Types:   defs,
		ResolvePrincipal: func(r *http.Request) Principal {
			return Principal{Anonymous: false, Role: "admin", IsAdmin: true}
		},
	}
	return api, service
}

func TestPublicBatchCreates100RecordsAtomically(t *testing.T) {
	api, colService := setupPublicBatchAPI(t)

	var ops []map[string]any
	for i := 1; i <= 100; i++ {
		ops = append(ops, map[string]any{
			"action": "create",
			"type":   "items",
			"data": map[string]any{
				"title": fmt.Sprintf("Item %d", i),
				"value": i * 10,
			},
		})
	}

	body, _ := json.Marshal(map[string]any{
		"atomic":     true,
		"operations": ops,
	})

	start := time.Now()
	req := httptest.NewRequest("POST", "/api/v1/content/items/batch", strings.NewReader(string(body)))
	req.SetPathValue("type", "items")
	w := httptest.NewRecorder()
	api.Batch(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	t.Logf("100 records batch created in %s", elapsed)

	// Verify all 100 records were stored
	records, err := colService.ListRecords(context.Background(), "items", 100, 0)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(records) != 100 {
		t.Fatalf("expected 100 records in database, got %d", len(records))
	}
}

func TestPublicBatchRollbackOnFailure(t *testing.T) {
	api, colService := setupPublicBatchAPI(t)

	// First operation is valid, second fails (missing required 'value' field)
	ops := []map[string]any{
		{
			"action": "create",
			"type":   "items",
			"data": map[string]any{
				"title": "Valid Item",
				"value": 100,
			},
		},
		{
			"action": "create",
			"type":   "items",
			"data": map[string]any{
				"title": "Invalid Item Missing Value",
				// missing required "value"
			},
		},
	}

	body, _ := json.Marshal(map[string]any{
		"atomic":     true,
		"operations": ops,
	})

	req := httptest.NewRequest("POST", "/api/v1/content/items/batch", strings.NewReader(string(body)))
	req.SetPathValue("type", "items")
	w := httptest.NewRecorder()
	api.Batch(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 UnprocessableEntity on atomic failure, got %d: %s", w.Code, w.Body.String())
	}

	// Verify zero records were created (clean rollback)
	records, err := colService.ListRecords(context.Background(), "items", 10, 0)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records after rollback, got %d", len(records))
	}
}

func TestPublicBatchNonAtomicContinues(t *testing.T) {
	api, colService := setupPublicBatchAPI(t)

	// First operation is valid, second fails, third is valid
	ops := []map[string]any{
		{
			"action": "create",
			"type":   "items",
			"data": map[string]any{
				"title": "Item 1",
				"value": 10,
			},
		},
		{
			"action": "create",
			"type":   "items",
			"data": map[string]any{
				"title": "Item 2 Missing Value",
			},
		},
		{
			"action": "create",
			"type":   "items",
			"data": map[string]any{
				"title": "Item 3",
				"value": 30,
			},
		},
	}

	body, _ := json.Marshal(map[string]any{
		"atomic":     false,
		"operations": ops,
	})

	req := httptest.NewRequest("POST", "/api/v1/content/items/batch", strings.NewReader(string(body)))
	req.SetPathValue("type", "items")
	w := httptest.NewRecorder()
	api.Batch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for non-atomic batch, got %d: %s", w.Code, w.Body.String())
	}

	// Verify items 1 and 3 were stored (2 records)
	records, err := colService.ListRecords(context.Background(), "items", 10, 0)
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records stored, got %d", len(records))
	}
}

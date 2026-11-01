package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	aicore "github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestAdminAICopilotExecuteAndUndo(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "copilot.sqlite")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("ApplyMigrations: %v", err)
	}

	colSvc := collections.Service{
		Repository: collectionstore.Store{
			DB:                 db,
			Dialect:            database.SQLite,
			RequireTenantScope: true,
		},
	}

	app := &serveApp{
		// The test feeds an executable JSON plan through the prompt, which is
		// only accepted in dev mode (production prompts are never plans).
		devMode:            true,
		db:                 db,
		collectionService:  colSvc,
		requireTenantScope: true,
		siteStore: sitestore.Store{
			Dialect:            database.SQLite,
			RequireTenantScope: true,
		},
		contentTypes: []contenttypes.Definition{
			{
				ID:     "boutique_products",
				Label:  "Ürünler",
				Public: true,
				Fields: []contenttypes.Field{
					{ID: "title", Label: "Başlık", Kind: "text", Required: true},
					{ID: "price", Label: "Fiyat", Kind: "number", Required: true},
				},
			},
		},
	}
	app.cfg.Database.Dialect = database.SQLite

	scope, err := tenant.NewScope("ipek-butik", "ipek-butik")
	if err != nil {
		t.Fatal(err)
	}
	tenantCtx := tenant.WithScope(ctx, scope)
	storeOwner := &auth.User{
		ID:     "owner-1",
		Email:  "owner@ipekbutik.com",
		Role:   auth.RoleAdmin,
		Active: true,
	}
	tenantCtx = auth.WithUser(tenantCtx, storeOwner)

	schemas, err := contenttypes.Schemas(app.contentTypes)
	if err != nil {
		t.Fatal(err)
	}
	if err := colSvc.SyncSchemas(tenantCtx, schemas); err != nil {
		t.Fatalf("SyncSchemas: %v", err)
	}

	// Seed an initial product
	initialProd, err := colSvc.CreateRecord(tenantCtx, "boutique_products", map[string]any{
		"title": "İpek Gömlek",
		"price": 1200.0,
	})
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	gov := tenant.DefaultGovernance("ipek-butik")
	gov.AIEnabled = true
	gov.AIMonthlyTokenLimit = 10000
	if err := tenant.UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatal(err)
	}

	// 1. Send a Copilot instruction that updates the existing product's price and creates a new product
	planPromptObj := aicore.CopilotPlan{
		Reply: "İpek Gömlek fiyatını 1450 TL yaptım ve Keten Pantolon ürününü ekledim.",
		Operations: []aicore.CopilotOperation{
			{
				Op:           "update",
				CollectionID: "boutique_products",
				RecordID:     initialProd.ID,
				Data:         map[string]any{"price": 1450.0},
			},
			{
				Op:           "create",
				CollectionID: "boutique_products",
				Data:         map[string]any{"title": "Keten Pantolon", "price": 950.0},
			},
		},
	}
	planBytes, _ := json.Marshal(planPromptObj)
	reqBody, _ := json.Marshal(map[string]string{"prompt": string(planBytes)})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/copilot", bytes.NewReader(reqBody)).WithContext(tenantCtx)
	rec := httptest.NewRecorder()
	app.adminAICopilotHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from copilot, got %d: %s", rec.Code, rec.Body.String())
	}

	var copilotResp struct {
		Data struct {
			Reply      string                  `json:"reply"`
			ActionLog  aicore.CopilotActionLog `json:"action_log"`
			Governance tenant.Governance       `json:"governance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &copilotResp); err != nil {
		t.Fatal(err)
	}
	if copilotResp.Data.ActionLog.ID == "" || len(copilotResp.Data.ActionLog.Operations) != 2 {
		t.Fatalf("unexpected action_log: %+v", copilotResp.Data.ActionLog)
	}
	if copilotResp.Data.Governance.AITokensUsed <= 0 {
		t.Fatalf("expected AI tokens to be consumed, got %d", copilotResp.Data.Governance.AITokensUsed)
	}

	// Verify changes took effect in database
	updatedProd, err := colSvc.GetRecord(tenantCtx, "boutique_products", initialProd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedProd.Data["price"] != 1450.0 || updatedProd.Data["title"] != "İpek Gömlek" {
		t.Fatalf("expected price 1450 and title preserved, got %+v", updatedProd.Data)
	}
	allProds, err := colSvc.QueryRecords(tenantCtx, "boutique_products", collections.Query{Limit: 10})
	if err != nil || len(allProds) != 2 {
		t.Fatalf("expected 2 products after copilot execution, got %d (%v)", len(allProds), err)
	}

	// 2. Undo the Copilot action via POST /api/v1/admin/ai/actions/{id}/undo
	reqUndo := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/actions/"+copilotResp.Data.ActionLog.ID+"/undo", nil).WithContext(tenantCtx)
	reqUndo.SetPathValue("id", copilotResp.Data.ActionLog.ID)
	recUndo := httptest.NewRecorder()
	app.adminAIActionUndoHandler(recUndo, reqUndo)
	if recUndo.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from undo, got %d: %s", recUndo.Code, recUndo.Body.String())
	}

	// Verify the updated product reverted to 1200.0 and the created product was deleted
	revertedProd, err := colSvc.GetRecord(tenantCtx, "boutique_products", initialProd.ID)
	if err != nil {
		t.Fatal(err)
	}
	if revertedProd.Data["price"] != 1200.0 {
		t.Fatalf("expected price reverted to 1200.0 after undo, got %v", revertedProd.Data["price"])
	}
	remainingProds, err := colSvc.QueryRecords(tenantCtx, "boutique_products", collections.Query{Limit: 10})
	if err != nil || len(remainingProds) != 1 {
		t.Fatalf("expected 1 product after undo, got %d", len(remainingProds))
	}

	// 3. Test AI token quota enforcement (429 Too Many Requests)
	gov.AITokensUsed = 10000
	gov.AIMonthlyTokenLimit = 10000
	if err := tenant.UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatal(err)
	}
	reqQuota := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/copilot", bytes.NewReader(reqBody)).WithContext(tenantCtx)
	recQuota := httptest.NewRecorder()
	app.adminAICopilotHandler(recQuota, reqQuota)
	if recQuota.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests when AI quota is exhausted, got %d", recQuota.Code)
	}
}

func TestAdminAICopilotRejectsUnknownFieldsAndProdRawPlans(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "copilot-validate.sqlite")
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatalf("ApplyMigrations: %v", err)
	}

	colSvc := collections.Service{
		Repository: collectionstore.Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true},
	}
	app := &serveApp{
		db:                 db,
		collectionService:  colSvc,
		requireTenantScope: true,
		siteStore:          sitestore.Store{Dialect: database.SQLite, RequireTenantScope: true},
		contentTypes: []contenttypes.Definition{
			{
				ID:     "products",
				Label:  "Ürünler",
				Public: true,
				Fields: []contenttypes.Field{
					{ID: "title", Label: "Başlık", Kind: "text", Required: true},
				},
			},
		},
	}
	app.cfg.Database.Dialect = database.SQLite

	scope, err := tenant.NewScope("shop", "shop")
	if err != nil {
		t.Fatal(err)
	}
	tenantCtx := auth.WithUser(tenant.WithScope(ctx, scope), &auth.User{ID: "u1", Email: "u@shop.com", Role: auth.RoleAdmin, Active: true})

	schemas, err := contenttypes.Schemas(app.contentTypes)
	if err != nil {
		t.Fatal(err)
	}
	if err := colSvc.SyncSchemas(tenantCtx, schemas); err != nil {
		t.Fatalf("SyncSchemas: %v", err)
	}

	gov := tenant.DefaultGovernance("shop")
	gov.AIMonthlyTokenLimit = 10000
	if err := tenant.UpsertGovernance(ctx, db, "sqlite", gov); err != nil {
		t.Fatal(err)
	}

	// 1. Production (devMode off): a JSON plan in the prompt must not execute.
	rawPlan := `{"reply":"tamam","operations":[{"op":"create","collection_id":"products","data":{"title":"X"}}]}`
	reqProd := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/copilot", bytes.NewReader([]byte(`{"prompt":`+strconv.Quote(rawPlan)+`}`))).WithContext(tenantCtx)
	recProd := httptest.NewRecorder()
	app.adminAICopilotHandler(recProd, reqProd)
	if recProd.Code != http.StatusOK {
		t.Fatalf("expected 200 heuristic reply in prod, got %d: %s", recProd.Code, recProd.Body.String())
	}
	prods, err := colSvc.QueryRecords(tenantCtx, "products", collections.Query{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 0 {
		t.Fatalf("raw JSON prompt executed %d operations in prod mode", len(prods))
	}

	// 2. Dev mode plan that targets a field outside the schema must be rejected
	// before any mutation (field allowlist).
	app.devMode = true
	badPlan := `{"reply":"tamam","operations":[{"op":"create","collection_id":"products","data":{"title":"X","evil_column":"y"}}]}`
	reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/admin/ai/copilot", bytes.NewReader([]byte(`{"prompt":`+strconv.Quote(badPlan)+`}`))).WithContext(tenantCtx)
	recBad := httptest.NewRecorder()
	app.adminAICopilotHandler(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d: %s", recBad.Code, recBad.Body.String())
	}
	prods, err = colSvc.QueryRecords(tenantCtx, "products", collections.Query{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(prods) != 0 {
		t.Fatalf("rejected plan still created %d records", len(prods))
	}
}

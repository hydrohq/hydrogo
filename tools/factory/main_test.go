package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func TestRunFactoryProvisionsAndSeedsBoutiqueStore(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "factory.sqlite")
	contentTypesDir := filepath.Join("..", "..", "content-types", "boutique")

	var out bytes.Buffer
	res, err := RunFactory(ctx, FactoryOptions{
		Dialect:         "sqlite",
		DSN:             dbPath,
		TenantID:        "noir-atelier",
		DisplayName:     "Noir Atelier",
		AdminEmail:      "owner@noiratelier.com",
		Subdomain:       "noir.butik.local",
		CustomDomain:    "www.noiratelier.com",
		Plan:            "vip",
		OwnerPhone:      "+905559998877",
		Prompt:          "Siyah ve altın detaylı lüks gece elbiseleri satan koyu temalı butik",
		ContentTypesDir: contentTypesDir,
		TrialDays:       30,
	}, &out)
	if err != nil {
		t.Fatalf("RunFactory failed: %v", err)
	}

	if res.TenantID != "noir-atelier" || res.ThemePreset != "noir_gold" {
		t.Fatalf("expected noir_gold preset for dark/gold prompt, got %+v", res)
	}
	if res.CategoriesSeeded != 3 || res.ProductsSeeded != 4 {
		t.Fatalf("expected 3 categories and 4 products seeded, got %+v", res)
	}
	if res.GeneratedPassword == "" {
		t.Fatal("expected generated password to be returned")
	}

	// Verify database records in tenant scope
	db, err := database.Open(ctx, database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gov, err := tenant.GetGovernance(ctx, db, "sqlite", "noir-atelier")
	if err != nil {
		t.Fatalf("GetGovernance: %v", err)
	}
	if gov.Plan != "vip" || gov.CustomDomain != "www.noiratelier.com" {
		t.Fatalf("unexpected governance: %+v", gov)
	}

	scope, _ := tenant.NewScope("noir-atelier", "noir-atelier")
	tenantCtx := tenant.WithScope(ctx, scope)
	colSvc := collections.Service{
		Repository: collectionstore.Store{
			DB:                 db,
			Dialect:            database.SQLite,
			RequireTenantScope: true,
		},
	}
	prods, err := colSvc.QueryRecords(tenantCtx, "boutique_products", collections.Query{Limit: 10})
	if err != nil || len(prods) != 4 {
		t.Fatalf("expected 4 products in tenant scope, got %d (%v)", len(prods), err)
	}
}

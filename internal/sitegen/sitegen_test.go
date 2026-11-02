package sitegen

import (
	"context"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type mockCollectionService struct {
	createdRecords []map[string]any
}

func (m *mockCollectionService) CreateRecord(ctx context.Context, collectionID string, data map[string]any) (collections.Record, error) {
	record := map[string]any{
		"collection_id": collectionID,
	}
	for k, v := range data {
		record[k] = v
	}
	m.createdRecords = append(m.createdRecords, record)
	return collections.Record{ID: "rec-test", Data: data}, nil
}

func TestSitePlanValidation(t *testing.T) {
	plan := &SitePlan{
		Brief: "El yapımı deri çanta",
		StoreSettings: map[string]any{
			"store_name": "Arda Leather",
		},
		Layout: []LayoutSection{
			{Type: "hero", Props: map[string]any{"title": "Deri Sanatı"}},
			{Type: "invalid_future_sec", Props: map[string]any{"x": "y"}},
			{Type: "pillars", Props: map[string]any{"title": "İlkeler"}},
		},
		Products: []ProductItem{
			{Title: "Klasik Çanta", Price: 1500},
			{Title: "", Price: -10}, // Should be fixed by Validate
		},
	}

	if err := Validate(plan); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if len(plan.Layout) != 2 {
		t.Errorf("expected 2 valid layout sections, got %d", len(plan.Layout))
	}
	if plan.Products[1].Title != "Ürün 2" {
		t.Errorf("expected empty title to default to 'Ürün 2', got %q", plan.Products[1].Title)
	}
	if plan.Products[1].Price != 0 {
		t.Errorf("expected negative price to clamp to 0, got %f", plan.Products[1].Price)
	}
	if plan.Products[0].Slug != "klasik-canta" {
		t.Errorf("expected slug 'klasik-canta', got %q", plan.Products[0].Slug)
	}
}

func TestGenerateDeterministicFallback(t *testing.T) {
	plan := GenerateDeterministicFallback("Koyu siyah ve altın takılar", "Noir Bijoux", "+905551234567")
	if plan.Theme.Preset != "noir_gold" {
		t.Errorf("expected preset 'noir_gold', got %q", plan.Theme.Preset)
	}
	if len(plan.Products) < 3 {
		t.Errorf("expected at least 3 products in fallback, got %d", len(plan.Products))
	}
	if len(plan.Layout) < 4 {
		t.Errorf("expected at least 4 layout sections in fallback, got %d", len(plan.Layout))
	}
	if err := Validate(&plan); err != nil {
		t.Fatalf("fallback plan failed validation: %v", err)
	}
}

func TestSiteGenExecute(t *testing.T) {
	mock := &mockCollectionService{}
	plan := GenerateDeterministicFallback("Doğal sabunlar", "Vera Botanik", "+905550001122")

	scope := tenant.Scope{TenantID: "tenant-vera"}
	svc := collections.Service{
		Repository: nil,
	}
	_ = svc

	// Call Execute with our interface-compatible mock or test directly
	err := Execute(context.Background(), mock, scope, plan)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(mock.createdRecords) != 1+len(plan.Products) {
		t.Errorf("expected %d created records (1 store_settings + %d products), got %d",
			1+len(plan.Products), len(plan.Products), len(mock.createdRecords))
	}
}

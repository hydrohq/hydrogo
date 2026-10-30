package sqlstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestAggregateRecordsAndGroupBy(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "aggregate_test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	service := collections.Service{Repository: Store{DB: db, Dialect: database.SQLite}}
	colID := "orders"
	schema := collections.Schema{
		ID:    colID,
		Label: "Orders",
		Fields: []collections.Field{
			{ID: "category", Label: "Category", Kind: collections.Text},
			{ID: "status", Label: "Status", Kind: collections.Text},
			{ID: "total", Label: "Total", Kind: collections.Number},
			{ID: "rating", Label: "Rating", Kind: collections.Number},
		},
	}
	if err := service.CreateSchema(ctx, schema); err != nil {
		t.Fatal(err)
	}

	orders := []map[string]any{
		{"category": "electronics", "status": "completed", "total": 100.0, "rating": 5.0},
		{"category": "electronics", "status": "completed", "total": 200.0, "rating": 3.0},
		{"category": "electronics", "status": "pending", "total": 50.0, "rating": 4.0},
		{"category": "clothing", "status": "completed", "total": 80.0, "rating": 4.0},
		{"category": "clothing", "status": "pending", "total": 20.0, "rating": 2.0},
	}
	for _, o := range orders {
		if _, err := service.CreateRecord(ctx, colID, o); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Total aggregation without group by
	totAgg := collections.AggregateQuery{
		Aggregates: []collections.AggregateSpec{
			{Func: collections.AggCount, Alias: "count"},
			{Func: collections.AggSum, Field: "total", Alias: "sum_total"},
			{Func: collections.AggAvg, Field: "rating", Alias: "avg_rating"},
			{Func: collections.AggMin, Field: "total", Alias: "min_total"},
			{Func: collections.AggMax, Field: "total", Alias: "max_total"},
		},
	}
	res, err := service.AggregateRecords(ctx, colID, totAgg)
	if err != nil {
		t.Fatalf("AggregateRecords total: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	row := res[0]
	if count, ok := row["count"].(int64); !ok || count != 5 {
		t.Errorf("expected count 5, got %v (%T)", row["count"], row["count"])
	}
	if sum, ok := row["sum_total"].(float64); !ok || sum != 450.0 {
		t.Errorf("expected sum_total 450.0, got %v (%T)", row["sum_total"], row["sum_total"])
	}

	// 2. Group by category
	grpAgg := collections.AggregateQuery{
		Aggregates: []collections.AggregateSpec{
			{Func: collections.AggCount, Alias: "count"},
			{Func: collections.AggSum, Field: "total", Alias: "sum_total"},
		},
		GroupBy: []string{"category"},
	}
	resGrp, err := service.AggregateRecords(ctx, colID, grpAgg)
	if err != nil {
		t.Fatalf("AggregateRecords group_by: %v", err)
	}
	if len(resGrp) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(resGrp))
	}

	// Check clothing
	cloth := resGrp[0]
	grpCloth, _ := cloth["group"].(map[string]any)
	if grpCloth["category"] != "clothing" {
		t.Errorf("expected clothing first, got %v", grpCloth["category"])
	}
	if count, _ := cloth["count"].(int64); count != 2 {
		t.Errorf("clothing count want 2, got %v", count)
	}
	if sum, _ := cloth["sum_total"].(float64); sum != 100.0 {
		t.Errorf("clothing sum want 100.0, got %v", sum)
	}

	// 3. Filtered aggregate (status = 'completed')
	fltAgg := collections.AggregateQuery{
		Where: []collections.Condition{
			{Field: "status", Op: collections.Equals, Typed: "completed"},
		},
		Aggregates: []collections.AggregateSpec{
			{Func: collections.AggCount, Alias: "count"},
			{Func: collections.AggSum, Field: "total", Alias: "sum_total"},
		},
	}
	resFlt, err := service.AggregateRecords(ctx, colID, fltAgg)
	if err != nil {
		t.Fatalf("AggregateRecords filtered: %v", err)
	}
	if len(resFlt) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resFlt))
	}
	if count, _ := resFlt[0]["count"].(int64); count != 3 {
		t.Errorf("completed count want 3, got %v", count)
	}
	if sum, _ := resFlt[0]["sum_total"].(float64); sum != 380.0 {
		t.Errorf("completed sum want 380.0, got %v", sum)
	}
}

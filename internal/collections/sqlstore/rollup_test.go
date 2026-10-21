package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// runRollupContract checks, against a real database, that a parent's totals
// follow its children inside one transaction and that a locked parent
// refuses changes to its children.
func runRollupContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	two := 2
	zero := 0
	parent := collections.Schema{ID: "rollup_docs", Label: "Docs",
		Rules: []collections.Rule{{When: "status == 'posted'", Readonly: true, Message: "locked"}},
		Fields: []collections.Field{
			{ID: "status", Label: "Status", Kind: collections.Select, Options: []string{"draft", "posted"}},
			{ID: "lines", Label: "Lines", Kind: collections.Number, Decimals: &zero, Rollup: &collections.Rollup{From: "rollup_lines", Via: "doc"}},
			{ID: "total", Label: "Total", Kind: collections.Number, Decimals: &two, Rollup: &collections.Rollup{From: "rollup_lines", Via: "doc", Sum: "amount"}},
			{ID: "with_tax", Label: "With tax", Kind: collections.Number, Decimals: &two, Computed: "total * 1.2"},
		}}
	child := collections.Schema{ID: "rollup_lines", Label: "Lines", RollupParents: []string{"rollup_docs"},
		Fields: []collections.Field{
			{ID: "doc", Label: "Doc", Kind: collections.Relation, Relation: "rollup_docs", Required: true},
			{ID: "amount", Label: "Amount", Kind: collections.Number, Decimals: &two},
		}}
	for _, schema := range []collections.Schema{parent, child} {
		if err := service.CreateSchema(ctx, schema); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_ = service.DeleteSchema(ctx, child.ID)
		_ = service.DeleteSchema(ctx, parent.ID)
	}()
	total := func(id string) (float64, float64, float64) {
		record, err := service.GetRecord(ctx, parent.ID, id)
		if err != nil {
			t.Fatal(err)
		}
		return record.Data["lines"].(float64), record.Data["total"].(float64), record.Data["with_tax"].(float64)
	}
	doc, err := service.CreateRecord(ctx, parent.ID, map[string]any{"status": "draft"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.CreateRecord(ctx, child.ID, map[string]any{"doc": doc.ID, "amount": 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecord(ctx, child.ID, map[string]any{"doc": doc.ID, "amount": 0.2}); err != nil {
		t.Fatal(err)
	}
	if lines, sum, taxed := total(doc.ID); lines != 2 || sum != 0.3 || taxed != 0.36 {
		t.Fatalf("after two lines: lines=%v total=%v with_tax=%v", lines, sum, taxed)
	}
	if err := service.DeleteRecord(ctx, child.ID, first.ID); err != nil {
		t.Fatal(err)
	}
	if lines, sum, _ := total(doc.ID); lines != 1 || sum != 0.2 {
		t.Fatalf("after delete: lines=%v total=%v", lines, sum)
	}
	// A failed child save changes nothing.
	if _, err := service.CreateRecord(ctx, child.ID, map[string]any{"doc": "missing", "amount": 5}); err == nil {
		t.Fatal("line for a missing parent was saved")
	}
	if _, sum, _ := total(doc.ID); sum != 0.2 {
		t.Fatalf("failed save moved the total to %v", sum)
	}
	// A locked parent refuses changes to its lines.
	if _, err := service.UpdateRecord(ctx, parent.ID, doc.ID, map[string]any{"status": "posted"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecord(ctx, child.ID, map[string]any{"doc": doc.ID, "amount": 1}); !errors.Is(err, collections.ErrRuleViolated) {
		t.Fatalf("line on locked parent: %v", err)
	}
	if _, sum, _ := total(doc.ID); sum != 0.2 {
		t.Fatalf("locked parent total moved to %v", sum)
	}
}

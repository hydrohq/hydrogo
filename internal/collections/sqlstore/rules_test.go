package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestRulesAndStatesOnSQLite(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "rules.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runRulesContract(t, db, database.SQLite)
}

func ordersSchema() collections.Schema {
	two := 2
	return collections.Schema{
		ID: "orders", Label: "Orders",
		Fields: []collections.Field{
			{ID: "status", Label: "Status", Kind: collections.Select, Options: []string{"draft", "approved", "posted", "void"}},
			{ID: "quantity", Label: "Quantity", Kind: collections.Number},
			{ID: "price", Label: "Price", Kind: collections.Number, Decimals: &two},
			{ID: "total", Label: "Total", Kind: collections.Number, Decimals: &two, Computed: "quantity * price"},
			{ID: "note", Label: "Note", Kind: collections.Text},
		},
		Rules: []collections.Rule{
			{Assert: "quantity > 0", Message: "Quantity must be positive"},
			{When: "status == 'posted'", Readonly: true, Message: "Posted orders are locked"},
		},
		States: &collections.StateMachine{Field: "status", Initial: "draft", Transitions: []collections.Transition{
			{From: "draft", To: "approved", Permission: "order.approve"},
			{From: "approved", To: "draft"},
			{From: "approved", To: "posted", When: "total > 0", Message: "Nothing to post"},
			{From: "posted", To: "void"},
		}},
	}
}

func runRulesContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	_ = service.DeleteSchema(ctx, "orders")
	if err := service.CreateSchema(ctx, ordersSchema()); err != nil {
		t.Fatal(err)
	}
	allow := func(permission string) bool { return permission == "order.approve" }
	deny := func(string) bool { return false }

	// Asserts reject a bad record; the state starts at the initial value whatever the client says.
	if _, err := service.CreateRecord(ctx, "orders", map[string]any{"quantity": 0.0, "price": 5.0}); !errors.Is(err, collections.ErrRuleViolated) {
		t.Fatalf("assert not enforced: %v", err)
	}
	order, err := service.CreateRecord(ctx, "orders", map[string]any{"quantity": 2.0, "price": 4.5, "status": "posted"})
	if err != nil {
		t.Fatal(err)
	}
	if order.Data["status"] != "draft" || order.Data["total"] != 9.0 {
		t.Fatalf("new order = %v", order.Data)
	}

	// The state cannot be edited through a normal save.
	edited, err := service.UpdateRecordVersion(ctx, "orders", order.ID, map[string]any{"quantity": 3.0, "price": 4.5, "status": "void", "note": "x"}, order.Version)
	if err != nil || edited.Data["status"] != "draft" || edited.Data["total"] != 13.5 {
		t.Fatalf("update = %v, %v", edited.Data, err)
	}

	// Transitions: unknown step, missing permission, then the allowed path.
	if _, err := service.Transition(ctx, "orders", order.ID, "posted", 0, allow); !errors.Is(err, collections.ErrTransitionNotAllowed) {
		t.Fatalf("draft to posted: %v", err)
	}
	if _, err := service.Transition(ctx, "orders", order.ID, "approved", 0, deny); !errors.Is(err, collections.ErrTransitionForbidden) {
		t.Fatalf("approve without permission: %v", err)
	}
	if _, err := service.Transition(ctx, "orders", order.ID, "approved", 0, nil); !errors.Is(err, collections.ErrTransitionForbidden) {
		t.Fatalf("approve without a permission check: %v", err)
	}
	approved, err := service.Transition(ctx, "orders", order.ID, "approved", edited.Version, allow)
	if err != nil || approved.Data["status"] != "approved" || approved.Version != edited.Version+1 {
		t.Fatalf("approve = %v, %v", approved.Data, err)
	}
	if _, err := service.Transition(ctx, "orders", order.ID, "approved", 1, allow); !errors.Is(err, collections.ErrVersionConflict) && !errors.Is(err, collections.ErrTransitionNotAllowed) {
		t.Fatalf("repeat approve: %v", err)
	}
	posted, err := service.Transition(ctx, "orders", order.ID, "posted", 0, deny)
	if err != nil || posted.Data["status"] != "posted" {
		t.Fatalf("post = %v, %v", posted.Data, err)
	}

	// Posted records are locked: no edit, no delete, no state hijack.
	if _, err := service.UpdateRecord(ctx, "orders", order.ID, map[string]any{"quantity": 9.0, "price": 1.0}); !errors.Is(err, collections.ErrRuleViolated) {
		t.Fatalf("edit of a posted order: %v", err)
	}
	if err := service.DeleteRecord(ctx, "orders", order.ID); !errors.Is(err, collections.ErrRuleViolated) {
		t.Fatalf("delete of a posted order: %v", err)
	}
	// ... but a transition still works.
	voided, err := service.Transition(ctx, "orders", order.ID, "void", 0, deny)
	if err != nil || voided.Data["status"] != "void" {
		t.Fatalf("void = %v, %v", voided.Data, err)
	}

	// A step whose condition fails.
	empty, err := service.CreateRecord(ctx, "orders", map[string]any{"quantity": 1.0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Transition(ctx, "orders", empty.ID, "approved", 0, allow); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Transition(ctx, "orders", empty.ID, "posted", 0, allow); !errors.Is(err, collections.ErrTransitionNotAllowed) {
		t.Fatalf("post with total 0: %v", err)
	}
}

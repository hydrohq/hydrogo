package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// runReferenceContract checks that a record other records point at cannot be
// deleted, says so with ErrRecordInUse rather than a raw database error, and
// can be deleted once nothing points at it.
func runReferenceContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	_ = service.DeleteSchema(ctx, "ref_children")
	_ = service.DeleteSchema(ctx, "ref_parents")
	parents := collections.Schema{ID: "ref_parents", Label: "Parents", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true}}}
	children := collections.Schema{ID: "ref_children", Label: "Children", Fields: []collections.Field{
		{ID: "name", Label: "Name", Kind: collections.Text, Required: true},
		{ID: "parent", Label: "Parent", Kind: collections.Relation, Relation: "ref_parents"},
		{ID: "others", Label: "Others", Kind: collections.RelationMany, Relation: "ref_parents"},
	}}
	for _, schema := range []collections.Schema{parents, children} {
		if err := service.CreateSchema(ctx, schema); err != nil {
			t.Fatal(err)
		}
	}
	single, err := service.CreateRecord(ctx, "ref_parents", map[string]any{"name": "single"})
	if err != nil {
		t.Fatal(err)
	}
	many, err := service.CreateRecord(ctx, "ref_parents", map[string]any{"name": "many"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.CreateRecord(ctx, "ref_children", map[string]any{"name": "child", "parent": single.ID, "others": []any{many.ID}})
	if err != nil {
		t.Fatal(err)
	}
	for name, parent := range map[string]collections.Record{"a single relation": single, "a many relation": many} {
		err := service.DeleteRecord(ctx, "ref_parents", parent.ID)
		if !errors.Is(err, collections.ErrRecordInUse) {
			t.Fatalf("deleting a record referenced through %s: err = %v, want ErrRecordInUse", name, err)
		}
		var inUse *collections.RecordInUseError
		if !errors.As(err, &inUse) {
			t.Fatalf("expected RecordInUseError, got %T: %v", err, err)
		}
		if inUse.RecordID != parent.ID || inUse.ReferencingRecordID != child.ID {
			t.Fatalf("unexpected inUse payload: %+v", inUse)
		}
	}
	if err := service.DeleteRecord(ctx, "ref_children", child.ID); err != nil {
		t.Fatal(err)
	}
	for _, parent := range []collections.Record{single, many} {
		if err := service.DeleteRecord(ctx, "ref_parents", parent.ID); err != nil {
			t.Fatalf("deleting an unreferenced record: %v", err)
		}
	}

	// Test cascade delete
	casParents := collections.Schema{ID: "cas_parents", Label: "CasParents", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text}}}
	casChildren := collections.Schema{ID: "cas_children", Label: "CasChildren", Fields: []collections.Field{
		{ID: "name", Label: "Name", Kind: collections.Text},
		{ID: "parent", Label: "Parent", Kind: collections.Relation, Relation: "cas_parents", OnDelete: "cascade"},
	}}
	if err := service.CreateSchema(ctx, casParents); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateSchema(ctx, casChildren); err != nil {
		t.Fatal(err)
	}
	p1, err := service.CreateRecord(ctx, "cas_parents", map[string]any{"name": "p1"})
	if err != nil {
		t.Fatal(err)
	}
	c1, err := service.CreateRecord(ctx, "cas_children", map[string]any{"name": "c1", "parent": p1.ID})
	if err != nil {
		t.Fatal(err)
	}
	// Deleting p1 should cascade and delete c1
	if err := service.DeleteRecord(ctx, "cas_parents", p1.ID); err != nil {
		t.Fatalf("cascade delete parent: %v", err)
	}
	if _, err := service.GetRecord(ctx, "cas_children", c1.ID); !errors.Is(err, collections.ErrRecordNotFound) {
		t.Fatalf("expected child to be deleted by cascade, got err: %v", err)
	}

	// Test set_null delete
	nullParents := collections.Schema{ID: "null_parents", Label: "NullParents", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text}}}
	nullChildren := collections.Schema{ID: "null_children", Label: "NullChildren", Fields: []collections.Field{
		{ID: "name", Label: "Name", Kind: collections.Text},
		{ID: "parent", Label: "Parent", Kind: collections.Relation, Relation: "null_parents", OnDelete: "set_null"},
		{ID: "others", Label: "Others", Kind: collections.RelationMany, Relation: "null_parents", OnDelete: "set_null"},
	}}
	if err := service.CreateSchema(ctx, nullParents); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateSchema(ctx, nullChildren); err != nil {
		t.Fatal(err)
	}
	np1, err := service.CreateRecord(ctx, "null_parents", map[string]any{"name": "np1"})
	if err != nil {
		t.Fatal(err)
	}
	np2, err := service.CreateRecord(ctx, "null_parents", map[string]any{"name": "np2"})
	if err != nil {
		t.Fatal(err)
	}
	nc1, err := service.CreateRecord(ctx, "null_children", map[string]any{"name": "nc1", "parent": np1.ID, "others": []any{np1.ID, np2.ID}})
	if err != nil {
		t.Fatal(err)
	}
	// Deleting np1 should set parent to null and remove np1 from others in nc1
	if err := service.DeleteRecord(ctx, "null_parents", np1.ID); err != nil {
		t.Fatalf("set_null delete parent: %v", err)
	}
	updatedChild, err := service.GetRecord(ctx, "null_children", nc1.ID)
	if err != nil {
		t.Fatalf("get child after set_null: %v", err)
	}
	if _, ok := updatedChild.Data["parent"]; ok {
		t.Fatalf("expected parent field to be cleared, got: %v", updatedChild.Data["parent"])
	}
	others, ok := updatedChild.Data["others"].([]any)
	if !ok || len(others) != 1 || others[0] != np2.ID {
		t.Fatalf("expected others to have only np2, got: %v", updatedChild.Data["others"])
	}
}

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
		if err := service.DeleteRecord(ctx, "ref_parents", parent.ID); !errors.Is(err, collections.ErrRecordInUse) {
			t.Fatalf("deleting a record referenced through %s: err = %v, want ErrRecordInUse", name, err)
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
}

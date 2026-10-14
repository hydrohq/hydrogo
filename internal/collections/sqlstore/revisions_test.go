package sqlstore

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func runRevisionContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Helper()
	ctx := collections.WithActor(context.Background(), "editor@example.test")
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	schema := collections.Schema{ID: "revised", Label: "Revised", Revisions: true, Fields: []collections.Field{{ID: "title", Label: "Title", Kind: collections.Text, Required: true}}}
	plain := collections.Schema{ID: "unrevised", Label: "Unrevised", Fields: schema.Fields}
	for _, item := range []collections.Schema{schema, plain} {
		if err := service.CreateSchema(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		_ = service.DeleteSchema(ctx, schema.ID)
		_ = service.DeleteSchema(ctx, plain.ID)
	}()
	record, err := service.CreateRecord(ctx, schema.ID, map[string]any{"title": "v1"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.GetRevision(ctx, schema.ID, record.ID, 1)
	if err != nil || first.Data["title"] != "v1" || first.Actor != "editor@example.test" || first.CreatedAt.IsZero() {
		t.Fatalf("%s: first revision=%+v err=%v", dialect, first, err)
	}
	const saves = collections.MaxRevisionsPerRecord + 3
	for version := 2; version <= saves; version++ {
		if _, err := service.UpdateRecordVersion(ctx, schema.ID, record.ID, map[string]any{"title": "v" + strconv.Itoa(version)}, version-1); err != nil {
			t.Fatal(err)
		}
	}
	revisions, err := service.ListRevisions(ctx, schema.ID, record.ID, 100)
	if err != nil || len(revisions) != collections.MaxRevisionsPerRecord {
		t.Fatalf("%s: revisions=%d err=%v", dialect, len(revisions), err)
	}
	if revisions[0].Version != saves || revisions[0].Data["title"] != "v"+strconv.Itoa(saves) || revisions[len(revisions)-1].Version != saves-collections.MaxRevisionsPerRecord+1 {
		t.Fatalf("%s: newest=%d oldest=%d", dialect, revisions[0].Version, revisions[len(revisions)-1].Version)
	}
	if _, err := service.GetRevision(ctx, schema.ID, record.ID, 1); !errors.Is(err, collections.ErrRevisionNotFound) {
		t.Fatalf("%s: pruned revision err=%v", dialect, err)
	}
	if few, err := service.ListRevisions(ctx, schema.ID, record.ID, 2); err != nil || len(few) != 2 {
		t.Fatalf("%s: limit: %d %v", dialect, len(few), err)
	}

	other, err := service.CreateRecord(ctx, plain.ID, map[string]any{"title": "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListRevisions(ctx, plain.ID, other.ID, 10); !errors.Is(err, collections.ErrRevisionNotFound) {
		t.Fatalf("%s: revisions of a schema without them: %v", dialect, err)
	}

	if err := service.DeleteRecord(ctx, schema.ID, record.ID); err != nil {
		t.Fatal(err)
	}
	if left, err := service.ListRevisions(ctx, schema.ID, record.ID, 10); err != nil || len(left) != 0 {
		t.Fatalf("%s: revisions survived their record: %d %v", dialect, len(left), err)
	}
}

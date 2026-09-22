package custom

import (
	"context"
	"database/sql"
	"testing"

	generated "github.com/ardaozyildirim/hydrogo/examples/relation-edition/generated/relation_record"
	_ "modernc.org/sqlite"
)

func relationRepository(t *testing.T) (*sql.DB, generated.RelationRecordRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(generated.MigrationSQL); err != nil {
		t.Fatal(err)
	}
	return db, generated.RelationRecordRepository{DB: db, Dialect: "sqlite"}
}

func TestManyRelationReplacementIsTransactionalAndTenantScoped(t *testing.T) {
	ctx := context.Background()
	db, repository := relationRepository(t)
	if err := repository.Create(ctx, "tenant-a", generated.RelationRecord{ID: "record-1", Title: "Daily report"}); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.ReplaceTagIdsTx(ctx, tx, "tenant-a", "record-1", []string{"tag-b", "tag-a"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	got, err := repository.ListTagIds(ctx, "tenant-a", "record-1")
	if err != nil || len(got) != 2 || got[0] != "tag-b" || got[1] != "tag-a" {
		t.Fatalf("relation IDs = %v, error = %v", got, err)
	}
	otherTenant, err := repository.ListTagIds(ctx, "tenant-b", "record-1")
	if err != nil || len(otherTenant) != 0 {
		t.Fatalf("cross-tenant relation read = %v, error = %v", otherTenant, err)
	}
}

func TestManyRelationRejectsDuplicateAndOversizedInputWithoutReplacing(t *testing.T) {
	ctx := context.Background()
	_, repository := relationRepository(t)
	if err := repository.Create(ctx, "tenant-a", generated.RelationRecord{ID: "record-1", Title: "Daily report"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.ReplaceTagIds(ctx, "tenant-a", "record-1", []string{"tag-a"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.ReplaceTagIds(ctx, "tenant-a", "record-1", []string{"tag-a", "tag-a"}); err == nil {
		t.Fatal("duplicate relation IDs were accepted")
	}
	tooMany := make([]string, 21)
	for i := range tooMany {
		tooMany[i] = "tag-" + string(rune('a'+i))
	}
	if err := repository.ReplaceTagIds(ctx, "tenant-a", "record-1", tooMany); err == nil {
		t.Fatal("relation IDs above the configured maximum were accepted")
	}
	got, err := repository.ListTagIds(ctx, "tenant-a", "record-1")
	if err != nil || len(got) != 1 || got[0] != "tag-a" {
		t.Fatalf("failed replacement changed persisted values: %v, error = %v", got, err)
	}
}

package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSequencesOnSQLite(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "seq.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	runSequenceContract(t, db, database.SQLite)
}

// runSequenceContract checks numbering, rollback and immutability against a
// real database of the given dialect.
func runSequenceContract(t *testing.T, db database.Executor, dialect database.Dialect) {
	t.Run("gapless", func(t *testing.T) { sequenceGapless(t, numberedService(t, db, dialect)) })
	t.Run("immutable", func(t *testing.T) { sequenceImmutable(t, numberedService(t, db, dialect)) })
	t.Run("rollback", func(t *testing.T) { sequenceRollback(t, numberedService(t, db, dialect)) })
}

// numberedService (re)creates an empty invoices type so each case starts at 1.
func numberedService(t *testing.T, db database.Executor, dialect database.Dialect) collections.Service {
	t.Helper()
	ctx := context.Background()
	service := collections.Service{Repository: Store{DB: db, Dialect: dialect}}
	_ = service.DeleteSchema(ctx, "invoices")
	err := service.CreateSchema(ctx, collections.Schema{ID: "invoices", Label: "Invoices", Fields: []collections.Field{
		{ID: "number", Label: "Number", Kind: collections.Text, Sequence: &collections.Sequence{Prefix: "INV-{yyyy}-", Pad: 5}},
		{ID: "customer", Label: "Customer", Kind: collections.Text, Required: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func sequenceGapless(t *testing.T, service collections.Service) {
	ctx := context.Background()
	year := yearScope()
	for want := 1; want <= 3; want++ {
		record, err := service.CreateRecord(ctx, "invoices", map[string]any{"customer": "A", "number": "FAKE"})
		if err != nil {
			t.Fatal(err)
		}
		if got, expected := record.Data["number"], padded(year, want); got != expected {
			t.Fatalf("record %d number = %v, want %s", want, got, expected)
		}
	}
	// A failed create (missing required field) must not use up a number.
	if _, err := service.CreateRecord(ctx, "invoices", map[string]any{}); !errors.Is(err, collections.ErrInvalidRecord) {
		t.Fatalf("expected validation error, got %v", err)
	}
	record, err := service.CreateRecord(ctx, "invoices", map[string]any{"customer": "B"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := record.Data["number"], year+"00004"; got != want {
		t.Fatalf("number after failed create = %v, want %s", got, want)
	}
}

func sequenceImmutable(t *testing.T, service collections.Service) {
	ctx := context.Background()
	created, err := service.CreateRecord(ctx, "invoices", map[string]any{"customer": "A"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateRecordVersion(ctx, "invoices", created.ID, map[string]any{"customer": "B", "number": "HACK-1"}, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Data["number"] != created.Data["number"] || updated.Data["customer"] != "B" {
		t.Fatalf("update changed the number or lost the edit: %v", updated.Data)
	}
}

func sequenceRollback(t *testing.T, service collections.Service) {
	ctx := context.Background()
	boom := errors.New("boom")
	err := service.InTransaction(ctx, func(tx collections.Service) error {
		if _, err := tx.CreateRecord(ctx, "invoices", map[string]any{"customer": "A"}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	record, err := service.CreateRecord(ctx, "invoices", map[string]any{"customer": "B"})
	if err != nil {
		t.Fatal(err)
	}
	if got := record.Data["number"]; got != yearScope()+"00001" {
		t.Fatalf("rolled-back create used up a number: %v", got)
	}
}

func TestSequenceSchemaValidation(t *testing.T) {
	bad := []collections.Field{
		{ID: "n", Label: "N", Kind: collections.Number, Sequence: &collections.Sequence{}},
		{ID: "n", Label: "N", Kind: collections.Text, Sequence: &collections.Sequence{Prefix: "bad prefix"}},
		{ID: "n", Label: "N", Kind: collections.Text, Sequence: &collections.Sequence{Prefix: "{year}"}},
		{ID: "n", Label: "N", Kind: collections.Text, Sequence: &collections.Sequence{Pad: 99}},
	}
	for index, field := range bad {
		if err := (collections.Schema{ID: "x", Label: "X", Fields: []collections.Field{field}}).Validate(); err == nil {
			t.Errorf("case %d accepted", index)
		}
	}
}

func yearScope() string {
	sequence := collections.Sequence{Prefix: "INV-{yyyy}-"}
	return sequence.Scope(time.Now())
}

func padded(scope string, n int) string {
	sequence := collections.Sequence{Pad: 5}
	return sequence.Format(scope, int64(n))
}

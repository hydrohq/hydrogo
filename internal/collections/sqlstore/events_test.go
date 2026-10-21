package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// A project with only core modules (no pages) still gets the outbox, and
// content events are written in the record's transaction.
func TestContentEventsInCoreOnlyDatabase(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "events.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrationsForModules(ctx, db, database.SQLite, migrations.FS, map[string]bool{"core": true, "auth": true, "collections": true}); err != nil {
		t.Fatal(err)
	}
	outbox := sqlstore.Store{Dialect: database.SQLite}
	var fail error
	store := Store{DB: db, Dialect: database.SQLite}
	store.Events = func(ctx context.Context, tx database.Executor, topic, aggregate string, version int) error {
		if fail != nil {
			return fail
		}
		return outbox.Enqueue(ctx, tx, topic+"-"+aggregate+"-"+string(rune('0'+version)), topic, aggregate, 1)
	}
	service := collections.Service{Repository: store}
	if err := service.CreateSchema(ctx, collections.Schema{ID: "notes", Label: "Notes", Fields: []collections.Field{{ID: "title", Label: "Title", Kind: collections.Text}}}); err != nil {
		t.Fatal(err)
	}
	topics := func() []string {
		rows, err := db.QueryContext(ctx, "SELECT topic || ' ' || aggregate_id FROM outbox_events ORDER BY topic, aggregate_id")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var result []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				t.Fatal(err)
			}
			result = append(result, s)
		}
		return result
	}
	record, err := service.CreateRecord(ctx, "notes", map[string]any{"title": "a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateRecord(ctx, "notes", record.ID, map[string]any{"title": "b"}); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteRecord(ctx, "notes", record.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{"content.created notes/" + record.ID, "content.deleted notes/" + record.ID, "content.updated notes/" + record.ID}
	if got := topics(); len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("events = %v, want %v", got, want)
	}
	// A failing event writer rolls the record back.
	fail = errors.New("outbox down")
	if _, err := service.CreateRecord(ctx, "notes", map[string]any{"title": "c"}); err == nil {
		t.Fatal("record saved without its event")
	}
	rows, err := service.QueryRecords(ctx, "notes", collections.Query{Limit: 10})
	if err != nil || len(rows) != 0 {
		t.Fatalf("rows = %v err=%v", rows, err)
	}
}

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

func TestSyncSchemasCreatesUpdatesAndProtectsStoredRecords(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "sync.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	service := collections.Service{Repository: Store{DB: db, Dialect: database.SQLite}}

	team := collections.Schema{ID: "team", Label: "Team", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true}}}
	// faq relates to team and is listed first, so sync must order by relations.
	faq := collections.Schema{ID: "faq", Label: "FAQ", Fields: []collections.Field{
		{ID: "question", Label: "Question", Kind: collections.Text, Required: true},
		{ID: "owner", Label: "Owner", Kind: collections.Relation, Relation: "team"},
	}}
	if err := service.SyncSchemas(ctx, []collections.Schema{faq, team}); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	if err := service.SyncSchemas(ctx, []collections.Schema{faq, team}); err != nil {
		t.Fatalf("idempotent sync: %v", err)
	}

	faq.Label = "Questions"
	faq.Fields = append(faq.Fields, collections.Field{ID: "answer", Label: "Answer", Kind: collections.RichText})
	if err := service.SyncSchemas(ctx, []collections.Schema{faq, team}); err != nil {
		t.Fatalf("additive change: %v", err)
	}
	stored, err := service.GetSchema(ctx, "faq")
	if err != nil || stored.Label != "Questions" || len(stored.Fields) != 3 {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}

	faq.Fields[0].Kind = collections.Number
	if err := service.SyncSchemas(ctx, []collections.Schema{faq, team}); !errors.Is(err, collections.ErrInvalidSchema) {
		t.Fatalf("kind change error=%v, want ErrInvalidSchema", err)
	}

	loop := collections.Schema{ID: "a", Label: "A", Fields: []collections.Field{{ID: "b", Label: "B", Kind: collections.Relation, Relation: "b"}}}
	back := collections.Schema{ID: "b", Label: "B", Fields: []collections.Field{{ID: "a", Label: "A", Kind: collections.Relation, Relation: "a"}}}
	if err := service.SyncSchemas(ctx, []collections.Schema{loop, back}); !errors.Is(err, collections.ErrInvalidSchema) {
		t.Fatalf("cycle error=%v, want ErrInvalidSchema", err)
	}
}

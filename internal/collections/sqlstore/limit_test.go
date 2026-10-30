package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestRecordCapIsConfigurableAndCanBeRemoved(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "limit.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	schema := collections.Schema{ID: "notes", Label: "Notes", Fields: []collections.Field{{ID: "title", Label: "Title", Kind: collections.Text, Required: true}}}
	for _, test := range []struct {
		name  string
		limit int
		want  int
	}{
		{"cap of three", 3, 3},
		{"no cap", -1, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := collections.Service{Repository: Store{DB: db, Dialect: database.SQLite, MaxRecords: test.limit}}
			schema := schema
			schema.ID = "notes_" + strconv.Itoa(test.limit+1)
			if err := service.CreateSchema(ctx, schema); err != nil {
				t.Fatal(err)
			}
			created := 0
			for i := 0; i < 5; i++ {
				_, err := service.CreateRecord(ctx, schema.ID, map[string]any{"title": "n" + strconv.Itoa(i)})
				if err != nil {
					if !errors.Is(err, collections.ErrInvalidRecord) {
						t.Fatalf("record %d: %v", i, err)
					}
					break
				}
				created++
			}
			if created != test.want {
				t.Fatalf("created %d records, want %d", created, test.want)
			}
		})
	}
	if (Store{}).recordCap() != collections.DefaultMaxRecords {
		t.Fatalf("zero MaxRecords must use the default cap")
	}
}

// runIndexContract checks that PostgreSQL gets one index per Indexed field and
// that an index is dropped when its field stops being indexed.
func runIndexContract(t *testing.T, db database.Executor) {
	t.Helper()
	ctx := context.Background()
	store := Store{DB: db, Dialect: database.Postgres}
	service := collections.Service{Repository: store}
	schema := collections.Schema{ID: "indexed_deals", Label: "Deals", Fields: []collections.Field{
		{ID: "name", Label: "Name", Kind: collections.Text, Required: true},
		{ID: "stage", Label: "Stage", Kind: collections.Select, Options: []string{"lead", "won"}, Indexed: true},
		{ID: "amount", Label: "Amount", Kind: collections.Number, Indexed: true},
		{ID: "tags", Label: "Tags", Kind: collections.MultiSelect, Options: []string{"a"}, Indexed: true},
		{ID: "notes", Label: "Notes", Kind: collections.Text, Indexed: true},
	}}
	if err := service.SyncSchemas(ctx, []collections.Schema{schema}); err != nil {
		t.Fatal(err)
	}
	names := func() []string {
		t.Helper()
		got, err := store.managedIndexes(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	count := func() int { return len(names()) }
	before := count()
	// stage (btree) + amount (btree) + notes (btree + trigram); tags has no
	// scalar representation to index.
	if before < 4 {
		t.Fatalf("expected indexes for stage, amount and notes (btree+trigram), found %d managed indexes", before)
	}
	hasTrigramIndex := false
	for _, name := range names() {
		if strings.Contains(name, "_trgm_") {
			hasTrigramIndex = true
			break
		}
	}
	if !hasTrigramIndex {
		t.Fatal("expected a trigram index for the indexed text field \"notes\"")
	}
	// A second sync changes nothing.
	if err := service.SyncSchemas(ctx, []collections.Schema{schema}); err != nil || count() != before {
		t.Fatalf("second sync: err=%v indexes=%d want %d", err, count(), before)
	}
	next := schema
	next.Fields = append([]collections.Field(nil), schema.Fields...)
	next.Fields[2].Indexed = false
	if err := service.SyncSchemas(ctx, []collections.Schema{next}); err != nil {
		t.Fatal(err)
	}
	if got := count(); got != before-1 {
		t.Fatalf("dropping an indexed flag left %d indexes, want %d", got, before-1)
	}
	if _, err := service.CreateRecord(ctx, "indexed_deals", map[string]any{"name": "x", "stage": "won", "amount": 5}); err != nil {
		t.Fatal(err)
	}
	if defs, _ := indexStatements(database.Postgres, "indexed_deals", collections.Field{ID: "x'; DROP TABLE y; --", Kind: collections.Text}, true); len(defs) != 0 {
		t.Fatal("an unsafe field id produced an index name")
	}
}

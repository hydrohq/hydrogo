package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
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

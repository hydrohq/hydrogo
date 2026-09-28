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

func TestStaticStoreSupportsSchemasUniqueValuesRelationsAndCRUD(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "collections.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	service := collections.Service{Repository: Store{DB: db, Dialect: database.SQLite}}
	topics := collections.Schema{ID: "topics", Label: "Topics", Fields: []collections.Field{{ID: "name", Label: "Name", Kind: collections.Text, Required: true, Unique: true}}}
	if err := service.CreateSchema(ctx, topics); err != nil {
		t.Fatal(err)
	}
	topic, err := service.CreateRecord(ctx, topics.ID, map[string]any{"name": "Hydro"})
	if err != nil {
		t.Fatal(err)
	}
	articles := collections.Schema{ID: "articles", Label: "Articles", Fields: []collections.Field{{ID: "slug", Label: "Slug", Kind: collections.Text, Required: true, Unique: true}, {ID: "topic", Label: "Topic", Kind: collections.Relation, Relation: topics.ID}}}
	if err := service.CreateSchema(ctx, articles); err != nil {
		t.Fatal(err)
	}
	article, err := service.CreateRecord(ctx, articles.ID, map[string]any{"slug": "one", "topic": topic.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRecord(ctx, articles.ID, map[string]any{"slug": "one", "topic": topic.ID}); !errors.Is(err, collections.ErrUniqueValue) {
		t.Fatalf("duplicate unique value error=%v", err)
	}
	if err := service.DeleteRecord(ctx, topics.ID, topic.ID); err == nil {
		t.Fatal("target record with a relation was deleted")
	}
	updated, err := service.UpdateRecord(ctx, articles.ID, article.ID, map[string]any{"slug": "two", "topic": topic.ID})
	if err != nil || updated.Version != 2 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	list, err := service.ListRecords(ctx, articles.ID, 10, 0)
	if err != nil || len(list) != 1 || list[0].ID != article.ID {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	if err := service.DeleteRecord(ctx, articles.ID, article.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteRecord(ctx, topics.ID, topic.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteSchema(ctx, topics.ID); err == nil {
		t.Fatal("schema with incoming relation was deleted")
	}
	if err := service.DeleteSchema(ctx, articles.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteSchema(ctx, topics.ID); err != nil {
		t.Fatal(err)
	}
}

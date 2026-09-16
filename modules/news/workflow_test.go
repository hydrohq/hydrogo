package news_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
)

func TestServiceUsesConfiguredEditorialWorkflow(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "workflow.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: database.SQLite}
	item := news.Item{ID: "workflow-1", Locale: "en", Slug: "workflow-1", Title: "Workflow", Body: "{}", Status: news.StatusDraft}
	if err := store.CreateDraft(ctx, db, item); err != nil {
		t.Fatal(err)
	}
	service := news.Service{
		Transactions: database.NewTransactor(db),
		Repository:   store,
		Workflow: v1.EditorialWorkflow{
			States:      []v1.EditorialState{v1.EditorialDraft, v1.EditorialPublished},
			Transitions: map[v1.EditorialState][]v1.EditorialState{v1.EditorialDraft: {}},
		},
		WorkflowRole: func(context.Context) string { return "editor" },
	}
	if err := service.Publish(ctx, item.ID, 1); !errors.Is(err, news.ErrInvalidWorkflowTransition) {
		t.Fatalf("publish error = %v, want ErrInvalidWorkflowTransition", err)
	}
	current, err := store.Get(ctx, db, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != news.StatusDraft {
		t.Fatalf("status after rejected transition = %s", current.Status)
	}
}

func TestServiceEnforcesRoleSpecificEditorialWorkflow(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "workflow-role.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: database.SQLite}
	item := news.Item{ID: "workflow-role-1", Locale: "en", Slug: "workflow-role-1", Title: "Workflow", Body: "{}", Status: news.StatusDraft}
	if err := store.CreateDraft(ctx, db, item); err != nil {
		t.Fatal(err)
	}
	service := news.Service{Transactions: database.NewTransactor(db), Repository: store, WorkflowRole: func(context.Context) string { return "editor" }, Workflow: v1.EditorialWorkflow{
		States:          []v1.EditorialState{v1.EditorialDraft, v1.EditorialPublished},
		Transitions:     map[v1.EditorialState][]v1.EditorialState{v1.EditorialDraft: {v1.EditorialPublished}},
		RoleTransitions: map[string]map[v1.EditorialState][]v1.EditorialState{"editor": {v1.EditorialDraft: {}}},
	}}
	if err := service.Publish(ctx, item.ID, 1); !errors.Is(err, news.ErrInvalidWorkflowTransition) {
		t.Fatalf("publish error=%v", err)
	}
}

package site_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	"github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
)

func TestSubmitIdempotentReplaysAndRejectsChangedPayload(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "site.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(context.Background(), db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	service := site.Service{Transactions: database.NewTransactor(db), Store: sqlstore.Store{Dialect: database.SQLite}}
	value := site.Submission{ID: "form-1", FormKey: "contact", Email: "user@example.com", Payload: `{"message":"hello"}`, SourceIPHash: "hash", IdempotencyKey: "0123456789abcdef", RequestHash: "request-1"}
	if replayed, err := service.SubmitIdempotent(context.Background(), value); err != nil || replayed {
		t.Fatalf("first submit replayed=%v err=%v", replayed, err)
	}
	value.ID = "form-2"
	if replayed, err := service.SubmitIdempotent(context.Background(), value); err != nil || !replayed {
		t.Fatalf("retry replayed=%v err=%v", replayed, err)
	}
	value.RequestHash = "request-2"
	if _, err := service.SubmitIdempotent(context.Background(), value); err != site.ErrIdempotencyConflict {
		t.Fatalf("changed payload error=%v", err)
	}
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM form_inbox WHERE form_key = 'contact'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored submissions = %d, want 1", count)
	}
}

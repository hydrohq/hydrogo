package sqlstore

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestRecordPersistsRequestIDFromContext(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:audit-request-id?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	requestCtx := security.WithRequestID(ctx, "req-audit-123")
	store := Store{Dialect: database.SQLite}
	if err := database.NewTransactor(db).Within(requestCtx, func(tx database.Executor) error {
		return store.Record(requestCtx, tx, "audit-1", "auth.login", "user-1")
	}); err != nil {
		t.Fatal(err)
	}
	var got sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT request_id FROM audit_events WHERE id = ?", "audit-1").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Valid || got.String != "req-audit-123" {
		t.Fatalf("request_id = %#v", got)
	}
}

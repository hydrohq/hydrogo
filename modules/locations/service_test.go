package locations_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/locations"
	"github.com/ardaozyildirim/hydrogo/modules/locations/repository/sqlstore"
)

func TestServiceAuditsMutationsAndRejectsStaleEditors(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "locations.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	service := locations.Service{Transactions: database.NewTransactor(db), Repository: sqlstore.New(database.SQLite), Audit: auditstore.Store{Dialect: database.SQLite}}
	item := locations.Location{Base: sectoral.Base{ID: "loc-1", Locale: "en", Slug: "branch", Title: "Branch"}, Address: "1 Main Street", MapURL: "https://maps.example.test/branch"}
	if err := service.Create(ctx, item); err != nil {
		t.Fatal(err)
	}
	first, err := service.Get(ctx, db, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 {
		t.Fatalf("created version=%d", first.Version)
	}
	stale := first
	updated := first
	updated.Title = "Updated"
	if err := service.Update(ctx, updated); err != nil {
		t.Fatal(err)
	}
	if err := service.Update(ctx, stale); !errors.Is(err, sectoral.ErrConflict) {
		t.Fatalf("stale update=%v", err)
	}
	var auditCount int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE target_id=?", item.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 2 {
		t.Fatalf("audit events=%d, want create and update only", auditCount)
	}
	if err := service.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM audit_events WHERE target_id=?", item.ID).Scan(&auditCount); err != nil || auditCount != 3 {
		t.Fatalf("delete audit count=%d err=%v", auditCount, err)
	}
}

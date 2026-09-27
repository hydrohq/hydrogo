package sqlstore

import (
	"context"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestSQLiteBanStoreAndImmutableAuditMigration(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "security.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := BanStore{Dialect: database.SQLite}
	prefix := netip.MustParsePrefix("192.0.2.1/32")
	if err := store.Put(ctx, db, prefix, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListActive(ctx, db, time.Now(), 10)
	if err != nil || len(items) != 1 || items[0].Prefix != prefix {
		t.Fatalf("ban list = %#v, %v", items, err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO audit_events(id,action,target_id) VALUES('audit-immutable','test','target')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE audit_events SET action='changed' WHERE id='audit-immutable'`); err == nil {
		t.Fatal("audit update was allowed")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM audit_events WHERE id='audit-immutable'`); err == nil {
		t.Fatal("audit delete was allowed")
	}
	if err := store.Remove(ctx, db, prefix); err != nil {
		t.Fatal(err)
	}
}

package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func chainDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	SetChain(true)
	t.Cleanup(func() { SetChain(false) })
	return db
}

func TestAuditChainDetectsTampering(t *testing.T) {
	ctx := context.Background()
	db := chainDB(t, "audit-chain-tamper")
	store := Store{Dialect: database.SQLite}
	// A legacy event from before the chain, then three chained ones, some
	// written on a bare connection and some inside a transaction.
	SetChain(false)
	if err := store.Record(ctx, db, "legacy", "auth.login", "u1"); err != nil {
		t.Fatal(err)
	}
	SetChain(true)
	for i := 1; i <= 2; i++ {
		if err := store.RecordChange(ctx, db, fmt.Sprintf("e%d", i), "content.updated", "r1", []byte(`{"v":1}`), []byte(`{"v":2}`)); err != nil {
			t.Fatal(err)
		}
	}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		return store.Record(ctx, tx, "e3", "auth.login", "u1")
	}); err != nil {
		t.Fatal(err)
	}
	report, err := store.VerifyChain(ctx, db, "", nil)
	if err != nil || len(report.Problems) != 0 || report.Links != 3 || report.Legacy != 1 {
		t.Fatalf("intact chain: %+v err=%v", report, err)
	}
	heads, _ := store.ChainHeads(ctx, db)
	if len(heads) != 1 || heads[0].Seq != 3 {
		t.Fatalf("heads = %+v", heads)
	}
	anchor := heads[0]

	// The log itself refuses edits; a database administrator could lift that,
	// so check what verification says when they do.
	if _, err := db.Exec("UPDATE audit_events SET after_json = '{}' WHERE id = 'e2'"); err == nil {
		t.Fatal("audit events accepted an update")
	}
	for _, drop := range []string{"audit_events_immutable_update", "audit_events_immutable_delete"} {
		if _, err := db.Exec("DROP TRIGGER " + drop); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("UPDATE audit_events SET after_json = '{\"v\":99}' WHERE id = 'e2'"); err != nil {
		t.Fatal(err)
	}
	if report, _ = store.VerifyChain(ctx, db, "", &anchor); len(report.Problems) != 1 || !strings.Contains(report.Problems[0], "link 2") || !strings.Contains(report.Problems[0], "changed") {
		t.Fatalf("changed event: %+v", report.Problems)
	}
	if _, err := db.Exec("DELETE FROM audit_events WHERE id = 'e3'"); err != nil {
		t.Fatal(err)
	}
	if report, _ = store.VerifyChain(ctx, db, "", nil); !strings.Contains(strings.Join(report.Problems, "|"), "is gone") {
		t.Fatalf("deleted event: %+v", report.Problems)
	}
	// Rewriting the chain consistently is caught by the anchor.
	if _, err := db.Exec("DROP TRIGGER audit_chain_immutable_update"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE audit_chain_heads SET seq = 1"); err != nil {
		t.Fatal(err)
	}
	if report, _ = store.VerifyChain(ctx, db, "", &anchor); !strings.Contains(strings.Join(report.Problems, "|"), "head") {
		t.Fatalf("moved head: %+v", report.Problems)
	}
}

func TestAuditChainKeepsTenantsApartAndSerializesWriters(t *testing.T) {
	ctx := context.Background()
	db := chainDB(t, "audit-chain-tenants")
	store := Store{Dialect: database.SQLite, RequireTenantScope: true}
	var scopes []tenant.Scope
	for _, id := range []string{"alpha", "beta"} {
		scope, err := tenant.NewScope("tenant-"+id, id)
		if err != nil {
			t.Fatal(err)
		}
		scopes = append(scopes, scope)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for n := 0; n < 20; n++ {
		for _, scope := range scopes {
			wg.Add(1)
			go func(n int, scope tenant.Scope) {
				defer wg.Done()
				scoped := tenant.WithScope(ctx, scope)
				errs <- database.NewTransactor(db).Within(scoped, func(tx database.Executor) error {
					return store.Record(scoped, tx, fmt.Sprintf("%s-%d", scope.TenantID, n), "content.updated", "r")
				})
			}(n, scope)
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil && !strings.Contains(err.Error(), "locked") && !strings.Contains(err.Error(), "busy") {
			t.Fatal(err)
		}
	}
	for _, scope := range scopes {
		report, err := store.VerifyChain(ctx, db, scope.TenantID, nil)
		if err != nil || len(report.Problems) != 0 || report.Links == 0 {
			t.Fatalf("%s: %+v err=%v", scope.TenantID, report, err)
		}
	}
}

func TestAuditChainRelinkAfterGap(t *testing.T) {
	ctx := context.Background()
	db := chainDB(t, "audit-chain-relink")
	store := Store{Dialect: database.SQLite}
	if err := store.Record(ctx, db, "a1", "auth.login", "u1"); err != nil {
		t.Fatal(err)
	}
	SetChain(false)
	for _, id := range []string{"gap1", "gap2"} {
		if err := store.Record(ctx, db, id, "auth.login", "u1"); err != nil {
			t.Fatal(err)
		}
	}
	SetChain(true)
	if report, _ := store.VerifyChain(ctx, db, "", nil); len(report.Problems) == 0 {
		t.Fatal("a gap in the chain went unnoticed")
	}
	linked, err := store.Relink(ctx, db, "")
	if err != nil || linked != 2 {
		t.Fatalf("relink = %d, %v", linked, err)
	}
	if report, err := store.VerifyChain(ctx, db, "", nil); err != nil || len(report.Problems) != 0 || report.Links != 3 {
		t.Fatalf("after relink: %+v err=%v", report, err)
	}
	if !ChainEnabled() {
		t.Fatal("relink left chaining switched off")
	}
}

// On Windows consecutive events used to carry the same timestamp, and the last
// event before chaining then looked like a missing link. Stamps are strictly
// increasing even when the clock does not move, so with a frozen clock an intact
// chain verifies, and a really later unlinked event is still reported.
func TestAuditChainIsIntactWhenTheClockDoesNotMove(t *testing.T) {
	ctx := context.Background()
	db := chainDB(t, "audit-chain-frozen-clock")
	store := Store{Dialect: database.SQLite}
	instant := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	wallClock = func() time.Time { return instant }
	t.Cleanup(func() { wallClock = time.Now })

	first, second := nowUTC(), nowUTC()
	if !second.After(first) {
		t.Fatalf("two stamps from a frozen clock must differ: %v %v", first, second)
	}

	SetChain(false)
	if err := store.Record(ctx, db, "legacy", "auth.login", "u1"); err != nil {
		t.Fatal(err)
	}
	SetChain(true)
	for i := 1; i <= 2; i++ {
		if err := store.RecordChange(ctx, db, fmt.Sprintf("e%d", i), "content.updated", "r1", []byte(`{"v":1}`), []byte(`{"v":2}`)); err != nil {
			t.Fatal(err)
		}
	}
	report, err := store.VerifyChain(ctx, db, "", nil)
	if err != nil || len(report.Problems) != 0 || report.Links != 2 || report.Legacy != 1 {
		t.Fatalf("frozen clock: %+v err=%v", report, err)
	}

	SetChain(false)
	if err := store.Record(ctx, db, "late", "auth.login", "u1"); err != nil {
		t.Fatal(err)
	}
	SetChain(true)
	report, err = store.VerifyChain(ctx, db, "", nil)
	if err != nil || len(report.Problems) != 1 || !strings.Contains(report.Problems[0], "not in it") {
		t.Fatalf("a later unlinked event must be reported: %+v err=%v", report, err)
	}
}

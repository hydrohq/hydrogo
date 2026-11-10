package sqlstore_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

func TestRefundRequiredStoreDeduplicationAndResolution(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "refund-required-test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	store := sqlstore.Store{Dialect: database.SQLite}

	item := commerce.RefundRequired{
		ID:          "rrq-1",
		OrderID:     "order-dup-1",
		Provider:    "paytr",
		AmountCents: 15000,
		Currency:    "TRY",
		Reason:      "currency mismatch",
		CreatedAt:   time.Now().UTC(),
	}

	// 1. Insert first time -> succeeds
	if err := store.RecordRefundRequired(ctx, db, item); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// 2. Insert duplicate (same order_id + provider, different ID) -> must not error and not duplicate
	itemDup := item
	itemDup.ID = "rrq-2"
	if err := store.RecordRefundRequired(ctx, db, itemDup); err != nil {
		t.Fatalf("duplicate insert returned error: %v", err)
	}

	// Verify only 1 row exists
	list, err := store.ListRefundRequired(ctx, db, false, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 row, got %d", len(list))
	}
	if list[0].ID != "rrq-1" || list[0].ResolvedAt != nil {
		t.Fatalf("unexpected row data: %+v", list[0])
	}

	// 3. Resolve the item
	resolved, err := store.ResolveRefundRequired(ctx, db, "rrq-1", "resolved manually via bank", time.Now().UTC())
	if err != nil || !resolved {
		t.Fatalf("failed to resolve: %v, resolved=%v", err, resolved)
	}

	// 4. Repeated resolve should return false (already resolved)
	repeat, err := store.ResolveRefundRequired(ctx, db, "rrq-1", "again", time.Now().UTC())
	if err != nil || repeat {
		t.Fatalf("expected repeat resolve=false, got %v (err: %v)", repeat, err)
	}

	// 5. Unresolved filter should now be empty
	unresolved, err := store.ListRefundRequired(ctx, db, true, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(unresolved) != 0 {
		t.Fatalf("expected 0 unresolved items, got %d", len(unresolved))
	}
}

func TestCustomerSMSDedupStore(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "sms-dedup-test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	store := sqlstore.Store{Dialect: database.SQLite}
	now := time.Now().UTC()

	// 1. First send: must record and return true
	inserted, err := store.RecordSMSDedup(ctx, db, "sms_order_paid_ord_1", "ord-1", "+905551234567", "order_paid", now)
	if err != nil {
		t.Fatalf("first sms dedup record error: %v", err)
	}
	if !inserted {
		t.Fatal("expected first sms dedup insert to return true, got false")
	}

	// 2. Second send with identical dedup key: must return false without error
	duplicate, err := store.RecordSMSDedup(ctx, db, "sms_order_paid_ord_1", "ord-1", "+905551234567", "order_paid", now)
	if err != nil {
		t.Fatalf("duplicate sms dedup record error: %v", err)
	}
	if duplicate {
		t.Fatal("expected duplicate sms dedup insert to return false, got true")
	}

	// 3. Different dedup key (e.g. for shipment event on same order): must succeed
	shipmentSMS, err := store.RecordSMSDedup(ctx, db, "sms_shipment_ord_1", "ord-1", "+905551234567", "shipment", now)
	if err != nil {
		t.Fatalf("different event sms dedup error: %v", err)
	}
	if !shipmentSMS {
		t.Fatal("expected different key sms dedup insert to return true, got false")
	}
}

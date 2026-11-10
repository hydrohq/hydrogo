package transport

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type mockOrderLookupRepo struct {
	commerce.Repository
	getOrderFn func(ctx context.Context, orderID string) (commerce.Order, error)
}

func (m mockOrderLookupRepo) GetOrder(ctx context.Context, _ database.Executor, orderID string) (commerce.Order, error) {
	if m.getOrderFn != nil {
		return m.getOrderFn(ctx, orderID)
	}
	return commerce.Order{}, errors.New("order not found")
}

func TestPendingTrackerCountPendingRetainsOrdersOnServiceError(t *testing.T) {
	h, _ := newTestHandler(t)
	svc := commerce.Service{
		Transactions: h.Service.Transactions,
		Repository: mockOrderLookupRepo{
			getOrderFn: func(ctx context.Context, orderID string) (commerce.Order, error) {
				return commerce.Order{}, errors.New("transient database connection error")
			},
		},
	}

	tracker := NewPendingOrderTracker()
	token, ok := tracker.TryReserve("err-ip", 5)
	if !ok {
		t.Fatal("failed to reserve slot")
	}
	tracker.Confirm("err-ip", token, "order-err-1")

	// Even though svc.Order fails, the order must still be counted as pending (fail-closed).
	count := tracker.CountPending(context.Background(), "err-ip", svc, time.Now(), time.Minute)
	if count != 1 {
		t.Fatalf("expected count 1 on service error, got %d", count)
	}

	// Context cancellation must also be treated as pending (fail-closed).
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	countCanceled := tracker.CountPending(canceledCtx, "err-ip", svc, time.Now(), time.Minute)
	if countCanceled != 1 {
		t.Fatalf("expected count 1 on canceled context, got %d", countCanceled)
	}
}

func TestPendingTrackerPreservesReservationsAddedOrConfirmedDuringCheck(t *testing.T) {
	h, _ := newTestHandler(t)
	entered := make(chan struct{})
	release := make(chan struct{})

	svc := commerce.Service{
		Transactions: h.Service.Transactions,
		Repository: mockOrderLookupRepo{
			getOrderFn: func(ctx context.Context, orderID string) (commerce.Order, error) {
				if orderID == "order-slow" {
					close(entered)
					<-release
					return commerce.Order{PaymentStatus: commerce.PaymentPending}, nil
				}
				return commerce.Order{PaymentStatus: commerce.PaymentPending}, nil
			},
		},
	}

	tracker := NewPendingOrderTracker()
	// Initial reservation and confirmed order
	slowToken, ok := tracker.TryReserve("active-ip", 10)
	if !ok {
		t.Fatal("failed to reserve initial slot")
	}
	tracker.Confirm("active-ip", slowToken, "order-slow")

	// Another reservation made before CountPending, but without orderID
	unconfirmedToken, ok := tracker.TryReserve("active-ip", 10)
	if !ok {
		t.Fatal("failed to reserve unconfirmed slot")
	}

	var count int
	done := make(chan struct{})
	go func() {
		count = tracker.CountPending(context.Background(), "active-ip", svc, time.Now(), time.Minute)
		close(done)
	}()

	// Wait until CountPending is inside svc.Order for "order-slow"
	<-entered

	// While CountPending is in-flight:
	// 1. Confirm the pre-existing reservation with a new order ID
	tracker.Confirm("active-ip", unconfirmedToken, "order-confirmed-midway")

	// 2. Add an entirely new reservation
	newToken, ok := tracker.TryReserve("active-ip", 10)
	if !ok {
		t.Fatal("failed to reserve new slot during check")
	}
	tracker.Confirm("active-ip", newToken, "order-new")

	close(release)
	<-done

	// All 3 entries (order-slow, unconfirmedToken->order-confirmed-midway, newToken->order-new) must be preserved.
	if count != 3 {
		t.Fatalf("expected 3 pending orders, got %d", count)
	}

	tracker.mu.Lock()
	orders := tracker.orders["active-ip"]
	tracker.mu.Unlock()

	if len(orders) != 3 {
		t.Fatalf("expected 3 orders stored, got %d", len(orders))
	}

	foundConfirmed := false
	foundNew := false
	for _, o := range orders {
		if o.reservation == unconfirmedToken && o.orderID == "order-confirmed-midway" {
			foundConfirmed = true
		}
		if o.reservation == newToken && o.orderID == "order-new" {
			foundNew = true
		}
	}
	if !foundConfirmed {
		t.Fatal("reservation confirmed during check was lost or not updated")
	}
	if !foundNew {
		t.Fatal("reservation added during check was lost")
	}
}

func TestPendingTrackerPaidOrderDropsFromCount(t *testing.T) {
	h, _ := newTestHandler(t)
	svc := commerce.Service{
		Transactions: h.Service.Transactions,
		Repository: mockOrderLookupRepo{
			getOrderFn: func(ctx context.Context, orderID string) (commerce.Order, error) {
				if orderID == "order-paid" {
					return commerce.Order{PaymentStatus: commerce.PaymentPaid}, nil
				}
				if orderID == "order-pending" {
					return commerce.Order{PaymentStatus: commerce.PaymentPending}, nil
				}
				return commerce.Order{}, errors.New("unknown")
			},
		},
	}

	tracker := NewPendingOrderTracker()
	tokPaid, _ := tracker.TryReserve("paid-ip", 5)
	tracker.Confirm("paid-ip", tokPaid, "order-paid")

	tokPending, _ := tracker.TryReserve("paid-ip", 5)
	tracker.Confirm("paid-ip", tokPending, "order-pending")

	count := tracker.CountPending(context.Background(), "paid-ip", svc, time.Now(), time.Minute)
	if count != 1 {
		t.Fatalf("expected count 1 after paid order dropped, got %d", count)
	}

	tracker.mu.Lock()
	orders := tracker.orders["paid-ip"]
	tracker.mu.Unlock()

	if len(orders) != 1 || orders[0].reservation != tokPending {
		t.Fatalf("paid order was not properly pruned: %+v", orders)
	}

	// Now pay the second order as well
	svcAllPaid := commerce.Service{
		Transactions: h.Service.Transactions,
		Repository: mockOrderLookupRepo{
			getOrderFn: func(ctx context.Context, orderID string) (commerce.Order, error) {
				return commerce.Order{PaymentStatus: commerce.PaymentPaid}, nil
			},
		},
	}

	countZero := tracker.CountPending(context.Background(), "paid-ip", svcAllPaid, time.Now(), time.Minute)
	if countZero != 0 {
		t.Fatalf("expected count 0 when all orders paid, got %d", countZero)
	}

	tracker.mu.Lock()
	_, exists := tracker.orders["paid-ip"]
	tracker.mu.Unlock()
	if exists {
		t.Fatal("expected paid-ip to be deleted from orders map when count is 0")
	}
}

package commerce_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

type fakeRefunder struct {
	err   error
	calls int
	got   commerce.PaymentCharge
	order commerce.Order
}

func (f *fakeRefunder) Refund(_ context.Context, o commerce.Order, c commerce.PaymentCharge) error {
	f.calls++
	f.got = c
	f.order = o
	return f.err
}

func paidOrder(t *testing.T, key string) (commerce.Service, commerce.Order, context.Context) {
	t.Helper()
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-refund-"+key, 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-refund-"+key, 1, 5000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-refund-" + key})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "Gateway2", EventID: order.ID, ProviderPaymentID: "pay-77", OrderID: order.ID, AmountCents: order.TotalCents}); err != nil {
		t.Fatal(err)
	}
	return svc, order, ctx
}

func TestRefundOrderReturnsMoneyOnceAndRecordsIt(t *testing.T) {
	svc, order, ctx := paidOrder(t, "ok")
	refunder := &fakeRefunder{}
	got, err := svc.RefundOrder(ctx, refunder, order.ID)
	if err != nil || got.PaymentStatus != commerce.PaymentRefunded {
		t.Fatalf("got %+v, %v", got, err)
	}
	if refunder.calls != 1 || refunder.got.Provider != "Gateway2" || refunder.got.ProviderPaymentID != "pay-77" {
		t.Fatalf("the provider must get the recorded charge: %+v (%d calls)", refunder.got, refunder.calls)
	}
	if refunder.order.IdempotencyKey != "ref_"+order.ID {
		t.Fatalf("idempotency key = %q, want 'ref_%s'", refunder.order.IdempotencyKey, order.ID)
	}
	if _, err := svc.RefundOrder(ctx, refunder, order.ID); !errors.Is(err, commerce.ErrOrderNotRefundable) || refunder.calls != 1 {
		t.Fatalf("a second refund must not reach the provider again: %v (%d calls)", err, refunder.calls)
	}

	// Unshipped order must have its stock returned (started with 5, 1 checked out -> 4, refunded -> 5)
	afterOrder, _, err := svc.OrderWithItems(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterOrder.PaymentStatus != commerce.PaymentRefunded {
		t.Fatalf("order status = %s, want refunded", afterOrder.PaymentStatus)
	}
}

func TestRefundOrderKeepsPendingStatusOnProviderFailure(t *testing.T) {
	svc, order, ctx := paidOrder(t, "fail")
	refunder := &fakeRefunder{err: errors.New("provider down")}
	if _, err := svc.RefundOrder(ctx, refunder, order.ID); err == nil {
		t.Fatal("a provider failure must return an error")
	}

	// Per hardening requirements: status must NOT be reverted to paid.
	// It must stay 'refund_pending' to alert operators and prevent double refund attempts.
	after, _, err := svc.OrderWithItems(ctx, order.ID)
	if err != nil || after.PaymentStatus != commerce.PaymentRefundPending {
		t.Fatalf("the order status must remain refund_pending: got %+v, %v", after, err)
	}

	// A retry while in refund_pending must be rejected to prevent duplicate calls to provider
	if _, err := svc.RefundOrder(ctx, refunder, order.ID); !errors.Is(err, commerce.ErrOrderNotRefundable) {
		t.Fatalf("expected ErrOrderNotRefundable for refund in progress, got: %v", err)
	}
	if refunder.calls != 1 {
		t.Fatalf("provider must not have been called again; got %d calls", refunder.calls)
	}
}

func TestRefundNeedsAPaidOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "variant-refund-pending", 5); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "variant-refund-pending", 1, 5000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, GuestEmail: "g@example.com", IdempotencyKey: "idem-refund-pending"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RefundOrder(ctx, &fakeRefunder{}, order.ID); !errors.Is(err, commerce.ErrOrderNotRefundable) {
		t.Fatalf("got %v", err)
	}
}

func TestRefundShippedOrderRequiresApprovalAndDoesNotRestoreStock(t *testing.T) {
	svc, order, ctx := paidOrder(t, "shipped")
	refunder := &fakeRefunder{}

	// Mark order as shipped: claim (unfulfilled -> preparing) then record (preparing -> shipped)
	err := svc.Transactions.Within(ctx, func(tx database.Executor) error {
		claimed, err := svc.Repository.ClaimShipment(ctx, tx, order.ID, time.Time{})
		if err != nil || !claimed {
			return err
		}
		shipped, err := svc.Repository.RecordShipment(ctx, tx, order.ID, commerce.ShipmentResult{Carrier: "carrierb", TrackingNumber: "TRK-123", TrackingURL: "http://track"}, time.Now().UTC())
		if err != nil || !shipped {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to record shipment: %v", err)
	}

	// Attempt refund without approval: must fail with ErrOrderShippedRequiresApproval
	if _, err := svc.RefundOrder(ctx, refunder, order.ID); !errors.Is(err, commerce.ErrOrderShippedRequiresApproval) {
		t.Fatalf("expected ErrOrderShippedRequiresApproval, got: %v", err)
	}
	if refunder.calls != 0 {
		t.Fatalf("provider must not be called when shipment check fails, got %d calls", refunder.calls)
	}

	// Attempt refund WITH approval: must succeed
	got, err := svc.RefundOrder(ctx, refunder, order.ID, commerce.WithAllowShipped(true))
	if err != nil {
		t.Fatalf("refund with approval failed: %v", err)
	}
	if got.PaymentStatus != commerce.PaymentRefunded {
		t.Fatalf("order payment_status = %s, want refunded", got.PaymentStatus)
	}
	if refunder.calls != 1 {
		t.Fatalf("expected 1 refunder call, got %d", refunder.calls)
	}
}

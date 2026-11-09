package commerce_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type fakeRefunder struct {
	err   error
	calls int
	got   commerce.PaymentCharge
}

func (f *fakeRefunder) Refund(_ context.Context, _ commerce.Order, c commerce.PaymentCharge) error {
	f.calls++
	f.got = c
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
	if _, err := svc.ConfirmPayment(ctx, commerce.Payment{Provider: "iyzico", EventID: order.ID, ProviderPaymentID: "pay-77", OrderID: order.ID, AmountCents: order.TotalCents}); err != nil {
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
	if refunder.calls != 1 || refunder.got.Provider != "iyzico" || refunder.got.ProviderPaymentID != "pay-77" {
		t.Fatalf("the provider must get the recorded charge: %+v (%d calls)", refunder.got, refunder.calls)
	}
	if _, err := svc.RefundOrder(ctx, refunder, order.ID); !errors.Is(err, commerce.ErrOrderNotRefundable) || refunder.calls != 1 {
		t.Fatalf("a second refund must not reach the provider again: %v (%d calls)", err, refunder.calls)
	}
}

func TestRefundOrderGivesTheClaimBackWhenTheProviderFails(t *testing.T) {
	svc, order, ctx := paidOrder(t, "fail")
	if _, err := svc.RefundOrder(ctx, &fakeRefunder{err: errors.New("provider down")}, order.ID); err == nil {
		t.Fatal("a provider failure must fail the refund")
	}
	after, _, err := svc.OrderWithItems(ctx, order.ID)
	if err != nil || after.PaymentStatus != commerce.PaymentPaid {
		t.Fatalf("the order must be paid again so the refund can be retried: %+v, %v", after, err)
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

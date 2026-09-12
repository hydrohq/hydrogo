package transport

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type orderReaderStub struct{ order commerce.Order }

func (s orderReaderStub) ListOrders(context.Context, database.Executor, int) ([]commerce.Order, error) {
	return nil, nil
}
func (s orderReaderStub) GetOrder(context.Context, database.Executor, string) (commerce.Order, error) {
	return s.order, nil
}

type paymentProviderStub struct{ called bool }

func (*paymentProviderStub) Name() string { return "test" }
func (p *paymentProviderStub) CreatePaymentIntent(_ context.Context, order commerce.Order) (commerce.PaymentIntent, error) {
	p.called = true
	return commerce.PaymentIntent{ID: "intent", OrderID: order.ID, Provider: "test"}, nil
}
func (*paymentProviderStub) VerifyWebhookSignature(*http.Request) (commerce.WebhookEvent, error) {
	return commerce.WebhookEvent{}, nil
}
func (*paymentProviderStub) Refund(context.Context, string, int64) error { return nil }

func TestNewIDIsNonEmpty(t *testing.T) {
	id, err := newID()
	if err != nil || id == "" {
		t.Fatalf("id = %q/%v", id, err)
	}
}

func TestResolveCartUserIDUsesStableMemberID(t *testing.T) {
	h := Handler{ResolveMemberID: func(*http.Request) (string, bool) { return "member-1", true }}
	req := httptest.NewRequest(http.MethodGet, "/shop/cart", nil)
	req.AddCookie(&http.Cookie{Name: "hydro_member_session", Value: "raw-browser-secret"})
	res := httptest.NewRecorder()
	if got := h.resolveCartUserID(res, req); got != "member-1" {
		t.Fatalf("cart identity = %q", got)
	}
	if len(res.Result().Cookies()) != 0 {
		t.Fatal("member identity unexpectedly minted a guest cart cookie")
	}
}

func TestPaymentIntentAcceptsOrderOwnerGuestCapability(t *testing.T) {
	provider := &paymentProviderStub{}
	h := Handler{Orders: orderReaderStub{order: commerce.Order{ID: "order-1", UserID: "guest-1", PaymentStatus: "pending"}}, Payments: map[string]commerce.PaymentProvider{"test": provider}}
	req := httptest.NewRequest(http.MethodPost, "/api/commerce/orders/order-1/payment-intent", bytes.NewBufferString(`{"provider":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("orderID", "order-1")
	req.AddCookie(&http.Cookie{Name: cartCookieName, Value: "guest-1"})
	res := httptest.NewRecorder()
	h.paymentIntent(res, req)
	if res.Code != http.StatusOK || !provider.called {
		t.Fatalf("status=%d provider called=%t", res.Code, provider.called)
	}
}

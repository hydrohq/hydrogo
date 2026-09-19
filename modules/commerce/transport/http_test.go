package transport

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type publicMediaURLStub struct {
	URL string
	Err error
}

func (s publicMediaURLStub) PublicURL(string) (string, error) { return s.URL, s.Err }

func TestProductMediaMarkupUsesConfiguredCDN(t *testing.T) {
	h := Handler{Media: publicMediaURLStub{URL: "https://cdn.example.test/media/product.webp"}}
	got := string(h.productMediaMarkup(commerce.Product{MediaID: "product", Title: "<product>"}))
	if !strings.Contains(got, `src="https://cdn.example.test/media/product.webp"`) {
		t.Fatalf("cdn URL missing: %s", got)
	}
	if strings.Contains(got, "<product>") || !strings.Contains(got, "&lt;product&gt;") {
		t.Fatalf("title was not escaped: %s", got)
	}
}

func TestProductMediaMarkupFallsBackWhenCDNUnavailable(t *testing.T) {
	h := Handler{Media: publicMediaURLStub{Err: errors.New("unavailable")}}
	got := string(h.productMediaMarkup(commerce.Product{MediaID: "product", Title: "Product"}))
	if !strings.Contains(got, `src="/media/product?variant=card"`) {
		t.Fatalf("local fallback missing: %s", got)
	}
}

func TestCommerceDocumentUsesConfiguredLocale(t *testing.T) {
	start := commerceDocumentStart("Mağaza", "en-US")
	footer := commerceEnterpriseFooter("en-US")
	for _, output := range []string{start, footer} {
		if !strings.Contains(output, `lang="en-US"`) && output == start {
			t.Fatalf("document language was not localized: %s", output)
		}
		if strings.Contains(output, "/public/pages/tr/") || strings.Contains(output, "/public/news/tr/") {
			t.Fatalf("legacy locale link leaked: %s", output)
		}
	}
}

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

type keyedPaymentProviderStub struct {
	paymentProviderStub
	key string
}

func (p *keyedPaymentProviderStub) CreatePaymentIntentWithKey(_ context.Context, order commerce.Order, key string) (commerce.PaymentIntent, error) {
	p.key = key
	return commerce.PaymentIntent{ID: "intent-keyed", OrderID: order.ID, Provider: "test"}, nil
}

func TestNewIDIsNonEmpty(t *testing.T) {
	id, err := newID()
	if err != nil || id == "" {
		t.Fatalf("id = %q/%v", id, err)
	}
}

func TestCatalogPageRejectsMalformedValues(t *testing.T) {
	for _, raw := range []string{"abc", "0", "10001"} {
		req := httptest.NewRequest(http.MethodGet, "/api/commerce/products?page="+raw, nil)
		if _, err := catalogPage(req); err == nil {
			t.Fatalf("page %q was accepted", raw)
		}
	}
	if page, err := catalogPage(httptest.NewRequest(http.MethodGet, "/api/commerce/products", nil)); err != nil || page != 1 {
		t.Fatalf("default page = %d/%v", page, err)
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

func TestCommerceActorIDAcceptsConsumerMobileMemberBoundary(t *testing.T) {
	h := Handler{ResolveMobileMemberID: func(r *http.Request) (string, bool) {
		if r.Header.Get("Authorization") == "Bearer member-token" {
			return "member-mobile-1", true
		}
		return "", false
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/mobile/member/commerce/cart", nil)
	req.Header.Set("Authorization", "Bearer member-token")
	if got, ok := h.commerceActorID(req); !ok || got != "member-mobile-1" {
		t.Fatalf("mobile actor = %q/%t", got, ok)
	}
	if got, ok := h.commerceCustomerID(req); !ok || got != "member-mobile-1" {
		t.Fatalf("mobile customer = %q/%t", got, ok)
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

func TestPaymentIntentPassesBoundedKeyToSupportingProvider(t *testing.T) {
	provider := &keyedPaymentProviderStub{}
	h := Handler{Orders: orderReaderStub{order: commerce.Order{ID: "order-1", UserID: "guest-1", PaymentStatus: "pending"}}, Payments: map[string]commerce.PaymentProvider{"test": provider}}
	req := httptest.NewRequest(http.MethodPost, "/api/commerce/orders/order-1/payment-intent", bytes.NewBufferString(`{"provider":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "payment-intent-key-01")
	req.SetPathValue("orderID", "order-1")
	req.AddCookie(&http.Cookie{Name: cartCookieName, Value: "guest-1"})
	res := httptest.NewRecorder()
	h.paymentIntent(res, req)
	if res.Code != http.StatusOK || provider.key != "payment-intent-key-01" {
		t.Fatalf("status=%d provider key=%q", res.Code, provider.key)
	}
}

func TestPaymentIntentRejectsMalformedKeyBeforeProvider(t *testing.T) {
	provider := &keyedPaymentProviderStub{}
	h := Handler{Orders: orderReaderStub{order: commerce.Order{ID: "order-1", UserID: "guest-1", PaymentStatus: "pending"}}, Payments: map[string]commerce.PaymentProvider{"test": provider}}
	req := httptest.NewRequest(http.MethodPost, "/api/commerce/orders/order-1/payment-intent", bytes.NewBufferString(`{"provider":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "bad\nkey")
	req.SetPathValue("orderID", "order-1")
	req.AddCookie(&http.Cookie{Name: cartCookieName, Value: "guest-1"})
	res := httptest.NewRecorder()
	h.paymentIntent(res, req)
	if res.Code != http.StatusBadRequest || provider.key != "" {
		t.Fatalf("status=%d provider key=%q", res.Code, provider.key)
	}
}

func TestSanitizePaymentScriptRedactsRuntimeError(t *testing.T) {
	got := sanitizePaymentScript("status.textContent=error.message||'Ödeme başlatılamadı.';")
	if got != "status.textContent='Ödeme başlatılamadı.';" {
		t.Fatalf("payment runtime error leaked: %q", got)
	}
}

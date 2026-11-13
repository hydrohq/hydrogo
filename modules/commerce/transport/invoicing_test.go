package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hydrohq/hydrogo/internal/notify"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

type fakeInvoiceProvider struct {
	name   string
	result commerce.InvoiceResult
	err    error
	calls  int
}

func (f *fakeInvoiceProvider) Name() string {
	if f.name == "" {
		return "bizimhesap"
	}
	return f.name
}

func (f *fakeInvoiceProvider) IssueInvoice(_ context.Context, doc commerce.InvoiceDocument) (commerce.InvoiceResult, error) {
	f.calls++
	if f.err != nil {
		return commerce.InvoiceResult{}, f.err
	}
	res := f.result
	if res.GUID == "" {
		res.GUID = "GUID-" + doc.OrderID
	}
	if res.URL == "" {
		res.URL = "https://invoice.example.test/" + res.GUID
	}
	return res, nil
}

type fakeInvoiceCustomerResolver struct {
	info commerce.InvoiceCustomerInfo
	ok   bool
	err  error
}

func (f fakeInvoiceCustomerResolver) ResolveInvoiceCustomer(_ context.Context, _ string) (commerce.InvoiceCustomerInfo, bool, error) {
	return f.info, f.ok, f.err
}

type mockInvoiceNotifier struct {
	mu       sync.Mutex
	messages []notify.Message
}

func (m *mockInvoiceNotifier) Send(_ context.Context, msg notify.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func TestInvoicingStatusEndpoint(t *testing.T) {
	h, _ := newTestHandler(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/commerce/invoicing/status", nil)
	h.getInvoicingStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var unconfigured struct {
		Configured      bool   `json:"configured"`
		Provider        string `json:"provider"`
		AutoIssueOnPaid bool   `json:"auto_issue_on_paid"`
	}
	decodeBody(t, rec, &unconfigured)
	if unconfigured.Configured || unconfigured.Provider != "" || unconfigured.AutoIssueOnPaid {
		t.Fatalf("unexpected unconfigured status: %+v", unconfigured)
	}

	h.InvoiceProvider = &fakeInvoiceProvider{name: "bizimhesap"}
	h.AutoIssueInvoiceOnPaid = true
	rec = httptest.NewRecorder()
	h.getInvoicingStatus(rec, req)
	var configured struct {
		Configured      bool   `json:"configured"`
		Provider        string `json:"provider"`
		AutoIssueOnPaid bool   `json:"auto_issue_on_paid"`
	}
	decodeBody(t, rec, &configured)
	if !configured.Configured || configured.Provider != "bizimhesap" || !configured.AutoIssueOnPaid {
		t.Fatalf("unexpected configured status: %+v", configured)
	}
}

func TestIssueOrderInvoiceEndpointStates(t *testing.T) {
	h, db := newTestHandler(t)
	ctx := context.Background()

	// 503 when no provider configured
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/any/invoice", nil)
	req.SetPathValue("id", "any")
	h.issueOrderInvoice(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured status = %d, want 503", rec.Code)
	}

	provider := &fakeInvoiceProvider{}
	h.InvoiceProvider = provider

	if err := h.Service.Repository.EnsureStock(ctx, db, "var-tr-inv", 10); err != nil {
		t.Fatal(err)
	}
	cart, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-tr-inv", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(h.Service, ctx, db, cart, "var-tr-inv", 1, 7500); err != nil {
		t.Fatal(err)
	}
	order, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{
		CartID:         cart.ID,
		GuestName:      "Deneme",
		GuestEmail:     "deneme@example.test",
		IdempotencyKey: "tr-inv-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 409 before order is paid
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/invoice", nil)
	req.SetPathValue("id", order.ID)
	rec = httptest.NewRecorder()
	h.issueOrderInvoice(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("unpaid status = %d, want 409 (%s)", rec.Code, rec.Body.String())
	}

	if ok, err := pay(h.Service, ctx, "paytr", "evt-tr-inv-1", order.ID); err != nil || !ok {
		t.Fatalf("pay failed: ok=%v err=%v", ok, err)
	}

	// 422 when corporate invoice details are incomplete
	h.Service.InvoiceCustomers = fakeInvoiceCustomerResolver{
		ok:   true,
		info: commerce.InvoiceCustomerInfo{InvoiceType: "kurumsal", CompanyName: "Broken Ltd", VKN: "123"},
	}
	rec = httptest.NewRecorder()
	h.issueOrderInvoice(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid customer status = %d, want 422 (%s)", rec.Code, rec.Body.String())
	}

	// 200 OK with valid individual customer
	h.Service.InvoiceCustomers = fakeInvoiceCustomerResolver{}
	notifier := &mockInvoiceNotifier{}
	h.Notifier = notifier
	h.Refunder = &fakeRefunder{}

	rec = httptest.NewRecorder()
	h.issueOrderInvoice(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("issue status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var resp commerce.Order
	decodeBody(t, rec, &resp)
	if resp.InvoiceStatus != commerce.InvoiceIssued || resp.InvoiceGUID != "GUID-"+order.ID {
		t.Fatalf("unexpected issued response: %+v", resp)
	}

	// 409 on duplicate call
	rec = httptest.NewRecorder()
	h.issueOrderInvoice(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate issue status = %d, want 409", rec.Code)
	}

	// Refunding an issued order dispatches Telegram/operator alert
	refundReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order.ID+"/refund", strings.NewReader(`{}`))
	refundReq.SetPathValue("id", order.ID)
	refundRec := httptest.NewRecorder()
	h.refundOrder(refundRec, refundReq)
	if refundRec.Code != http.StatusOK {
		t.Fatalf("refund status = %d, want 200 (%s)", refundRec.Code, refundRec.Body.String())
	}
	WaitNotifications()
	if len(notifier.messages) != 1 || !strings.Contains(notifier.messages[0].Title, "e-Fatura") || !strings.Contains(notifier.messages[0].Body, "GUID-"+order.ID) {
		t.Fatalf("expected issued-invoice refund alert, got %+v", notifier.messages)
	}

	// 502 on provider error
	cart2, err := h.Service.GetOrCreateCart(ctx, db, "", "anon-tr-inv-2", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(h.Service, ctx, db, cart2, "var-tr-inv", 1, 5000); err != nil {
		t.Fatal(err)
	}
	order2, err := h.Service.Checkout(ctx, commerce.CheckoutRequest{
		CartID:         cart2.ID,
		GuestEmail:     "err@example.test",
		IdempotencyKey: "tr-inv-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := pay(h.Service, ctx, "paytr", "evt-tr-inv-2", order2.ID); err != nil || !ok {
		t.Fatalf("pay2 failed: ok=%v err=%v", ok, err)
	}
	h.InvoiceProvider = &fakeInvoiceProvider{err: errors.New("timeout")}
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/orders/"+order2.ID+"/invoice", nil)
	req2.SetPathValue("id", order2.ID)
	rec2 := httptest.NewRecorder()
	h.issueOrderInvoice(rec2, req2)
	if rec2.Code != http.StatusBadGateway {
		t.Fatalf("provider error status = %d, want 502 (%s)", rec2.Code, rec2.Body.String())
	}
}

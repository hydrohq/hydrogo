package commerce_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

type recordingOutbox struct {
	mu     sync.Mutex
	topics []string
}

func (o *recordingOutbox) Enqueue(_ context.Context, _ database.Executor, _, topic, aggregateID string, _ int) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.topics = append(o.topics, topic+":"+aggregateID)
	return nil
}

type fakeInvoiceCustomerResolver struct {
	byOrder map[string]commerce.InvoiceCustomerInfo
	err     error
}

func (f fakeInvoiceCustomerResolver) ResolveInvoiceCustomer(_ context.Context, orderID string) (commerce.InvoiceCustomerInfo, bool, error) {
	if f.err != nil {
		return commerce.InvoiceCustomerInfo{}, false, f.err
	}
	info, ok := f.byOrder[orderID]
	return info, ok, nil
}

type fakeInvoiceProvider struct {
	mu      sync.Mutex
	name    string
	calls   int
	lastDoc commerce.InvoiceDocument
	result  commerce.InvoiceResult
	err     error
}

func (f *fakeInvoiceProvider) Name() string {
	if f.name == "" {
		return "bizimhesap"
	}
	return f.name
}

func (f *fakeInvoiceProvider) IssueInvoice(_ context.Context, doc commerce.InvoiceDocument) (commerce.InvoiceResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastDoc = doc
	if f.err != nil {
		return commerce.InvoiceResult{}, f.err
	}
	res := f.result
	if res.GUID == "" {
		res.GUID = "BH-GUID-" + doc.OrderID
	}
	if res.URL == "" {
		res.URL = "https://example.test/invoices/" + res.GUID
	}
	return res, nil
}

func assertInvoiceLineInvariants(t *testing.T, doc commerce.InvoiceDocument, expectedTotalCents int64) {
	t.Helper()
	var sumGross, sumDiscount, sumNet, sumTax, sumTotal int64
	for i, l := range doc.Lines {
		if l.GrossCents-l.DiscountNetCents != l.NetCents {
			t.Fatalf("line[%d] gross (%d) - discount (%d) != net (%d)", i, l.GrossCents, l.DiscountNetCents, l.NetCents)
		}
		if l.NetCents+l.TaxCents != l.TotalCents {
			t.Fatalf("line[%d] net (%d) + tax (%d) != total (%d)", i, l.NetCents, l.TaxCents, l.TotalCents)
		}
		sumGross += l.GrossCents
		sumDiscount += l.DiscountNetCents
		sumNet += l.NetCents
		sumTax += l.TaxCents
		sumTotal += l.TotalCents
	}
	if sumGross != doc.Amounts.GrossCents || sumDiscount != doc.Amounts.DiscountCents ||
		sumNet != doc.Amounts.NetCents || sumTax != doc.Amounts.TaxCents || sumTotal != doc.Amounts.TotalCents {
		t.Fatalf("line sums (%d,%d,%d,%d,%d) != amounts (%+v)", sumGross, sumDiscount, sumNet, sumTax, sumTotal, doc.Amounts)
	}
	if doc.Amounts.TotalCents != expectedTotalCents {
		t.Fatalf("amounts.TotalCents = %d, want %d", doc.Amounts.TotalCents, expectedTotalCents)
	}
}

func TestBuildInvoiceDocumentInclusiveAndExclusiveTaxWithDiscount(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)

	t.Run("tax inclusive multi item with uneven rounding and shipping", func(t *testing.T) {
		items := []commerce.OrderItem{
			{ID: "i1", SKU: "BEY-01", Title: "Kettlebell", Quantity: 2, UnitPriceCents: 3333, TotalCents: 6666},
			{ID: "i2", SKU: "BEY-02", Title: "Barbell Pad", Quantity: 1, UnitPriceCents: 7777, TotalCents: 7777},
			{ID: "i3", SKU: "BEY-03", Title: "Towel", Quantity: 1, UnitPriceCents: 1999, TotalCents: 1999},
		}
		// Subtotal = 16442, discount = 1000, shipping = 4990 => Total = 20432
		order := commerce.Order{
			ID:            "ord-inc-1",
			GuestName:     "Arda",
			GuestEmail:    "arda@example.test",
			GuestAddress:  "Moda Cad. 10",
			GuestDistrict: "Kadikoy",
			GuestCity:     "Istanbul",
			Currency:      "TRY",
			SubtotalCents: 16442,
			DiscountCode:  "HOSGELDIN",
			DiscountCents: 1000,
			ShippingCents: 4990,
			TotalCents:    20432,
		}
		doc, err := commerce.BuildInvoiceDocument(order, items, commerce.PricingConfig{TaxRate: 0.10, TaxInclusive: true}, commerce.InvoiceCustomerInfo{}, false, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if doc.Currency != "TL" {
			t.Fatalf("currency = %q, want TL", doc.Currency)
		}
		if len(doc.Lines) != 4 {
			t.Fatalf("len(lines) = %d, want 4 (3 items + shipping)", len(doc.Lines))
		}
		assertInvoiceLineInvariants(t, doc, order.TotalCents)
	})

	t.Run("tax exclusive with discount and shipping", func(t *testing.T) {
		items := []commerce.OrderItem{
			{ID: "i1", SKU: "SKU-A", Title: "Mat", Quantity: 1, UnitPriceCents: 10000, TotalCents: 10000},
		}
		// Net after 1000 discount = 9000, 20% KDV = 1800, shipping = 1500 => Total = 12300
		order := commerce.Order{
			ID:            "ord-exc-1",
			GuestEmail:    "guest@example.test",
			Currency:      "TRY",
			SubtotalCents: 10000,
			DiscountCents: 1000,
			TaxCents:      1800,
			ShippingCents: 1500,
			TotalCents:    12300,
		}
		doc, err := commerce.BuildInvoiceDocument(order, items, commerce.PricingConfig{TaxRate: 0.20, TaxInclusive: false}, commerce.InvoiceCustomerInfo{}, false, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertInvoiceLineInvariants(t, doc, order.TotalCents)
		if doc.Amounts.TaxCents != 1800 {
			t.Fatalf("tax = %d, want 1800", doc.Amounts.TaxCents)
		}
	})
}

func TestBuildInvoiceDocumentValidation(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	baseOrder := commerce.Order{ID: "ord-val-1", GuestName: "Test Buyer", TotalCents: 5000, Currency: "TRY"}
	items := []commerce.OrderItem{{ID: "i1", Title: "Grip", Quantity: 1, UnitPriceCents: 5000, TotalCents: 5000}}

	// Individual blank TCKN -> DefaultIndividualTaxNo
	doc, err := commerce.BuildInvoiceDocument(baseOrder, items, commerce.PricingConfig{}, commerce.InvoiceCustomerInfo{InvoiceType: "bireysel"}, true, now)
	if err != nil {
		t.Fatalf("individual blank tckn error: %v", err)
	}
	if doc.Customer.IsCompany || doc.Customer.TaxNo != commerce.DefaultIndividualTaxNo {
		t.Fatalf("unexpected individual customer: %+v", doc.Customer)
	}

	// Individual invalid TCKN -> ErrInvoiceCustomerInvalid
	_, err = commerce.BuildInvoiceDocument(baseOrder, items, commerce.PricingConfig{}, commerce.InvoiceCustomerInfo{InvoiceType: "bireysel", TCKN: "12345"}, true, now)
	if !errors.Is(err, commerce.ErrInvoiceCustomerInvalid) {
		t.Fatalf("expected ErrInvoiceCustomerInvalid, got %v", err)
	}

	// Corporate missing VKN/tax_office -> ErrInvoiceCustomerInvalid
	_, err = commerce.BuildInvoiceDocument(baseOrder, items, commerce.PricingConfig{}, commerce.InvoiceCustomerInfo{InvoiceType: "kurumsal", CompanyName: "ACME AS", VKN: "123"}, true, now)
	if !errors.Is(err, commerce.ErrInvoiceCustomerInvalid) {
		t.Fatalf("expected ErrInvoiceCustomerInvalid for bad VKN, got %v", err)
	}

	// Corporate valid -> IsCompany=true
	corpDoc, err := commerce.BuildInvoiceDocument(baseOrder, items, commerce.PricingConfig{}, commerce.InvoiceCustomerInfo{
		InvoiceType: "kurumsal",
		CompanyName: "Beyond Sports A.S.",
		VKN:         "1234567890",
		TaxOffice:   "Kadikoy",
	}, true, now)
	if err != nil {
		t.Fatalf("corporate valid error: %v", err)
	}
	if !corpDoc.Customer.IsCompany || corpDoc.Customer.TaxNo != "1234567890" || corpDoc.Customer.TaxOffice != "Kadikoy" || corpDoc.Customer.Title != "Beyond Sports A.S." {
		t.Fatalf("unexpected corporate customer: %+v", corpDoc.Customer)
	}

	// Zero total -> ErrOrderNotInvoiceable
	zeroOrder := baseOrder
	zeroOrder.TotalCents = 0
	if _, err := commerce.BuildInvoiceDocument(zeroOrder, items, commerce.PricingConfig{}, commerce.InvoiceCustomerInfo{}, false, now); !errors.Is(err, commerce.ErrOrderNotInvoiceable) {
		t.Fatalf("expected ErrOrderNotInvoiceable for zero total, got %v", err)
	}
}

func TestIssueOrderInvoiceFlowAndConcurrency(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	outbox := &recordingOutbox{}
	svc.Outbox = outbox
	svc.Pricing = commerce.PricingConfig{TaxRate: 0.10, TaxInclusive: true}

	if err := svc.Repository.EnsureStock(ctx, db, "var-inv-1", 10); err != nil {
		t.Fatal(err)
	}
	cart := seedCartWithItem(t, svc, db, "var-inv-1", 2, 11000)
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID:         cart.ID,
		GuestName:      "Ali Yilmaz",
		GuestEmail:     "ali@example.test",
		GuestAddress:   "Bagdat Cad. 1",
		GuestCity:      "Istanbul",
		IdempotencyKey: "inv-order-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	svc.InvoiceCustomers = fakeInvoiceCustomerResolver{
		byOrder: map[string]commerce.InvoiceCustomerInfo{
			order.ID: {
				InvoiceType: "kurumsal",
				CompanyName: "Beyonfit Ltd.",
				VKN:         "9876543210",
				TaxOffice:   "Besiktas",
			},
		},
	}

	provider := &fakeInvoiceProvider{}
	// Unpaid order must be rejected
	if _, err := svc.IssueOrderInvoice(ctx, provider, order.ID); !errors.Is(err, commerce.ErrOrderNotInvoiceable) {
		t.Fatalf("expected ErrOrderNotInvoiceable before payment, got %v", err)
	}

	ok, err := pay(svc, ctx, "paytr", "evt-inv-1", order.ID)
	if err != nil || !ok {
		t.Fatalf("pay failed: ok=%v err=%v", ok, err)
	}

	issued, err := svc.IssueOrderInvoice(ctx, provider, order.ID)
	if err != nil {
		t.Fatalf("IssueOrderInvoice failed: %v", err)
	}
	if issued.InvoiceStatus != commerce.InvoiceIssued || issued.InvoiceGUID == "" || issued.InvoiceURL == "" || issued.InvoiceProvider != "bizimhesap" {
		t.Fatalf("unexpected issued order state: %+v", issued)
	}
	if !provider.lastDoc.Customer.IsCompany || provider.lastDoc.Customer.TaxNo != "9876543210" {
		t.Fatalf("corporate tax details not resolved: %+v", provider.lastDoc.Customer)
	}
	if len(outbox.topics) == 0 || outbox.topics[len(outbox.topics)-1] != "order.invoiced:"+order.ID {
		t.Fatalf("expected order.invoiced outbox event, got %v", outbox.topics)
	}

	// Second attempt on already issued order returns ErrOrderNotInvoiceable without calling provider again
	if _, err := svc.IssueOrderInvoice(ctx, provider, order.ID); !errors.Is(err, commerce.ErrOrderNotInvoiceable) {
		t.Fatalf("expected ErrOrderNotInvoiceable on duplicate, got %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}

	// Failed attempt records InvoiceFailed + InvoiceError and allows manual retry
	cart2 := seedCartWithItem(t, svc, db, "var-inv-1", 1, 5500)
	order2, err := svc.Checkout(ctx, commerce.CheckoutRequest{
		CartID:         cart2.ID,
		GuestEmail:     "retry@example.test",
		IdempotencyKey: "inv-order-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := pay(svc, ctx, "paytr", "evt-inv-2", order2.ID); err != nil || !ok {
		t.Fatalf("pay2 failed: ok=%v err=%v", ok, err)
	}
	failingProvider := &fakeInvoiceProvider{err: errors.New("upstream 503")}
	if _, err := svc.IssueOrderInvoice(ctx, failingProvider, order2.ID); !errors.Is(err, commerce.ErrInvoiceProviderFailed) {
		t.Fatalf("expected ErrInvoiceProviderFailed, got %v", err)
	}
	failedOrder, _, err := svc.OrderWithItems(ctx, order2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failedOrder.InvoiceStatus != commerce.InvoiceFailed || !strings.Contains(failedOrder.InvoiceError, "upstream 503") {
		t.Fatalf("expected failed invoice metadata persisted, got %+v", failedOrder)
	}

	// Retry with healthy provider must succeed
	retried, err := svc.IssueOrderInvoice(ctx, provider, order2.ID)
	if err != nil {
		t.Fatalf("manual retry after failure failed: %v", err)
	}
	if retried.InvoiceStatus != commerce.InvoiceIssued {
		t.Fatalf("expected issued after retry, got %q", retried.InvoiceStatus)
	}
}

func TestInvoiceWorkerIssueDueAndAlert(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	if err := svc.Repository.EnsureStock(ctx, db, "var-w-1", 10); err != nil {
		t.Fatal(err)
	}

	makePaidOrder := func(key string) commerce.Order {
		c := seedCartWithItem(t, svc, db, "var-w-1", 1, 9000)
		o, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: c.ID, GuestEmail: key + "@example.test", IdempotencyKey: key})
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := pay(svc, ctx, "paytr", "evt-"+key, o.ID); err != nil || !ok {
			t.Fatalf("pay %s failed: ok=%v err=%v", key, ok, err)
		}
		return o
	}

	settledOrder := makePaidOrder("worker-settled")
	freshOrder := makePaidOrder("worker-fresh")

	// Backdate only settledOrder past the 15s settle delay.
	if _, err := db.ExecContext(ctx, "UPDATE commerce_customer_orders SET updated_at = ? WHERE id = ?", time.Now().UTC().Add(-2*time.Minute).Format(time.RFC3339Nano), settledOrder.ID); err != nil {
		t.Fatal(err)
	}

	provider := &fakeInvoiceProvider{}
	notifier := &mockReminderNotifier{}
	worker := commerce.InvoiceWorker{
		Service:     svc,
		Provider:    provider,
		Notifier:    notifier,
		SettleDelay: 15 * time.Second,
	}

	count, err := worker.IssueDue(ctx)
	if err != nil {
		t.Fatalf("IssueDue failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("issued count = %d, want 1", count)
	}

	gotFresh, _, err := svc.OrderWithItems(ctx, freshOrder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotFresh.InvoiceStatus != commerce.InvoiceNone {
		t.Fatalf("fresh order within settle window should stay none, got %q", gotFresh.InvoiceStatus)
	}

	// Now settle freshOrder and fail its issuance -> expect Telegram alert and failed status.
	if _, err := db.ExecContext(ctx, "UPDATE commerce_customer_orders SET updated_at = ? WHERE id = ?", time.Now().UTC().Add(-2*time.Minute).Format(time.RFC3339Nano), freshOrder.ID); err != nil {
		t.Fatal(err)
	}
	worker.Provider = &fakeInvoiceProvider{err: errors.New("tax no mismatch")}
	count, err = worker.IssueDue(ctx)
	if err == nil {
		t.Fatal("expected error from failing provider")
	}
	if count != 0 {
		t.Fatalf("expected 0 issued on failure, got %d", count)
	}
	if len(notifier.messages) != 1 || !strings.Contains(notifier.messages[0].Body, freshOrder.ID) {
		t.Fatalf("expected 1 alert containing order ID %s, got %+v", freshOrder.ID, notifier.messages)
	}
}

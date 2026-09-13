package commerce

import (
	"context"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type customerOrderReaderStub struct{}

func (customerOrderReaderStub) ListOrdersByUserID(context.Context, database.Executor, string, int) ([]Order, error) {
	return []Order{{ID: "order-1", UserID: "member-1", Currency: "TRY", TotalCents: 1250, PaymentStatus: "paid", ShippingStatus: "pending", Customer: PaymentCustomer{IdentityNumber: "must-not-leak"}, Items: []OrderItem{{ProductID: "p-1", SKU: "sku-1", Title: "Bottle", Quantity: 1, UnitCents: 1250, TotalCents: 1250}}}}, nil
}

func TestCustomerOrderReaderProjectsWithoutCustomerSecrets(t *testing.T) {
	got, err := (CustomerOrderReader{Reader: customerOrderReaderStub{}}).ListOrdersByUserID(context.Background(), nil, "member-1", 100)
	if err != nil {
		t.Fatalf("ListOrdersByUserID: %v", err)
	}
	if len(got) != 1 || len(got[0].Items) != 1 || got[0].Items[0].Title != "Bottle" {
		t.Fatalf("unexpected projection: %#v", got)
	}
	if got[0].ID != "order-1" || got[0].TotalCents != 1250 {
		t.Fatalf("missing order fields: %#v", got[0])
	}
}

func TestCustomerOrderReaderRequiresRepository(t *testing.T) {
	if _, err := (CustomerOrderReader{}).ListOrdersByUserID(context.Background(), nil, "member-1", 100); err == nil {
		t.Fatal("expected missing reader error")
	}
}

var _ customerorders.Reader = CustomerOrderReader{}

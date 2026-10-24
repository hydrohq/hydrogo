package commerce_test

import (
	"context"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	"github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
)

func TestMemberOrdersReaderListsOnlyTheRequestingMembersOrders(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-wallet", 5); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "member-a", "", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, "variant-wallet", 1, 500); err != nil {
		t.Fatal(err)
	}
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, MemberID: "member-a", IdempotencyKey: "member-orders-1", ItemTitles: map[string]string{"variant-wallet": "Wallet"}, ItemSKUs: map[string]string{"variant-wallet": "SKU-WALLET"}})
	if err != nil {
		t.Fatal(err)
	}

	reader := commerce.MemberOrdersReader{Repository: sqlstore.Store{Dialect: database.SQLite}}
	var contractReader customerorders.DetailReader = reader

	orders, err := contractReader.ListOrdersByUserID(ctx, db, "member-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 || orders[0].ID != order.ID {
		t.Fatalf("expected exactly order %s for member-a, got %+v", order.ID, orders)
	}
	if orders[0].Items[0].SKU != "SKU-WALLET" || orders[0].Items[0].Title != "Wallet" || orders[0].Items[0].Quantity != 1 {
		t.Fatalf("expected the checked-out item to be projected, got %+v", orders[0].Items)
	}

	others, err := contractReader.ListOrdersByUserID(ctx, db, "member-b", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(others) != 0 {
		t.Fatalf("expected no orders for an unrelated member, got %+v", others)
	}
}

func TestMemberOrdersReaderDetailRejectsAnotherMembersOrder(t *testing.T) {
	svc, db := newTestService(t)
	ctx := context.Background()
	store := svc.Repository

	if err := store.EnsureStock(ctx, db, "variant-belt-detail", 5); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.GetOrCreateCart(ctx, db, "member-a", "", "TRY")
	if err != nil {
		t.Fatal(err)
	}
	if err := addPriced(svc, ctx, db, cart, "variant-belt-detail", 1, 500); err != nil {
		t.Fatal(err)
	}
	order, err := svc.Checkout(ctx, commerce.CheckoutRequest{CartID: cart.ID, MemberID: "member-a", IdempotencyKey: "member-orders-2"})
	if err != nil {
		t.Fatal(err)
	}

	reader := commerce.MemberOrdersReader{Repository: sqlstore.Store{Dialect: database.SQLite}}
	if _, err := reader.GetOrderByUserID(ctx, db, "member-a", order.ID); err != nil {
		t.Fatalf("expected the owning member to read their own order, got %v", err)
	}
	if _, err := reader.GetOrderByUserID(ctx, db, "member-b", order.ID); err == nil {
		t.Fatal("expected an error for a non-owning member")
	}
}

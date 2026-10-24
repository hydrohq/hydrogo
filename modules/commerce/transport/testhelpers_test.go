package transport

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// addPriced lists variant at price in the service's static catalog, then
// adds it to the cart -- the cart takes the catalog's price, never the
// caller's.
func addPriced(svc commerce.Service, ctx context.Context, db database.Executor, cart commerce.Cart, variant string, quantity int, price int64) error {
	svc.Prices.(commerce.StaticPrices)[variant] = commerce.Price{Cents: price}
	return svc.AddItem(ctx, db, cart, variant, quantity)
}

// pay confirms a provider payment for the full order total.
func pay(svc commerce.Service, ctx context.Context, provider, eventID, orderID string) (bool, error) {
	order, _, err := svc.OrderWithItems(ctx, orderID)
	if err != nil {
		return false, err
	}
	return svc.ConfirmPayment(ctx, commerce.Payment{Provider: provider, EventID: eventID, OrderID: orderID, AmountCents: order.TotalCents})
}

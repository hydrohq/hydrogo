package commerce

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// memberOrderRepository is the narrow slice of Repository MemberOrdersReader
// needs -- a member's own order history and its line items, nothing from the
// checkout/stock side.
type memberOrderRepository interface {
	ListOrdersByMemberID(ctx context.Context, db database.Executor, memberID string, limit int) ([]Order, error)
	GetOrderByMemberID(ctx context.Context, db database.Executor, memberID, orderID string) (Order, error)
	ListOrderItems(ctx context.Context, db database.Executor, orderID string) ([]OrderItem, error)
}

// MemberOrdersReader implements internal/contracts/customerorders.DetailReader
// against modules/commerce's own tables, so the member account surface
// (modules/members) can show order history without importing this module
// directly -- cmd/hydro, the composition root, wires this in only when the
// commerce module is enabled.
type MemberOrdersReader struct {
	Repository memberOrderRepository
}

func toContractOrder(order Order, items []OrderItem) customerorders.Order {
	contractItems := make([]customerorders.Item, 0, len(items))
	for _, item := range items {
		contractItems = append(contractItems, customerorders.Item{
			VariantID:  item.VariantID,
			SKU:        item.SKU,
			Title:      item.Title,
			Quantity:   item.Quantity,
			UnitCents:  item.UnitPriceCents,
			TotalCents: item.TotalCents,
		})
	}
	return customerorders.Order{
		ID:             order.ID,
		UserID:         order.MemberID,
		Currency:       order.Currency,
		Items:          contractItems,
		TotalCents:     order.TotalCents,
		PaymentStatus:  string(order.PaymentStatus),
		ShippingStatus: string(order.FulfillmentStatus),
	}
}

// ListOrdersByUserID returns a member's own orders, newest first.
func (r MemberOrdersReader) ListOrdersByUserID(ctx context.Context, db database.Executor, userID string, limit int) ([]customerorders.Order, error) {
	orders, err := r.Repository.ListOrdersByMemberID(ctx, db, userID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]customerorders.Order, 0, len(orders))
	for _, order := range orders {
		items, err := r.Repository.ListOrderItems(ctx, db, order.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, toContractOrder(order, items))
	}
	return result, nil
}

// GetOrderByUserID returns one order, scoped to the member who placed it.
func (r MemberOrdersReader) GetOrderByUserID(ctx context.Context, db database.Executor, userID, orderID string) (customerorders.Order, error) {
	order, err := r.Repository.GetOrderByMemberID(ctx, db, userID, orderID)
	if err != nil {
		return customerorders.Order{}, err
	}
	items, err := r.Repository.ListOrderItems(ctx, db, order.ID)
	if err != nil {
		return customerorders.Order{}, err
	}
	return toContractOrder(order, items), nil
}

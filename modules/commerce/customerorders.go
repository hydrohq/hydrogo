package commerce

import (
	"context"
	"errors"

	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// CustomerOrderReader adapts Commerce's richer order repository to the
// capability projection consumed by Members without coupling Members to this
// module's domain types.
type CustomerOrderReader struct {
	Reader OrderReaderByUserID
}

func (r CustomerOrderReader) ListOrdersByUserID(ctx context.Context, db database.Executor, userID string, limit int) ([]customerorders.Order, error) {
	if r.Reader == nil {
		return nil, errors.New("commerce customer order reader is required")
	}
	orders, err := r.Reader.ListOrdersByUserID(ctx, db, userID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]customerorders.Order, 0, len(orders))
	for _, order := range orders {
		projection := customerorders.Order{ID: order.ID, UserID: order.UserID, Currency: order.Currency, TotalCents: order.TotalCents, PaymentStatus: order.PaymentStatus, ShippingStatus: order.ShippingStatus, Items: make([]customerorders.Item, 0, len(order.Items))}
		for _, item := range order.Items {
			projection.Items = append(projection.Items, customerorders.Item{ProductID: item.ProductID, VariantID: item.VariantID, SKU: item.SKU, Title: item.Title, Quantity: item.Quantity, UnitCents: item.UnitCents, TotalCents: item.TotalCents})
		}
		result = append(result, projection)
	}
	return result, nil
}

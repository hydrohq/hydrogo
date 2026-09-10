package commerce

import (
	"context"
	"fmt"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Store interface {
	FindOrderByIdempotency(context.Context, database.Executor, string, string) (Order, bool, error)
	ReserveProduct(context.Context, database.Executor, string, int) (Product, error)
	CreateOrder(context.Context, database.Executor, Order, string) error
}

type CheckoutCartReader interface {
	GetCart(context.Context, database.Executor, string, string) (Cart, error)
}

type CheckoutCartClearer interface {
	ClearCart(context.Context, database.Executor, string, string) error
}

type VariantCheckoutStore interface {
	ReserveVariant(context.Context, database.Executor, string, int) (ProductVariant, error)
}

type Outbox interface {
	Enqueue(context.Context, database.Executor, string, string, string, int) error
}

type Service struct {
	Transactions database.Transactor
	Store        Store
	Outbox       Outbox
}

func (s Service) PlaceOrder(ctx context.Context, checkout Checkout) (Order, error) {
	if err := checkout.Validate(); err != nil {
		return Order{}, err
	}
	if s.Store == nil || s.Transactions == (database.Transactor{}) {
		return Order{}, errorsDependency
	}
	var order Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		if existing, found, err := s.Store.FindOrderByIdempotency(ctx, tx, checkout.UserID, checkout.IdempotencyKey); err != nil {
			return err
		} else if found {
			order = existing
			return nil
		}
		if checkout.CartID != "" {
			reader, ok := s.Store.(CheckoutCartReader)
			if !ok {
				return errorsDependency
			}
			cart, err := reader.GetCart(ctx, tx, checkout.UserID, checkout.CartID)
			if err != nil || cart.ID != checkout.CartID {
				return ErrInvalidCheckout
			}
			checkout.Items = make([]CheckoutItem, 0, len(cart.Items))
			for _, item := range cart.Items {
				checkout.Items = append(checkout.Items, CheckoutItem{ProductID: item.ProductID, VariantID: item.VariantID, Quantity: item.Quantity})
			}
			if err := checkout.Validate(); err != nil {
				return err
			}
		}
		order = Order{ID: checkout.ID, UserID: checkout.UserID, PaymentStatus: "pending", ShippingStatus: "pending"}
		for _, item := range checkout.Items {
			var product Product
			variantID := ""
			if item.VariantID != "" {
				variantStore, ok := s.Store.(VariantCheckoutStore)
				if !ok {
					return errorsDependency
				}
				variant, err := variantStore.ReserveVariant(ctx, tx, item.VariantID, item.Quantity)
				if err != nil {
					return err
				}
				product = Product{ID: variant.ProductID, SKU: variant.SKU, Title: variant.Title, PriceCents: variant.PriceCents, Currency: variant.Currency, Status: variant.Status}
				variantID = variant.ID
			} else {
				var err error
				product, err = s.Store.ReserveProduct(ctx, tx, item.ProductID, item.Quantity)
				if err != nil {
					return err
				}
			}
			if order.Currency == "" {
				order.Currency = product.Currency
			}
			if order.Currency != product.Currency {
				return ErrInvalidCheckout
			}
			line := OrderItem{ProductID: product.ID, VariantID: variantID, SKU: product.SKU, Title: product.Title, Quantity: item.Quantity, UnitCents: product.PriceCents, TotalCents: product.PriceCents * int64(item.Quantity)}
			order.Items = append(order.Items, line)
			order.TotalCents += line.TotalCents
		}
		if err := s.Store.CreateOrder(ctx, tx, order, checkout.IdempotencyKey); err != nil {
			return err
		}
		if checkout.CartID != "" {
			if clearer, ok := s.Store.(CheckoutCartClearer); ok {
				if err := clearer.ClearCart(ctx, tx, checkout.UserID, checkout.CartID); err != nil {
					return err
				}
			}
		}
		if s.Outbox != nil {
			if err := s.Outbox.Enqueue(ctx, tx, fmt.Sprintf("commerce-order-%s", order.ID), "order.created", order.ID, 1); err != nil {
				return err
			}
			if err := s.Outbox.Enqueue(ctx, tx, fmt.Sprintf("commerce-invoice-%s", order.ID), "email.send_invoice", order.ID, 1); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

var errorsDependency = fmt.Errorf("commerce service dependency is required")

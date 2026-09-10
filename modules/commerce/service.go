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
		order = Order{ID: checkout.ID, UserID: checkout.UserID, PaymentStatus: "pending", ShippingStatus: "pending"}
		for _, item := range checkout.Items {
			product, err := s.Store.ReserveProduct(ctx, tx, item.ProductID, item.Quantity)
			if err != nil {
				return err
			}
			if order.Currency == "" {
				order.Currency = product.Currency
			}
			if order.Currency != product.Currency {
				return ErrInvalidCheckout
			}
			line := OrderItem{ProductID: product.ID, SKU: product.SKU, Title: product.Title, Quantity: item.Quantity, UnitCents: product.PriceCents, TotalCents: product.PriceCents * int64(item.Quantity)}
			order.Items = append(order.Items, line)
			order.TotalCents += line.TotalCents
		}
		if err := s.Store.CreateOrder(ctx, tx, order, checkout.IdempotencyKey); err != nil {
			return err
		}
		if s.Outbox != nil {
			return s.Outbox.Enqueue(ctx, tx, fmt.Sprintf("commerce-order-%s", order.ID), "order.created", order.ID, 1)
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

var errorsDependency = fmt.Errorf("commerce service dependency is required")

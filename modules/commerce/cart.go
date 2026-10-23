package commerce

import (
	"context"
	"errors"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// Shopper identifies whose cart a request touches: a logged-in member or an
// anonymous visitor's cookie token, never both.
type Shopper struct {
	MemberID       string
	AnonymousToken string
	Currency       string
}

// Cart operations for transport. Each runs in one transaction so the cart
// lookup, the change and the read-back see the same state (and, on
// PostgreSQL, the same tenant binding).

// CartFor returns the shopper's cart and its items, creating an empty cart
// on first use.
func (s Service) CartFor(ctx context.Context, shopper Shopper) (Cart, []CartItem, error) {
	var (
		cart  Cart
		items []CartItem
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if cart, err = s.GetOrCreateCart(ctx, tx, shopper.MemberID, shopper.AnonymousToken, shopper.Currency); err != nil {
			return err
		}
		items, err = s.Repository.ListItems(ctx, tx, cart.ID)
		return err
	})
	return cart, items, err
}

// AddToCart sets a line's quantity at the catalog price and returns the
// updated cart.
func (s Service) AddToCart(ctx context.Context, shopper Shopper, variantID string, quantity int) (Cart, []CartItem, error) {
	var (
		cart  Cart
		items []CartItem
	)
	// Priced before the transaction: the price source may read the database
	// itself, and with a one-connection pool (the compact profile) a lookup
	// inside the transaction waits forever for the connection it holds.
	if quantity <= 0 {
		return Cart{}, nil, errors.New("quantity must be positive")
	}
	unitPrice, err := s.unitPrice(ctx, variantID, shopper.Currency)
	if err != nil {
		return Cart{}, nil, err
	}
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if cart, err = s.GetOrCreateCart(ctx, tx, shopper.MemberID, shopper.AnonymousToken, shopper.Currency); err != nil {
			return err
		}
		if err := s.addLine(ctx, tx, cart, variantID, quantity, unitPrice); err != nil {
			return err
		}
		items, err = s.Repository.ListItems(ctx, tx, cart.ID)
		return err
	})
	return cart, items, err
}

// RemoveFromCart deletes one line from the shopper's own cart.
func (s Service) RemoveFromCart(ctx context.Context, shopper Shopper, itemID string) (Cart, []CartItem, error) {
	var (
		cart  Cart
		items []CartItem
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if cart, err = s.GetOrCreateCart(ctx, tx, shopper.MemberID, shopper.AnonymousToken, shopper.Currency); err != nil {
			return err
		}
		if err := s.Repository.RemoveItem(ctx, tx, cart.ID, itemID); err != nil {
			return err
		}
		items, err = s.Repository.ListItems(ctx, tx, cart.ID)
		return err
	})
	return cart, items, err
}

// Stock returns a variant's stock row.
func (s Service) Stock(ctx context.Context, variantID string) (Stock, error) {
	var stock Stock
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		stock, err = s.Repository.GetStock(ctx, tx, variantID)
		return err
	})
	return stock, err
}

// SetStock sets a variant's available count (admin restock).
func (s Service) SetStock(ctx context.Context, variantID string, available int) (Stock, error) {
	var stock Stock
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.Repository.EnsureStock(ctx, tx, variantID, available); err != nil {
			return err
		}
		var err error
		stock, err = s.Repository.GetStock(ctx, tx, variantID)
		return err
	})
	return stock, err
}

// CreateDiscountCode stores a new promo code (admin).
func (s Service) CreateDiscountCode(ctx context.Context, code DiscountCode) error {
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Repository.CreateDiscountCode(ctx, tx, code)
	})
}

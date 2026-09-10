package commerce

import (
	"context"
	"errors"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrInvalidCart = errors.New("invalid cart")

type Cart struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Currency   string     `json:"currency"`
	Items      []CartItem `json:"items"`
	TotalCents int64      `json:"total_cents"`
}
type CartItem struct {
	ID         string `json:"id"`
	CartID     string `json:"cart_id"`
	ProductID  string `json:"product_id"`
	SKU        string `json:"sku"`
	Title      string `json:"title"`
	Quantity   int    `json:"quantity"`
	UnitCents  int64  `json:"unit_cents"`
	TotalCents int64  `json:"total_cents"`
}

type CartStore interface {
	GetOrCreateCart(context.Context, database.Executor, string, string) (Cart, error)
	GetCart(context.Context, database.Executor, string, string) (Cart, error)
	AddCartItem(context.Context, database.Executor, string, string, int) (Cart, error)
	RemoveCartItem(context.Context, database.Executor, string, string) error
}

type CartService struct {
	Transactions database.Transactor
	Store        CartStore
}

func (s CartService) Get(ctx context.Context, db database.Executor, userID string) (Cart, error) {
	if strings.TrimSpace(userID) == "" || s.Store == nil {
		return Cart{}, ErrInvalidCart
	}
	return s.Store.GetOrCreateCart(ctx, db, userID, "TRY")
}
func (s CartService) Add(ctx context.Context, userID, productID string, quantity int) (Cart, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(productID) == "" || quantity < 1 || quantity > 1000 || s.Store == nil {
		return Cart{}, ErrInvalidCart
	}
	var cart Cart
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		cart, err = s.Store.AddCartItem(ctx, tx, userID, productID, quantity)
		return err
	})
	return cart, err
}
func (s CartService) Remove(ctx context.Context, userID, productID string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(productID) == "" || s.Store == nil {
		return ErrInvalidCart
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return s.Store.RemoveCartItem(ctx, tx, userID, productID) })
}

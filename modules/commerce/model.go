package commerce

import (
	"errors"
	"strings"
)

type ProductStatus string

const (
	ProductDraft      ProductStatus = "draft"
	ProductPublished  ProductStatus = "published"
	ProductOutOfStock ProductStatus = "out_of_stock"
)

type Product struct {
	ID         string
	SKU        string
	Title      string
	PriceCents int64
	Currency   string
	Stock      int
	Status     ProductStatus
}

type OrderItem struct {
	ProductID  string `json:"product_id"`
	SKU        string `json:"sku"`
	Title      string `json:"title"`
	Quantity   int    `json:"quantity"`
	UnitCents  int64  `json:"unit_cents"`
	TotalCents int64  `json:"total_cents"`
}

type Order struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	Currency       string      `json:"currency"`
	Items          []OrderItem `json:"items"`
	TotalCents     int64       `json:"total_cents"`
	PaymentStatus  string      `json:"payment_status"`
	ShippingStatus string      `json:"shipping_status"`
}

type Checkout struct {
	ID             string
	UserID         string
	IdempotencyKey string
	Items          []CheckoutItem
}

type CheckoutItem struct {
	ProductID string
	Quantity  int
}

var (
	ErrInvalidCheckout     = errors.New("invalid checkout")
	ErrOutOfStock          = errors.New("product is out of stock")
	ErrIdempotencyConflict = errors.New("idempotency key belongs to another checkout")
)

func (c Checkout) Validate() error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.UserID) == "" || strings.TrimSpace(c.IdempotencyKey) == "" || len(c.Items) == 0 {
		return ErrInvalidCheckout
	}
	seen := make(map[string]struct{}, len(c.Items))
	for _, item := range c.Items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 || item.Quantity > 1000 {
			return ErrInvalidCheckout
		}
		if _, ok := seen[item.ProductID]; ok {
			return ErrInvalidCheckout
		}
		seen[item.ProductID] = struct{}{}
	}
	return nil
}

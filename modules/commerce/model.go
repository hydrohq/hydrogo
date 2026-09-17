package commerce

import (
	"errors"
	"math"
	"strings"
)

type ProductStatus string

const (
	ProductDraft      ProductStatus = "draft"
	ProductPublished  ProductStatus = "published"
	ProductOutOfStock ProductStatus = "out_of_stock"
)

type Product struct {
	ID          string        `json:"id"`
	SKU         string        `json:"sku"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	MediaID     string        `json:"media_id,omitempty"`
	PriceCents  int64         `json:"price_cents"`
	Currency    string        `json:"currency"`
	Stock       int           `json:"stock"`
	Status      ProductStatus `json:"status"`
}

type ProductVariant struct {
	ID         string        `json:"id"`
	ProductID  string        `json:"product_id"`
	SKU        string        `json:"sku"`
	Title      string        `json:"title"`
	PriceCents int64         `json:"price_cents"`
	Currency   string        `json:"currency,omitempty"`
	Stock      int           `json:"stock"`
	Status     ProductStatus `json:"status"`
}

type OrderItem struct {
	ProductID  string `json:"product_id"`
	VariantID  string `json:"variant_id,omitempty"`
	SKU        string `json:"sku"`
	Title      string `json:"title"`
	Quantity   int    `json:"quantity"`
	UnitCents  int64  `json:"unit_cents"`
	TotalCents int64  `json:"total_cents"`
}

type Order struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	IdempotencyHash string          `json:"-"`
	Currency        string          `json:"currency"`
	Items           []OrderItem     `json:"items"`
	TotalCents      int64           `json:"total_cents"`
	PaymentStatus   string          `json:"payment_status"`
	ShippingStatus  string          `json:"shipping_status"`
	Customer        PaymentCustomer `json:"customer,omitempty"`
}

// PaymentCustomer is the immutable checkout snapshot needed by hosted
// payment providers. It is copied into the order so a later payment attempt
// does not depend on mutable account profile data.
type PaymentCustomer struct {
	Name           string `json:"name"`
	IdentityNumber string `json:"-"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	City           string `json:"city"`
	Country        string `json:"country"`
	PostalCode     string `json:"postal_code"`
	ClientIP       string `json:"-"`
}

type Checkout struct {
	ID             string
	UserID         string
	IdempotencyKey string
	CartID         string
	Items          []CheckoutItem
	Customer       PaymentCustomer
}

type CheckoutItem struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id,omitempty"`
	Quantity  int    `json:"quantity"`
}

var (
	ErrInvalidCheckout       = errors.New("invalid checkout")
	ErrOutOfStock            = errors.New("product is out of stock")
	ErrIdempotencyConflict   = errors.New("idempotency key belongs to another checkout")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
)

// ValidateIdempotencyKey keeps retry keys bounded and safe to transport across
// HTTP, logs, and provider adapters. The visible-ASCII contract matches the
// form submission boundary while avoiding control characters and unbounded
// allocation in checkout paths.
func ValidateIdempotencyKey(key string) error {
	if len(key) < 16 || len(key) > 128 {
		return ErrInvalidIdempotencyKey
	}
	for i := 0; i < len(key); i++ {
		if key[i] < 0x21 || key[i] > 0x7e {
			return ErrInvalidIdempotencyKey
		}
	}
	return nil
}

func checkedCentsTotal(unit int64, quantity int) (int64, bool) {
	if unit < 0 || quantity <= 0 {
		return 0, false
	}
	if unit > 0 && int64(quantity) > math.MaxInt64/unit {
		return 0, false
	}
	return unit * int64(quantity), true
}

func checkedCentsAdd(total, line int64) (int64, bool) {
	if total < 0 || line < 0 || total > math.MaxInt64-line {
		return 0, false
	}
	return total + line, true
}

func (c Checkout) Validate() error {
	if strings.TrimSpace(c.ID) == "" || strings.TrimSpace(c.UserID) == "" || ValidateIdempotencyKey(c.IdempotencyKey) != nil || (len(c.Items) == 0 && strings.TrimSpace(c.CartID) == "") {
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

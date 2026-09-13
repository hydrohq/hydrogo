// Package customerorders defines the narrow order projection exposed to the
// member account surface. It intentionally contains no commerce implementation
// details so modules can depend on a stable capability contract.
package customerorders

import (
	"context"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type Item struct {
	ProductID  string `json:"product_id"`
	VariantID  string `json:"variant_id,omitempty"`
	SKU        string `json:"sku"`
	Title      string `json:"title"`
	Quantity   int    `json:"quantity"`
	UnitCents  int64  `json:"unit_cents"`
	TotalCents int64  `json:"total_cents"`
}

type Order struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	Currency       string `json:"currency"`
	Items          []Item `json:"items"`
	TotalCents     int64  `json:"total_cents"`
	PaymentStatus  string `json:"payment_status"`
	ShippingStatus string `json:"shipping_status"`
}

// Reader is the only capability needed by the member account surface.
type Reader interface {
	ListOrdersByUserID(context.Context, database.Executor, string, int) ([]Order, error)
}

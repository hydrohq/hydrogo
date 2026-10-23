// Package commerce is the cart, checkout and order lifecycle module.
// Product/variant/category catalog data is NOT here: it lives in the
// declarative content-type engine (content-types/commerce/), the same way
// content-types/erp/products.yaml already works, since it needs no atomic
// SQL. What's here is only what genuinely needs it: stock reservation,
// cart line items, and order/payment records, per AGENTS.md's own rule —
// modules/news is the reference module for content that needs custom
// behaviour; everything else should be a content type.
package commerce

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidCart        = errors.New("invalid cart")
	ErrInvalidItem        = errors.New("invalid cart item")
	ErrInvalidOrder       = errors.New("invalid order")
	ErrCartNotFound       = errors.New("cart not found")
	ErrItemNotFound       = errors.New("cart item not found")
	ErrOrderNotFound      = errors.New("order not found")
	ErrOutOfStock         = errors.New("not enough stock available")
	ErrVersionConflict    = errors.New("record was changed by someone else")
	ErrRepositoryRequired = errors.New("commerce repository is required")
)

// PaymentStatus and FulfillmentStatus are plain Go enums with guarded
// transitions enforced in service.go -- orders are not content-type
// records, so collections.StateMachine (used by content-types/erp's
// sales_orders.yaml) does not apply here; this is the same concept,
// reimplemented for a type that lives in its own SQL table.
type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "pending"
	PaymentPaid     PaymentStatus = "paid"
	PaymentFailed   PaymentStatus = "failed"
	PaymentRefunded PaymentStatus = "refunded"
)

type FulfillmentStatus string

const (
	FulfillmentUnfulfilled FulfillmentStatus = "unfulfilled"
	FulfillmentPreparing   FulfillmentStatus = "preparing"
	FulfillmentShipped     FulfillmentStatus = "shipped"
	FulfillmentDelivered   FulfillmentStatus = "delivered"
	FulfillmentCancelled   FulfillmentStatus = "cancelled"
)

// Stock is the available-to-sell count for one product variant
// (content-types/commerce/variants.yaml record id). It is the only place
// in this module an atomic, conditional SQL update matters: see
// repository/sqlstore's ReserveStock.
type Stock struct {
	ID        string    `json:"id"`
	VariantID string    `json:"variant_id"`
	Available int       `json:"available"`
	Reserved  int       `json:"reserved"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Cart belongs to exactly one of a logged-in member or an anonymous
// visitor, never both, never neither -- enforced by both the migration's
// CHECK constraint and Validate below.
type Cart struct {
	ID             string    `json:"id"`
	MemberID       string    `json:"member_id,omitempty"`
	AnonymousToken string    `json:"anonymous_token,omitempty"`
	Currency       string    `json:"currency"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// Owner reports whether the cart is a member's or an anonymous visitor's.
func (c Cart) IsMember() bool { return c.MemberID != "" }

type CartItem struct {
	ID             string    `json:"id"`
	CartID         string    `json:"cart_id"`
	VariantID      string    `json:"variant_id"`
	Quantity       int       `json:"quantity"`
	UnitPriceCents int64     `json:"unit_price_cents"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Order is created once, atomically, from a cart at checkout; Validate
// requires an IdempotencyKey because the same checkout request can arrive
// more than once (a retried client request, a double click) and must
// produce exactly one order -- see repository/sqlstore's Checkout.
type Order struct {
	ID              string `json:"id"`
	MemberID        string `json:"member_id,omitempty"`
	GuestName       string `json:"guest_name,omitempty"`
	GuestEmail      string `json:"guest_email,omitempty"`
	GuestPhone      string `json:"guest_phone,omitempty"`
	GuestAddress    string `json:"guest_address,omitempty"`
	GuestCity       string `json:"guest_city,omitempty"`
	GuestCountry    string `json:"guest_country,omitempty"`
	GuestPostalCode string `json:"guest_postal_code,omitempty"`
	IdempotencyKey  string `json:"-"`
	IdempotencyHash string `json:"-"`
	Currency        string `json:"currency"`
	// SubtotalCents is the sum of line items, before any discount, tax or
	// shipping. DiscountCents comes off it; TaxCents and ShippingCents add
	// on top (TaxCents is informational only when tax is price-inclusive --
	// see Service.Pricing's own doc comment -- the rate is already baked
	// into SubtotalCents in that mode). TotalCents is what actually gets
	// charged and is the only field StartPayment ever reads.
	SubtotalCents     int64             `json:"subtotal_cents"`
	DiscountCode      string            `json:"discount_code,omitempty"`
	DiscountCents     int64             `json:"discount_cents"`
	TaxCents          int64             `json:"tax_cents"`
	ShippingCents     int64             `json:"shipping_cents"`
	TotalCents        int64             `json:"total_cents"`
	PaymentStatus     PaymentStatus     `json:"payment_status"`
	FulfillmentStatus FulfillmentStatus `json:"fulfillment_status"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// IsGuest reports whether the order has no member account behind it.
func (o Order) IsGuest() bool { return o.MemberID == "" }

type OrderItem struct {
	ID             string `json:"id"`
	OrderID        string `json:"order_id"`
	VariantID      string `json:"variant_id"`
	SKU            string `json:"sku"`
	Title          string `json:"title"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	TotalCents     int64  `json:"total_cents"`
}

// PaymentCharge is one attempt to pay for an order through a provider
// (PayTR, in Phase 3); an order can have more than one if a first attempt
// failed. PaymentNotification (not modeled as a Go struct -- it has no
// behaviour beyond existing) is the dedup record for provider callbacks.
type PaymentCharge struct {
	ID                string        `json:"id"`
	OrderID           string        `json:"order_id"`
	Provider          string        `json:"provider"`
	ProviderPaymentID string        `json:"provider_payment_id"`
	AmountCents       int64         `json:"amount_cents"`
	Currency          string        `json:"currency"`
	Status            PaymentStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// ValidateCart checks a cart is well-formed before it is persisted.
func ValidateCart(c Cart) error {
	if strings.TrimSpace(c.Currency) == "" {
		return errors.New("cart currency is required")
	}
	hasMember := strings.TrimSpace(c.MemberID) != ""
	hasAnonymous := strings.TrimSpace(c.AnonymousToken) != ""
	if hasMember == hasAnonymous {
		return errors.New("cart needs exactly one of member_id or anonymous_token")
	}
	return nil
}

// ValidateItem checks a cart line item is well-formed before it is persisted.
func ValidateItem(item CartItem) error {
	if strings.TrimSpace(item.VariantID) == "" {
		return errors.New("cart item variant_id is required")
	}
	if item.Quantity <= 0 {
		return errors.New("cart item quantity must be positive")
	}
	if item.UnitPriceCents < 0 {
		return errors.New("cart item price cannot be negative")
	}
	return nil
}

// ValidateOrder checks an order is well-formed before it is persisted.
func ValidateOrder(o Order) error {
	if strings.TrimSpace(o.IdempotencyKey) == "" {
		return errors.New("order idempotency_key is required")
	}
	if strings.TrimSpace(o.Currency) == "" {
		return errors.New("order currency is required")
	}
	if o.TotalCents < 0 {
		return errors.New("order total cannot be negative")
	}
	if o.IsGuest() && strings.TrimSpace(o.GuestEmail) == "" {
		return errors.New("a guest order needs guest_email")
	}
	return nil
}

package commerce

import (
	"errors"
	"math"
	"time"
)

var (
	ErrDiscountCodeNotFound  = errors.New("discount code not found")
	ErrDiscountCodeInactive  = errors.New("discount code is not active")
	ErrDiscountCodeNotYet    = errors.New("discount code is not active yet")
	ErrDiscountCodeExpired   = errors.New("discount code has expired")
	ErrDiscountCodeExhausted = errors.New("discount code has reached its redemption limit")
	ErrDiscountCodeMinimum   = errors.New("order subtotal is below the discount code's minimum")
)

type DiscountKind string

const (
	DiscountPercent DiscountKind = "percent"
	DiscountFixed   DiscountKind = "fixed"
)

// DiscountCode is a promo code redeemed at checkout. RedeemedCount is
// incremented by one atomic, conditional SQL update (see
// repository/sqlstore's RedeemDiscountCode), the same guard ReserveStock
// uses for the same reason: two checkouts racing the last redemption must
// never both succeed.
type DiscountCode struct {
	ID               string       `json:"id"`
	Code             string       `json:"code"`
	Kind             DiscountKind `json:"kind"`
	Value            int64        `json:"value"` // percent: 1-100; fixed: cents
	MinSubtotalCents int64        `json:"min_subtotal_cents"`
	MaxRedemptions   int          `json:"max_redemptions"` // 0 = unlimited
	RedeemedCount    int          `json:"redeemed_count"`
	Active           bool         `json:"active"`
	StartsAt         time.Time    `json:"starts_at"` // zero = no start restriction
	EndsAt           time.Time    `json:"ends_at"`   // zero = no end restriction
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// DiscountFor validates the code against a cart's subtotal and now, and
// returns the cents it takes off -- never more than subtotalCents, so a
// fixed-amount code larger than the order simply zeroes it out rather than
// going negative.
func (d DiscountCode) DiscountFor(subtotalCents int64, now time.Time) (int64, error) {
	if !d.Active {
		return 0, ErrDiscountCodeInactive
	}
	if !d.StartsAt.IsZero() && now.Before(d.StartsAt) {
		return 0, ErrDiscountCodeNotYet
	}
	if !d.EndsAt.IsZero() && now.After(d.EndsAt) {
		return 0, ErrDiscountCodeExpired
	}
	if d.MaxRedemptions > 0 && d.RedeemedCount >= d.MaxRedemptions {
		return 0, ErrDiscountCodeExhausted
	}
	if subtotalCents < d.MinSubtotalCents {
		return 0, ErrDiscountCodeMinimum
	}

	var amount int64
	switch d.Kind {
	case DiscountPercent:
		amount = int64(math.Round(float64(subtotalCents) * float64(d.Value) / 100))
	case DiscountFixed:
		amount = d.Value
	}
	if amount > subtotalCents {
		amount = subtotalCents
	}
	return amount, nil
}

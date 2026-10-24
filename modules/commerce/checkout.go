package commerce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// CheckoutRequest carries what Checkout needs beyond the cart itself.
type CheckoutRequest struct {
	CartID          string
	MemberID        string
	GuestName       string
	GuestEmail      string
	GuestPhone      string
	GuestAddress    string
	GuestCity       string
	GuestDistrict   string
	GuestCountry    string
	GuestPostalCode string
	IdempotencyKey  string
	// Currency is the cart's currency (the store's configured default).
	// Empty falls back to TRY for callers that predate this field.
	Currency string
	// ItemTitles/SKUs let the caller supply catalog data (from the
	// content-type record) for each cart item, keyed by variant id -- the
	// service has no direct access to content-type records. They are
	// display text only; prices always come from Service.Prices.
	ItemTitles map[string]string
	ItemSKUs   map[string]string
	// DiscountCode is optional; an invalid, expired, exhausted or
	// below-minimum code returns ErrDiscountCodeInvalid rather than
	// silently checking out at full price.
	DiscountCode string
}

var (
	// ErrDiscountCodeInvalid wraps every reason a code does not apply, so
	// transport can tell "this code is bad" apart from other failures.
	ErrDiscountCodeInvalid = errors.New("discount code cannot be applied to this order")
	// ErrIdempotencyConflict means the idempotency key already belongs to
	// another cart's order. The other order is never returned.
	ErrIdempotencyConflict = errors.New("idempotency key was already used for a different checkout")
	// ErrEmptyCart means there is nothing to check out.
	ErrEmptyCart = errors.New("cart is empty")
)

// idempotencyHash binds an idempotency key to the cart that used it, so a
// key cannot be replayed from another cart to read someone else's order.
func idempotencyHash(req CheckoutRequest) string {
	sum := sha256.Sum256([]byte(req.CartID + "\x00" + req.MemberID + "\x00" + req.IdempotencyKey))
	return hex.EncodeToString(sum[:])
}

func (r CheckoutRequest) currency() string {
	if strings.TrimSpace(r.Currency) == "" {
		return "TRY"
	}
	return strings.TrimSpace(r.Currency)
}

// Checkout is PlaceOrder without the created flag, for callers that do not
// run side effects.
func (s Service) Checkout(ctx context.Context, req CheckoutRequest) (Order, error) {
	order, _, err := s.PlaceOrder(ctx, req)
	return order, err
}

// PlaceOrder reserves stock for every item in the cart and creates the
// order in one transaction: if any item can't be reserved, nothing is
// reserved and no order exists. Every line is priced from Service.Prices.
// created is false when the idempotency key matched an order this same cart
// already placed; callers must then skip side effects (e-mails, payment
// start) they already ran the first time.
func (s Service) PlaceOrder(ctx context.Context, req CheckoutRequest) (Order, bool, error) {
	if req.IdempotencyKey == "" {
		return Order{}, false, errors.New("checkout requires an idempotency key")
	}
	hash := idempotencyHash(req)
	currency := req.currency()
	prices, err := s.priceCart(ctx, req.CartID, currency)
	if err != nil {
		return Order{}, false, err
	}
	var (
		result  Order
		created bool
	)
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		// A retry returns the original order without re-validating: the
		// discount code would otherwise see its own redemption as
		// "exhausted" and fail a retry that should just succeed again.
		if existing, found, err := s.Repository.GetOrderByIdempotencyKey(ctx, tx, req.IdempotencyKey); err != nil {
			return err
		} else if found {
			if existing.IdempotencyHash != hash {
				return ErrIdempotencyConflict
			}
			result = existing
			return nil
		}

		items, err := s.Repository.ListItems(ctx, tx, req.CartID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return ErrEmptyCart
		}

		orderID, err := newID("order_")
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		var subtotal int64
		orderItems := make([]OrderItem, 0, len(items))
		for _, item := range items {
			unitPrice, ok := prices[item.VariantID]
			if !ok {
				// Added after the prices were read; never charge an unpriced line.
				return ErrCartChanged
			}
			ok, err := s.Repository.ReserveStock(ctx, tx, item.VariantID, item.Quantity)
			if err != nil {
				return err
			}
			if !ok {
				return ErrOutOfStock
			}
			lineTotal := unitPrice * int64(item.Quantity)
			subtotal += lineTotal
			itemID, err := newID("oi_")
			if err != nil {
				return err
			}
			orderItems = append(orderItems, OrderItem{
				ID: itemID, OrderID: orderID, VariantID: item.VariantID,
				SKU: req.ItemSKUs[item.VariantID], Title: req.ItemTitles[item.VariantID],
				Quantity: item.Quantity, UnitPriceCents: unitPrice, TotalCents: lineTotal,
			})
		}

		discountCode, discountCents, err := s.resolveDiscount(ctx, tx, req.DiscountCode, subtotal, now)
		if err != nil {
			return err
		}
		breakdown := s.Pricing.Price(subtotal, discountCents)

		order := Order{
			ID: orderID, MemberID: req.MemberID,
			GuestName: req.GuestName, GuestEmail: req.GuestEmail, GuestPhone: req.GuestPhone,
			GuestAddress: req.GuestAddress, GuestCity: req.GuestCity, GuestDistrict: req.GuestDistrict, GuestCountry: req.GuestCountry, GuestPostalCode: req.GuestPostalCode,
			IdempotencyKey: req.IdempotencyKey, IdempotencyHash: hash, Currency: currency,
			SubtotalCents: breakdown.SubtotalCents, DiscountCents: breakdown.DiscountCents, TaxCents: breakdown.TaxCents, ShippingCents: breakdown.ShippingCents, TotalCents: breakdown.TotalCents,
			PaymentStatus: PaymentPending, FulfillmentStatus: FulfillmentUnfulfilled,
			CreatedAt: now, UpdatedAt: now,
		}
		if discountCode != nil {
			order.DiscountCode = discountCode.Code
		}
		stored, isNew, err := s.Repository.CreateOrder(ctx, tx, order, orderItems)
		if err != nil {
			return err
		}
		if !isNew {
			// Lost a race with a concurrent request carrying the same key:
			// returning the error rolls back this attempt's reservations.
			if stored.IdempotencyHash != hash {
				return ErrIdempotencyConflict
			}
			result = stored
			return errRetryLostRace
		}
		if discountCode != nil {
			applied, err := s.Repository.RedeemDiscountCode(ctx, tx, discountCode.ID, discountCode.MaxRedemptions)
			if err != nil {
				return err
			}
			if !applied {
				// Another checkout took the last redemption between the
				// read above and this guarded UPDATE.
				return ErrDiscountCodeInvalid
			}
		}
		if err := s.Repository.ClearCart(ctx, tx, req.CartID); err != nil {
			return err
		}
		if err := s.emit(ctx, tx, "order.created", stored.ID); err != nil {
			return err
		}
		result, created = stored, true
		return nil
	})
	if errors.Is(err, errRetryLostRace) {
		return result, false, nil
	}
	if err != nil {
		return Order{}, false, err
	}
	return result, created, nil
}

// ErrCartChanged means an item was added while the checkout was pricing the
// cart. Retrying prices the new contents.
var ErrCartChanged = errors.New("the cart changed during checkout")

// priceCart reads the cart in a short transaction, then prices each variant
// outside any transaction (the price source may need its own connection).
func (s Service) priceCart(ctx context.Context, cartID, currency string) (map[string]int64, error) {
	var items []CartItem
	if err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		items, err = s.Repository.ListItems(ctx, tx, cartID)
		return err
	}); err != nil {
		return nil, err
	}
	prices := make(map[string]int64, len(items))
	for _, item := range items {
		if _, done := prices[item.VariantID]; done {
			continue
		}
		price, err := s.unitPrice(ctx, item.VariantID, currency)
		if err != nil {
			return nil, err
		}
		prices[item.VariantID] = price
	}
	return prices, nil
}

// errRetryLostRace rolls back a checkout that lost the idempotency race;
// the winner's order is returned to the caller instead of an error.
var errRetryLostRace = errors.New("checkout lost the idempotency race")

func (s Service) resolveDiscount(ctx context.Context, tx database.Executor, code string, subtotal int64, now time.Time) (*DiscountCode, int64, error) {
	if strings.TrimSpace(code) == "" {
		return nil, 0, nil
	}
	discount, err := s.Repository.GetDiscountCodeByCode(ctx, tx, code)
	if err != nil {
		if errors.Is(err, ErrDiscountCodeNotFound) {
			return nil, 0, ErrDiscountCodeInvalid
		}
		return nil, 0, err
	}
	cents, err := discount.DiscountFor(subtotal, now)
	if err != nil {
		return nil, 0, ErrDiscountCodeInvalid
	}
	return &discount, cents, nil
}

package commerce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// Repository is the subset of repository/sqlstore.Store the service needs,
// narrowed to an interface per modules/news's pattern so Service can be
// tested without a real database.
type Repository interface {
	EnsureStock(ctx context.Context, db database.Executor, variantID string, available int) error
	GetStock(ctx context.Context, db database.Executor, variantID string) (Stock, error)
	ReserveStock(ctx context.Context, db database.Executor, variantID string, quantity int) (bool, error)
	ReleaseReservation(ctx context.Context, db database.Executor, variantID string, quantity int) error
	CommitReservation(ctx context.Context, db database.Executor, variantID string, quantity int) error

	CreateCart(ctx context.Context, db database.Executor, cart Cart) error
	GetCartByMember(ctx context.Context, db database.Executor, memberID string) (Cart, bool, error)
	GetCartByAnonymousToken(ctx context.Context, db database.Executor, token string) (Cart, bool, error)
	AddOrUpdateItem(ctx context.Context, db database.Executor, item CartItem) error
	RemoveItem(ctx context.Context, db database.Executor, cartID, itemID string) error
	ListItems(ctx context.Context, db database.Executor, cartID string) ([]CartItem, error)

	CreateOrder(ctx context.Context, db database.Executor, order Order, items []OrderItem) (Order, bool, error)
	GetOrder(ctx context.Context, db database.Executor, id string) (Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, db database.Executor, key string) (Order, bool, error)
	ListOrderItems(ctx context.Context, db database.Executor, orderID string) ([]OrderItem, error)
	ListPendingOrdersOlderThan(ctx context.Context, db database.Executor, cutoff time.Time, limit int) ([]Order, error)
	UpdatePaymentStatus(ctx context.Context, db database.Executor, orderID string, from, to PaymentStatus) (bool, error)
	RecordPaymentNotification(ctx context.Context, db database.Executor, provider, eventID string) (bool, error)
	// ListNotifiedOrdersNotPaid and StockReservationDrift back
	// Service.Reconcile -- a read-only consistency sweep, not part of the
	// checkout/payment path itself. See reconcile.go.
	ListNotifiedOrdersNotPaid(ctx context.Context, db database.Executor, provider string, limit int) ([]Order, error)
	StockReservationDrift(ctx context.Context, db database.Executor, limit int) ([]ReservationDrift, error)

	GetDiscountCodeByCode(ctx context.Context, db database.Executor, code string) (DiscountCode, error)
	RedeemDiscountCode(ctx context.Context, db database.Executor, id string, maxRedemptions int) (bool, error)
	// CreateDiscountCode is not called by Checkout; it is here only so the
	// admin transport handler can reach it through the same narrow
	// interface setStock already uses for EnsureStock.
	CreateDiscountCode(ctx context.Context, db database.Executor, code DiscountCode) error

	// RecordShipment backs Service.CreateShipment: a conditional UPDATE
	// guarded by fulfillment_status, the same atomic-transition pattern as
	// UpdatePaymentStatus, so calling it twice for the same order (an
	// operator double-clicking "ship") only ever calls the carrier and
	// writes these columns once.
	RecordShipment(ctx context.Context, db database.Executor, orderID, carrier, trackingNumber, trackingURL string, shippedAt time.Time) (bool, error)
}

// OutboxWriter is the subset of internal/outbox/sqlstore.Store the service
// needs to publish order lifecycle events, narrowed per modules/news's
// OutboxWriter pattern so Service stays testable without a real outbox.
type OutboxWriter interface {
	Enqueue(ctx context.Context, tx database.Executor, id, topic, aggregateID string, payloadVersion int) error
}

type Service struct {
	Transactions database.Transactor
	Repository   Repository
	// CartTTL is how long an idle cart (and its stock reservations made at
	// checkout) lives before worker.go reclaims it. Defaults to 30 minutes.
	CartTTL time.Duration
	// Outbox is optional; when set, Checkout and ConfirmPayment publish
	// order.created/order.paid for cmd/hydro's webhook delivery. Nil is a
	// valid, fully functional configuration -- an inactive commerce module
	// adds no events, and a project with no configured webhooks need not
	// pay for enqueuing them.
	Outbox OutboxWriter
	// Pricing is operator-set tax/shipping configuration, applied to every
	// checkout; see PricingConfig's own doc comment. The zero value adds
	// no tax and no shipping, exactly reproducing the total-is-just-the-
	// subtotal behaviour this module had before Pricing existed.
	Pricing PricingConfig
}

// emit enqueues one order lifecycle event, deterministically keyed so a
// retried caller (Checkout's idempotent retry, ConfirmPayment's redelivered
// webhook) can never enqueue the same event twice -- the outbox's own
// dedup, matching modules/news/service.go's events() helper.
func (s Service) emit(ctx context.Context, db database.Executor, topic, aggregateID string) error {
	if s.Outbox == nil {
		return nil
	}
	return s.Outbox.Enqueue(ctx, db, "commerce-outbox-"+topic+"-"+aggregateID, topic, aggregateID, 1)
}

func (s Service) cartTTL() time.Duration {
	if s.CartTTL <= 0 {
		return 30 * time.Minute
	}
	return s.CartTTL
}

func newID(prefix string) (string, error) {
	raw := make([]byte, 15)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(raw), nil
}

// GetOrCreateCart returns the caller's existing cart (by member id or
// anonymous token -- exactly one must be set) or creates a new, empty one.
func (s Service) GetOrCreateCart(ctx context.Context, db database.Executor, memberID, anonymousToken, currency string) (Cart, error) {
	if (memberID == "") == (anonymousToken == "") {
		return Cart{}, ErrInvalidCart
	}
	var (
		cart  Cart
		found bool
		err   error
	)
	if memberID != "" {
		cart, found, err = s.Repository.GetCartByMember(ctx, db, memberID)
	} else {
		cart, found, err = s.Repository.GetCartByAnonymousToken(ctx, db, anonymousToken)
	}
	if err != nil {
		return Cart{}, err
	}
	if found {
		return cart, nil
	}
	id, err := newID("cart_")
	if err != nil {
		return Cart{}, err
	}
	now := time.Now().UTC()
	cart = Cart{ID: id, MemberID: memberID, AnonymousToken: anonymousToken, Currency: currency, CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(s.cartTTL())}
	if err := s.Repository.CreateCart(ctx, db, cart); err != nil {
		return Cart{}, err
	}
	return cart, nil
}

// AddItem adds or updates a line item's desired quantity. Stock is not
// reserved here -- only at Checkout -- so browsing and adding to cart never
// contends with other shoppers' purchases; only the actual purchase attempt
// does.
func (s Service) AddItem(ctx context.Context, db database.Executor, cartID, variantID string, quantity int, unitPriceCents int64) error {
	id, err := newID("item_")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	item := CartItem{ID: id, CartID: cartID, VariantID: variantID, Quantity: quantity, UnitPriceCents: unitPriceCents, CreatedAt: now, UpdatedAt: now}
	return s.Repository.AddOrUpdateItem(ctx, db, item)
}

// CheckoutRequest carries what Checkout needs beyond the cart itself.
type CheckoutRequest struct {
	CartID          string
	MemberID        string
	GuestName       string
	GuestEmail      string
	GuestPhone      string
	GuestAddress    string
	GuestCity       string
	GuestCountry    string
	GuestPostalCode string
	IdempotencyKey  string
	// ItemTitles/SKUs let the caller supply catalog data (from the
	// content-type record) for each cart item, keyed by variant id -- the
	// service has no direct access to content-type records.
	ItemTitles map[string]string
	ItemSKUs   map[string]string
	// DiscountCode is optional; an invalid, expired, exhausted or
	// below-minimum code returns ErrDiscountCodeInvalid rather than
	// silently checking out at full price, so the storefront can show the
	// shopper why it didn't apply.
	DiscountCode string
}

// ErrDiscountCodeInvalid wraps whichever specific discount error
// DiscountCode.DiscountFor returned, so transport can tell "this code is
// bad" apart from every other checkout failure without matching on the
// exact underlying reason.
var ErrDiscountCodeInvalid = errors.New("discount code cannot be applied to this order")

// Checkout reserves stock for every item in the cart and creates the order,
// all inside one transaction: if ANY item can't be reserved (someone else
// bought the last one first), the whole attempt rolls back -- nothing is
// reserved, no order is created, and the caller gets ErrOutOfStock naming
// which item. A retried request with the same IdempotencyKey returns the
// original order instead of reserving stock twice (CreateOrder's own
// idempotency, inherited here).
func (s Service) Checkout(ctx context.Context, req CheckoutRequest) (Order, error) {
	if req.IdempotencyKey == "" {
		return Order{}, errors.New("checkout requires an idempotency key")
	}
	var result Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		// A retried request (the same idempotency key) returns the
		// original order outright, without re-validating anything --
		// re-checking the discount code here would otherwise see its own
		// first successful redemption as "exhausted" and incorrectly fail
		// a retry that should just succeed again with the same result.
		if existing, found, err := s.Repository.GetOrderByIdempotencyKey(ctx, tx, req.IdempotencyKey); err != nil {
			return err
		} else if found {
			result = existing
			return nil
		}

		items, err := s.Repository.ListItems(ctx, tx, req.CartID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("cart is empty")
		}

		orderID, err := newID("order_")
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		var subtotal int64
		orderItems := make([]OrderItem, 0, len(items))
		for _, item := range items {
			ok, err := s.Repository.ReserveStock(ctx, tx, item.VariantID, item.Quantity)
			if err != nil {
				return err
			}
			if !ok {
				return ErrOutOfStock
			}
			lineTotal := item.UnitPriceCents * int64(item.Quantity)
			subtotal += lineTotal
			itemID, err := newID("oi_")
			if err != nil {
				return err
			}
			orderItems = append(orderItems, OrderItem{
				ID: itemID, OrderID: orderID, VariantID: item.VariantID,
				SKU: req.ItemSKUs[item.VariantID], Title: req.ItemTitles[item.VariantID],
				Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, TotalCents: lineTotal,
			})
		}

		var discountCode DiscountCode
		var hasDiscountCode bool
		if strings.TrimSpace(req.DiscountCode) != "" {
			discountCode, err = s.Repository.GetDiscountCodeByCode(ctx, tx, req.DiscountCode)
			if err != nil {
				if errors.Is(err, ErrDiscountCodeNotFound) {
					return ErrDiscountCodeInvalid
				}
				return err
			}
			if _, discountErr := discountCode.DiscountFor(subtotal, now); discountErr != nil {
				return ErrDiscountCodeInvalid
			}
			hasDiscountCode = true
		}
		var discountCents int64
		if hasDiscountCode {
			discountCents, _ = discountCode.DiscountFor(subtotal, now)
		}
		breakdown := s.Pricing.Price(subtotal, discountCents)

		order := Order{
			ID: orderID, MemberID: req.MemberID,
			GuestName: req.GuestName, GuestEmail: req.GuestEmail, GuestPhone: req.GuestPhone,
			GuestAddress: req.GuestAddress, GuestCity: req.GuestCity, GuestCountry: req.GuestCountry, GuestPostalCode: req.GuestPostalCode,
			IdempotencyKey: req.IdempotencyKey, IdempotencyHash: req.IdempotencyKey,
			SubtotalCents: breakdown.SubtotalCents, DiscountCents: breakdown.DiscountCents, TaxCents: breakdown.TaxCents, ShippingCents: breakdown.ShippingCents, TotalCents: breakdown.TotalCents,
			PaymentStatus: PaymentPending, FulfillmentStatus: FulfillmentUnfulfilled,
			CreatedAt: now, UpdatedAt: now,
		}
		if hasDiscountCode {
			order.DiscountCode = discountCode.Code
		}
		if order.Currency == "" {
			order.Currency = "TRY"
		}
		created, isNew, err := s.Repository.CreateOrder(ctx, tx, order, orderItems)
		if err != nil {
			return err
		}
		if isNew && hasDiscountCode {
			if _, err := s.Repository.RedeemDiscountCode(ctx, tx, discountCode.ID, discountCode.MaxRedemptions); err != nil {
				return err
			}
		}
		if isNew {
			if err := s.emit(ctx, tx, "order.created", created.ID); err != nil {
				return err
			}
		} else {
			// A retried request: the reservations made above in THIS
			// transaction are rolled back automatically (the order already
			// exists from the original attempt, which did its own
			// reservation) -- returning the existing order is correct, but
			// we must not keep this attempt's redundant reservations.
			for _, item := range items {
				if err := s.Repository.ReleaseReservation(ctx, tx, item.VariantID, item.Quantity); err != nil {
					return err
				}
			}
		}
		result = created
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	return result, nil
}

// ConfirmPayment is called once a payment provider's callback is verified
// (see modules/commerce/providers in Phase 3). It is itself idempotent: a
// redelivered notification (provider, providerEventID) is recorded at most
// once (RecordPaymentNotification), and the payment_status transition only
// applies once (UpdatePaymentStatus's conditional UPDATE) -- so calling this
// twice for the same event is safe and does not double-commit stock or send
// a second confirmation.
func (s Service) ConfirmPayment(ctx context.Context, db database.Executor, provider, providerEventID, orderID string) (bool, error) {
	isNewEvent, err := s.Repository.RecordPaymentNotification(ctx, db, provider, providerEventID)
	if err != nil {
		return false, err
	}
	if !isNewEvent {
		return false, nil
	}
	applied, err := s.Repository.UpdatePaymentStatus(ctx, db, orderID, PaymentPending, PaymentPaid)
	if err != nil {
		return false, err
	}
	if !applied {
		return false, nil
	}
	items, err := s.Repository.ListOrderItems(ctx, db, orderID)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if err := s.Repository.CommitReservation(ctx, db, item.VariantID, item.Quantity); err != nil {
			return false, err
		}
	}
	if err := s.emit(ctx, db, "order.paid", orderID); err != nil {
		return false, err
	}
	return true, nil
}

// ErrOrderNotShippable is returned when CreateShipment is asked to ship an
// order that has not been paid yet, or that has already been shipped (or
// moved past shipped) -- handing an unpaid order to a carrier would commit
// a real shipping cost against a sale that may never settle, and calling
// the carrier again for an already-shipped order would create a second
// real-world shipment, not just a harmless duplicate write.
var ErrOrderNotShippable = errors.New("order is not paid and unshipped, so it cannot be shipped")

// CreateShipment hands a paid order to the given carrier and records the
// tracking details it returns. It is deliberately not wired into
// ConfirmPayment: shipping is a separate, usually manual (an operator
// packs the order, or a schedule batches several) step from payment, not
// an automatic consequence of it, so Checkout's transport calls this only
// when an operator (or a future auto-fulfillment job) explicitly asks for
// it -- see transport/http.go's createShipment handler.
//
// Unlike ConfirmPayment (which dedups a payment webhook that already
// happened out in the world before it ever calls this service), the
// carrier call here IS the real-world side effect, so the eligibility
// check happens before calling the carrier, not only at the final write:
// a retried request against an order that is no longer unfulfilled/
// preparing is rejected without ever touching the carrier again.
func (s Service) CreateShipment(ctx context.Context, db database.Executor, provider ShippingProvider, orderID string) (Order, error) {
	order, err := s.Repository.GetOrder(ctx, db, orderID)
	if err != nil {
		return Order{}, err
	}
	if order.PaymentStatus != PaymentPaid || (order.FulfillmentStatus != FulfillmentUnfulfilled && order.FulfillmentStatus != FulfillmentPreparing) {
		return Order{}, ErrOrderNotShippable
	}
	items, err := s.Repository.ListOrderItems(ctx, db, orderID)
	if err != nil {
		return Order{}, err
	}
	result, err := provider.CreateShipment(ctx, order, items)
	if err != nil {
		return Order{}, err
	}
	shippedAt := time.Now().UTC()
	applied, err := s.Repository.RecordShipment(ctx, db, orderID, provider.Name(), result.TrackingNumber, result.TrackingURL, shippedAt)
	if err != nil {
		return Order{}, err
	}
	if applied {
		if err := s.emit(ctx, db, "order.fulfilled", orderID); err != nil {
			return Order{}, err
		}
	}
	return s.Repository.GetOrder(ctx, db, orderID)
}

// ExpireOrder is worker.go's reservation-expiry job, one order at a time: an
// order whose payment never completed within the hold window is moved from
// pending to failed and its items' stock reservations are released back to
// available. Reservations belong to the ORDER's items (made atomically at
// Checkout), not the cart's current contents -- the cart may have changed
// or emptied since checkout, so releasing by cart would release the wrong
// quantities.
//
// Guarded the same way ConfirmPayment is: the payment_status transition
// only applies once (pending -> failed), so calling this twice for the same
// order (the worker re-scanning before the first pass's effects are
// visible, or being re-run after a crash) releases stock exactly once, not
// per call. If the order was already paid in the meantime, the transition
// fails to apply and nothing is released -- a payment can never be
// silently undone by an expiry sweep that was simply too slow.
func (s Service) ExpireOrder(ctx context.Context, db database.Executor, orderID string) (bool, error) {
	applied, err := s.Repository.UpdatePaymentStatus(ctx, db, orderID, PaymentPending, PaymentFailed)
	if err != nil {
		return false, err
	}
	if !applied {
		return false, nil
	}
	items, err := s.Repository.ListOrderItems(ctx, db, orderID)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if err := s.Repository.ReleaseReservation(ctx, db, item.VariantID, item.Quantity); err != nil {
			return false, err
		}
	}
	return true, nil
}

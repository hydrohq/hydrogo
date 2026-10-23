package commerce

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	ClearCart(ctx context.Context, db database.Executor, cartID string) error

	CreateOrder(ctx context.Context, db database.Executor, order Order, items []OrderItem) (Order, bool, error)
	GetOrder(ctx context.Context, db database.Executor, id string) (Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, db database.Executor, key string) (Order, bool, error)
	ListOrderItems(ctx context.Context, db database.Executor, orderID string) ([]OrderItem, error)
	ListPendingOrdersOlderThan(ctx context.Context, db database.Executor, cutoff time.Time, limit int) ([]Order, error)
	// ListOrders backs the admin order-browsing endpoint; either status
	// filter as "" skips it. See repository/sqlstore's own doc comment for
	// the pagination convention (limit+1 rows, no COUNT query).
	ListOrders(ctx context.Context, db database.Executor, paymentStatus PaymentStatus, fulfillmentStatus FulfillmentStatus, limit, offset int) ([]Order, error)
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

	// ClaimShipment, ReleaseShipmentClaim and RecordShipment are
	// Service.CreateShipment's three guarded steps: claim a paid,
	// unfulfilled order (unfulfilled -> preparing) before calling the
	// carrier, give the claim back if the carrier refuses, and record the
	// tracking details (preparing -> shipped). A claim older than
	// staleBefore can be taken again, so a crash mid-call does not strand
	// the order.
	ClaimShipment(ctx context.Context, db database.Executor, orderID string, staleBefore time.Time) (bool, error)
	ReleaseShipmentClaim(ctx context.Context, db database.Executor, orderID string) error
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
	// Prices is the authoritative unit price for every cart line; see
	// PriceSource. Checkout fails with ErrNoPriceSource when it is nil.
	Prices PriceSource
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

// AddItem adds or updates a line item's desired quantity, priced from
// Service.Prices (never from the caller). Stock is not reserved here --
// only at Checkout. The stored price is for display; Checkout prices every
// line again, so a price change between add and pay is honoured.
//
// Do not call AddItem inside a transaction when Prices reads the database:
// with a one-connection pool the lookup waits for the connection the
// transaction holds. AddToCart prices first, then opens its transaction.
func (s Service) AddItem(ctx context.Context, db database.Executor, cart Cart, variantID string, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	unitPrice, err := s.unitPrice(ctx, variantID, cart.Currency)
	if err != nil {
		return err
	}
	return s.addLine(ctx, db, cart, variantID, quantity, unitPrice)
}

func (s Service) addLine(ctx context.Context, db database.Executor, cart Cart, variantID string, quantity int, unitPrice int64) error {
	id, err := newID("item_")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	item := CartItem{ID: id, CartID: cart.ID, VariantID: variantID, Quantity: quantity, UnitPriceCents: unitPrice, CreatedAt: now, UpdatedAt: now}
	return s.Repository.AddOrUpdateItem(ctx, db, item)
}

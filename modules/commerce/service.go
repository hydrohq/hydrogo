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

	CreateOrder(ctx context.Context, db database.Executor, order Order, items []OrderItem) (Order, bool, error)
	GetOrder(ctx context.Context, db database.Executor, id string) (Order, error)
	ListOrderItems(ctx context.Context, db database.Executor, orderID string) ([]OrderItem, error)
	ListPendingOrdersOlderThan(ctx context.Context, db database.Executor, cutoff time.Time, limit int) ([]Order, error)
	UpdatePaymentStatus(ctx context.Context, db database.Executor, orderID string, from, to PaymentStatus) (bool, error)
	RecordPaymentNotification(ctx context.Context, db database.Executor, provider, eventID string) (bool, error)
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
}

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
		var total int64
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
			total += lineTotal
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

		order := Order{
			ID: orderID, MemberID: req.MemberID,
			GuestName: req.GuestName, GuestEmail: req.GuestEmail, GuestPhone: req.GuestPhone,
			GuestAddress: req.GuestAddress, GuestCity: req.GuestCity, GuestCountry: req.GuestCountry, GuestPostalCode: req.GuestPostalCode,
			IdempotencyKey: req.IdempotencyKey, IdempotencyHash: req.IdempotencyKey,
			TotalCents: total, PaymentStatus: PaymentPending, FulfillmentStatus: FulfillmentUnfulfilled,
			CreatedAt: now, UpdatedAt: now,
		}
		if order.Currency == "" {
			order.Currency = "TRY"
		}
		created, isNew, err := s.Repository.CreateOrder(ctx, tx, order, orderItems)
		if err != nil {
			return err
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

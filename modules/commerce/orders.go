package commerce

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hydrohq/hydrogo/internal/database"
)

// Every method in this file runs its reads and writes inside
// Transactions.Within. On PostgreSQL that is also what binds the tenant for
// row-level security; a call made on the raw pool skips RLS entirely.

var (
	// ErrLatePayment means a provider confirmed payment for an order the
	// expiry sweep had already failed and released. Money was taken for an
	// order that no longer holds stock: an operator must refund or restore
	// it. The notification is recorded so redeliveries stay quiet.
	ErrLatePayment = errors.New("payment arrived for an order that had already expired")
	// ErrPaymentAmountMismatch means the provider reports a different
	// amount than the order total. The order is not marked paid.
	ErrPaymentAmountMismatch = errors.New("paid amount does not match the order total")
	// ErrOrderNotPayable means the order is no longer awaiting payment.
	ErrOrderNotPayable = errors.New("this order is no longer awaiting payment")
	// ErrOrderNotShippable is returned when CreateShipment is asked to ship
	// an order that is unpaid, already shipped, or being shipped right now.
	ErrOrderNotShippable = errors.New("order is not paid and unshipped, so it cannot be shipped")
)

// shipmentClaimTTL is how long a shipment claim blocks a retry. A claim
// older than this belongs to a request that crashed mid carrier call.
const shipmentClaimTTL = 10 * time.Minute

// Payment is a provider's verified statement that an order was paid.
type Payment struct {
	Provider string
	// EventID is the provider's own redelivery key; a repeat is a no-op.
	EventID string
	OrderID string
	// ProviderPaymentID is the provider's own id for the payment (what a
	// refund needs); EventID is used when it is empty.
	ProviderPaymentID string
	// AmountCents is what the provider says it collected. It must cover the
	// order total; more is allowed (installment interest is added by some
	// providers on top of the order amount).
	AmountCents int64
}

// OrderByNumber loads an order by its human-readable order_number (or by ID when
// the input starts with "order_").
func (s Service) OrderByNumber(ctx context.Context, code string) (Order, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(code), "#"))
	if trimmed == "" {
		return Order{}, ErrOrderNotFound
	}
	if strings.HasPrefix(trimmed, "order_") {
		return s.Order(ctx, trimmed)
	}
	finder, ok := s.Repository.(interface {
		GetOrderByNumber(context.Context, database.Executor, string) (Order, error)
	})
	if !ok {
		return Order{}, ErrOrderNotFound
	}
	var order Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		order, err = finder.GetOrderByNumber(ctx, tx, trimmed)
		return err
	})
	return order, err
}

// ConfirmPayment applies a verified payment notification. It is idempotent:
// a redelivered event is recorded at most once and the pending -> paid
// transition applies at most once. applied is true only for the call that
// actually moved the order to paid. ErrLatePayment, ErrPaymentAmountMismatch
// and ErrOrderNotFound are returned after the notification is durably
// recorded; the caller should acknowledge the provider and alert a human.
func (s Service) ConfirmPayment(ctx context.Context, payment Payment) (bool, error) {
	var (
		applied bool
		outcome error
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		isNewEvent, err := s.Repository.RecordPaymentNotification(ctx, tx, payment.Provider, payment.EventID)
		if err != nil || !isNewEvent {
			return err
		}
		order, err := s.Repository.GetOrder(ctx, tx, payment.OrderID)
		if errors.Is(err, ErrOrderNotFound) {
			outcome = ErrOrderNotFound
			return nil
		}
		if err != nil {
			return err
		}
		if payment.AmountCents < order.TotalCents {
			outcome = fmt.Errorf("%w: paid %d, order total %d", ErrPaymentAmountMismatch, payment.AmountCents, order.TotalCents)
			return nil
		}
		ok, err := s.Repository.UpdatePaymentStatus(ctx, tx, payment.OrderID, PaymentPending, PaymentPaid)
		if err != nil {
			return err
		}
		if !ok {
			current, err := s.Repository.GetOrder(ctx, tx, payment.OrderID)
			if err != nil {
				return err
			}
			if current.PaymentStatus == PaymentFailed {
				outcome = ErrLatePayment
				// Stock is deliberately not touched: it was released when the order failed and may be sold already.
				slog.ErrorContext(ctx, "CRITICAL: Late payment on cancelled order", "order_id", payment.OrderID, "provider", payment.Provider, "amount_cents", payment.AmountCents)
				providerPaymentID := payment.ProviderPaymentID
				if providerPaymentID == "" {
					providerPaymentID = payment.EventID
				}
				chargeID, err := newID("chg_")
				if err != nil {
					return err
				}
				if err := s.Repository.RecordPaymentCharge(ctx, tx, PaymentCharge{
					ID:                chargeID,
					OrderID:           payment.OrderID,
					Provider:          payment.Provider,
					ProviderPaymentID: providerPaymentID,
					AmountCents:       payment.AmountCents,
					Currency:          current.Currency,
					Status:            PaymentFailed,
				}); err != nil {
					return err
				}
				rrqID, err := newID("rrq_")
				if err != nil {
					return err
				}
				if err := s.Repository.RecordRefundRequired(ctx, tx, RefundRequired{
					ID:          rrqID,
					OrderID:     payment.OrderID,
					Provider:    payment.Provider,
					AmountCents: payment.AmountCents,
					Currency:    current.Currency,
					Reason:      "ErrLatePayment: payment received after order failed or expired",
					CreatedAt:   time.Now().UTC(),
				}); err != nil {
					return err
				}
			}
			return nil
		}
		items, err := s.Repository.ListOrderItems(ctx, tx, payment.OrderID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := s.Repository.CommitReservation(ctx, tx, item.VariantID, item.Quantity); err != nil {
				return err
			}
		}
		providerPaymentID := payment.ProviderPaymentID
		if providerPaymentID == "" {
			providerPaymentID = payment.EventID
		}
		chargeID, err := newID("chg_")
		if err != nil {
			return err
		}
		if err := s.Repository.RecordPaymentCharge(ctx, tx, PaymentCharge{ID: chargeID, OrderID: payment.OrderID, Provider: payment.Provider, ProviderPaymentID: providerPaymentID, AmountCents: payment.AmountCents, Currency: order.Currency, Status: PaymentPaid}); err != nil {
			return err
		}
		if err := s.emit(ctx, tx, "order.paid", payment.OrderID); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
		if database.IsUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return applied, outcome
}

// ExpireOrder is the reservation-expiry job for one order: pending -> failed
// and its items' reservations released, in one transaction. The guarded
// transition makes a repeat call (or a payment that won the race) a no-op.
func (s Service) ExpireOrder(ctx context.Context, orderID string) (bool, error) {
	var applied bool
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		ok, err := s.Repository.UpdatePaymentStatus(ctx, tx, orderID, PaymentPending, PaymentFailed)
		if err != nil || !ok {
			return err
		}
		items, err := s.Repository.ListOrderItems(ctx, tx, orderID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if err := s.Repository.ReleaseReservation(ctx, tx, item.VariantID, item.Quantity); err != nil {
				return err
			}
		}
		// The coupon redemption taken at checkout is returned too, so the shopper can use the code again.
		order, err := s.Repository.GetOrder(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if order.DiscountCode != "" && !strings.HasPrefix(order.DiscountCode, MethodDiscountPrefix) {
			if err := s.Repository.ReleaseDiscountCode(ctx, tx, order.DiscountCode); err != nil {
				return err
			}
		}
		applied = true
		return nil
	})
	return applied, err
}

// CreateShipment hands a paid order to the carrier. The order is claimed
// (unfulfilled -> preparing) in its own short transaction before the carrier
// is called, so two concurrent requests cannot both reach the carrier and no
// database lock is held during the network call. A carrier failure gives
// the claim back; success records the tracking details (preparing ->
// shipped) and emits order.fulfilled.
func (s Service) CreateShipment(ctx context.Context, provider ShippingProvider, orderID string) (Order, error) {
	var (
		order Order
		items []OrderItem
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		claimed, err := s.Repository.ClaimShipment(ctx, tx, orderID, time.Now().UTC().Add(-shipmentClaimTTL))
		if err != nil {
			return err
		}
		if !claimed {
			if _, err := s.Repository.GetOrder(ctx, tx, orderID); err != nil {
				return err
			}
			return ErrOrderNotShippable
		}
		if order, err = s.Repository.GetOrder(ctx, tx, orderID); err != nil {
			return err
		}
		items, err = s.Repository.ListOrderItems(ctx, tx, orderID)
		return err
	})
	if err != nil {
		return Order{}, err
	}

	result, carrierErr := provider.CreateShipment(ctx, order, items)
	if carrierErr != nil {
		releaseErr := s.Transactions.Within(ctx, func(tx database.Executor) error {
			return s.Repository.ReleaseShipmentClaim(ctx, tx, orderID)
		})
		return Order{}, errors.Join(carrierErr, releaseErr)
	}

	var shipped Order
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		if result.Carrier == "" {
			result.Carrier = provider.Name()
		}
		applied, err := s.Repository.RecordShipment(ctx, tx, orderID, result, time.Now().UTC())
		if err != nil {
			return err
		}
		if !applied {
			return fmt.Errorf("shipment %s for order %s was created at the carrier but the order left the preparing state; record it by hand", result.TrackingNumber, orderID)
		}
		if err := s.emit(ctx, tx, "order.fulfilled", orderID); err != nil {
			return err
		}
		shipped, err = s.Repository.GetOrder(ctx, tx, orderID)
		return err
	})
	return shipped, err
}

// ListShipmentOffers queries an offer-capable shipping provider for available
// carrier quotes on a paid, unshipped order without locking its claim state.
func (s Service) ListShipmentOffers(ctx context.Context, provider ShippingProvider, orderID string) (ShipmentOfferDraft, error) {
	offerer, ok := provider.(ShipmentOfferer)
	if !ok {
		return ShipmentOfferDraft{}, ErrShippingActionUnsupported
	}
	var (
		order Order
		items []OrderItem
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if order, err = s.Repository.GetOrder(ctx, tx, orderID); err != nil {
			return err
		}
		if order.PaymentStatus != PaymentPaid || (order.FulfillmentStatus != FulfillmentUnfulfilled && order.FulfillmentStatus != FulfillmentPreparing) {
			return ErrOrderNotShippable
		}
		items, err = s.Repository.ListOrderItems(ctx, tx, orderID)
		return err
	})
	if err != nil {
		return ShipmentOfferDraft{}, err
	}
	return offerer.ListShipmentOffers(ctx, order, items)
}

// CancelShipment cancels an already created carrier shipment and returns the
// order to unfulfilled so an operator can create a replacement shipment.
func (s Service) CancelShipment(ctx context.Context, provider ShippingProvider, orderID string) (Order, error) {
	canceller, ok := provider.(ShipmentOrderCanceller)
	if !ok {
		return Order{}, ErrShippingActionUnsupported
	}
	var order Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if order, err = s.Repository.GetOrder(ctx, tx, orderID); err != nil {
			return err
		}
		if order.FulfillmentStatus != FulfillmentShipped {
			return ErrOrderNotCancellableShipment
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	if err := canceller.CancelOrderShipment(ctx, order); err != nil {
		return Order{}, err
	}
	var updated Order
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		applied, err := s.Repository.CancelShipmentRecord(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if !applied {
			return ErrOrderNotCancellableShipment
		}
		updated, err = s.Repository.GetOrder(ctx, tx, orderID)
		return err
	})
	return updated, err
}

// OrderByShipmentID returns the order linked to a provider shipment id.
func (s Service) OrderByShipmentID(ctx context.Context, shipmentID string) (Order, error) {
	var order Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		order, err = s.Repository.GetOrderByShipmentID(ctx, tx, shipmentID)
		return err
	})
	return order, err
}

// SyncShipmentDelivery refreshes tracking details from an authoritative
// carrier lookup and transitions a shipped order to delivered once.
func (s Service) SyncShipmentDelivery(ctx context.Context, shipmentID, trackingNumber, trackingURL, labelURL string, delivered bool, deliveredAt time.Time) (Order, bool, error) {
	var (
		updated Order
		applied bool
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		order, err := s.Repository.GetOrderByShipmentID(ctx, tx, shipmentID)
		if err != nil {
			return err
		}
		var nextNum, nextURL, nextLabel string
		if trackingNumber != "" && trackingNumber != order.ShippingTrackingNumber {
			nextNum = trackingNumber
		}
		if trackingURL != "" && trackingURL != order.ShippingTrackingURL {
			nextURL = trackingURL
		}
		if labelURL != "" && labelURL != order.ShippingLabelURL {
			nextLabel = labelURL
		}
		if nextNum != "" || nextURL != "" || nextLabel != "" {
			if err := s.Repository.UpdateShipmentTracking(ctx, tx, order.ID, nextNum, nextURL, nextLabel); err != nil {
				return err
			}
		}
		if delivered {
			when := deliveredAt
			if when.IsZero() {
				when = time.Now().UTC()
			}
			ok, err := s.Repository.MarkOrderDelivered(ctx, tx, order.ID, when)
			if err != nil {
				return err
			}
			if ok {
				applied = true
				if err := s.emit(ctx, tx, "order.delivered", order.ID); err != nil {
					return err
				}
			}
		}
		updated, err = s.Repository.GetOrder(ctx, tx, order.ID)
		return err
	})
	return updated, applied, err
}

// StartPayment (re)starts a payment for a pending order, for when the first
// attempt at checkout failed (a network blip to the provider).
func (s Service) StartPayment(ctx context.Context, provider PaymentProvider, orderID string) (PaymentStart, error) {
	order, items, err := s.OrderWithItems(ctx, orderID)
	if err != nil {
		return PaymentStart{}, err
	}
	if order.PaymentStatus != PaymentPending {
		return PaymentStart{}, ErrOrderNotPayable
	}
	start, err := provider.StartPayment(ctx, order, items)
	if err != nil {
		return PaymentStart{}, err
	}
	method := strings.TrimSpace(start.Method)
	if method == "" {
		method = strings.TrimSpace(PaymentMethod(ctx))
	}
	if (method != "" && method != order.PaymentMethod) || (start.ProviderRef != "" && start.ProviderRef != order.PaymentRef) || method != "havale" {
		if err := s.Transactions.Within(ctx, func(tx database.Executor) error {
			if err := s.Repository.UpdateOrderPaymentAttempt(ctx, tx, order.ID, method, start.ProviderRef); err != nil {
				return err
			}
			if method != "havale" {
				return s.Repository.ExtendOrderHold(ctx, tx, order.ID, time.Now().UTC().Add(25*time.Minute))
			}
			return nil
		}); err != nil {
			return PaymentStart{}, err
		}
	}
	return start, nil
}

// ExtendOrderHold pushes back an unpaid order's reservation expiry deadline
// (for example, while a card payment sits under gateway fraud review).
func (s Service) ExtendOrderHold(ctx context.Context, orderID string, holdUntil time.Time) error {
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Repository.ExtendOrderHold(ctx, tx, orderID, holdUntil)
	})
}

// Order reads one order by ID.
func (s Service) Order(ctx context.Context, orderID string) (Order, error) {
	var order Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		order, err = s.Repository.GetOrder(ctx, tx, orderID)
		return err
	})
	return order, err
}

// OrderWithItems reads one order and its line items.
func (s Service) OrderWithItems(ctx context.Context, orderID string) (Order, []OrderItem, error) {
	var (
		order Order
		items []OrderItem
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if order, err = s.Repository.GetOrder(ctx, tx, orderID); err != nil {
			return err
		}
		items, err = s.Repository.ListOrderItems(ctx, tx, orderID)
		return err
	})
	return order, items, err
}

// ListOrders is the admin order list; see Repository.ListOrders for the
// limit+1 paging convention.
func (s Service) ListOrders(ctx context.Context, paymentStatus PaymentStatus, fulfillmentStatus FulfillmentStatus, limit, offset int) ([]Order, error) {
	var orders []Order
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		orders, err = s.Repository.ListOrders(ctx, tx, paymentStatus, fulfillmentStatus, limit, offset)
		return err
	})
	return orders, err
}

package commerce

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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
	// AmountCents is what the provider says it collected. It must cover the
	// order total; more is allowed (installment interest is added by some
	// providers on top of the order amount).
	AmountCents int64
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
		if err := s.emit(ctx, tx, "order.paid", payment.OrderID); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
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
		applied, err := s.Repository.RecordShipment(ctx, tx, orderID, provider.Name(), result.TrackingNumber, result.TrackingURL, time.Now().UTC())
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
	return provider.StartPayment(ctx, order, items)
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

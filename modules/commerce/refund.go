package commerce

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hydrohq/hydrogo/internal/database"
)

// PaymentRefunder is implemented by a payment provider that can return money.
// It refunds the full charge; the order's own total is charge.AmountCents or less.
type PaymentRefunder interface {
	Refund(ctx context.Context, order Order, charge PaymentCharge) error
}

type refundConfig struct {
	AllowShipped bool
}

// RefundOption configures refund behaviour.
type RefundOption func(*refundConfig)

// WithAllowShipped permits refunding an order whose fulfillment status is shipped or delivered.
func WithAllowShipped(allow bool) RefundOption {
	return func(c *refundConfig) {
		c.AllowShipped = allow
	}
}

// RefundOrder returns a paid order's money through its provider.
// Flow:
// 1. Checks order is in paid state and not already refunded/pending (guards against double refunds).
// 2. Checks fulfillment status: shipped/delivered requires explicit WithAllowShipped(true).
// 3. Atomically flips payment_status to 'refund_pending' and generates an idempotency key before calling the provider.
// 4. Calls refunder.Refund.
// 5. On timeout/provider error, status remains 'refund_pending' (NOT reverted to paid) and alerts operator.
// 6. On success, transitions to 'refunded', and restores stock only if the order was not shipped.
func (s Service) RefundOrder(ctx context.Context, refunder PaymentRefunder, orderID string, opts ...RefundOption) (Order, error) {
	var cfg refundConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var (
		order  Order
		charge PaymentCharge
	)

	// Step 1: Validate state and claim 'refund_pending' atomically before calling provider.
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if order, err = s.Repository.GetOrder(ctx, tx, orderID); err != nil {
			return err
		}

		if order.PaymentStatus == PaymentRefundPending || order.PaymentStatus == PaymentRefunded {
			return ErrRefundAlreadyInProgress
		}
		if order.PaymentStatus != PaymentPaid {
			return ErrOrderNotRefundable
		}

		// Shipment status check: require permission when shipped/delivered
		if order.FulfillmentStatus == FulfillmentShipped || order.FulfillmentStatus == FulfillmentDelivered {
			if !cfg.AllowShipped {
				return ErrOrderShippedRequiresApproval
			}
		}

		if charge, err = s.Repository.LatestPaymentCharge(ctx, tx, orderID); err != nil {
			return err
		}

		ok, err := s.Repository.UpdatePaymentStatus(ctx, tx, orderID, PaymentPaid, PaymentRefundPending)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRefundAlreadyInProgress
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}

	// Step 2: Call provider with idempotency key
	order.IdempotencyKey = "ref_" + orderID
	if refundErr := refunder.Refund(ctx, order, charge); refundErr != nil {
		// On timeout or provider error: DO NOT revert status back to paid.
		// Leave it in 'refund_pending' to prevent double refunds and alert operator.
		_ = s.Transactions.Within(ctx, func(tx database.Executor) error {
			return s.emit(ctx, tx, "order.refund_failed", orderID)
		})
		slog.ErrorContext(ctx, "payment provider refund failed; order left in refund_pending", "order_id", orderID, "error", refundErr)
		return Order{}, fmt.Errorf("provider refund failed: %w", refundErr)
	}

	// Step 3: Provider succeeded -> finalize to 'refunded' and restore stock only if unshipped.
	var refunded Order
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		ok, err := s.Repository.UpdatePaymentStatus(ctx, tx, orderID, PaymentRefundPending, PaymentRefunded)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRefundAlreadyInProgress
		}

		// Only restock when the order has not been shipped
		if order.FulfillmentStatus != FulfillmentShipped && order.FulfillmentStatus != FulfillmentDelivered {
			items, err := s.Repository.ListOrderItems(ctx, tx, orderID)
			if err != nil {
				return err
			}
			for _, item := range items {
				if err := s.Repository.ReturnStock(ctx, tx, item.VariantID, item.Quantity); err != nil {
					return err
				}
			}
			if order.DiscountCode != "" && !strings.HasPrefix(order.DiscountCode, MethodDiscountPrefix) {
				if err := s.Repository.ReleaseDiscountCode(ctx, tx, order.DiscountCode); err != nil {
					return err
				}
			}
		}

		if err := s.emit(ctx, tx, "order.refunded", orderID); err != nil {
			return err
		}
		refunded, err = s.Repository.GetOrder(ctx, tx, orderID)
		return err
	})
	return refunded, err
}

package commerce

import (
	"context"
	"errors"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var (
	// ErrNoPaymentCharge means no provider payment is recorded for the order
	// (it was paid before charges were recorded, or never paid).
	ErrNoPaymentCharge = errors.New("no payment is recorded for this order")
	// ErrOrderNotRefundable means the order is not in the paid state.
	ErrOrderNotRefundable = errors.New("only a paid order can be refunded")
)

// PaymentRefunder is implemented by a payment provider that can return money.
// It refunds the full charge; the order's own total is charge.AmountCents or less.
type PaymentRefunder interface {
	Refund(ctx context.Context, order Order, charge PaymentCharge) error
}

// RefundOrder returns a paid order's money through its provider. The order is
// claimed first (paid -> refunded, guarded) so two concurrent requests cannot
// both reach the provider; a provider failure gives the claim back. Stock is
// not returned automatically: an operator decides whether the goods came back.
func (s Service) RefundOrder(ctx context.Context, refunder PaymentRefunder, orderID string) (Order, error) {
	var (
		order  Order
		charge PaymentCharge
	)
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		if order, err = s.Repository.GetOrder(ctx, tx, orderID); err != nil {
			return err
		}
		if order.PaymentStatus != PaymentPaid {
			return ErrOrderNotRefundable
		}
		if charge, err = s.Repository.LatestPaymentCharge(ctx, tx, orderID); err != nil {
			return err
		}
		ok, err := s.Repository.UpdatePaymentStatus(ctx, tx, orderID, PaymentPaid, PaymentRefunded)
		if err != nil {
			return err
		}
		if !ok {
			return ErrOrderNotRefundable
		}
		return nil
	})
	if err != nil {
		return Order{}, err
	}
	if refundErr := refunder.Refund(ctx, order, charge); refundErr != nil {
		revertErr := s.Transactions.Within(ctx, func(tx database.Executor) error {
			_, err := s.Repository.UpdatePaymentStatus(ctx, tx, orderID, PaymentRefunded, PaymentPaid)
			return err
		})
		return Order{}, errors.Join(refundErr, revertErr)
	}
	var refunded Order
	err = s.Transactions.Within(ctx, func(tx database.Executor) error {
		if err := s.emit(ctx, tx, "order.refunded", orderID); err != nil {
			return err
		}
		var err error
		refunded, err = s.Repository.GetOrder(ctx, tx, orderID)
		return err
	})
	return refunded, err
}

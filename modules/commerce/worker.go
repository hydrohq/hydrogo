package commerce

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// Worker is the reservation-expiry sweep: an order whose payment never
// completed within Hold is expired, releasing its stock back to available.
// This is a periodic SQL sweep (mirrors modules/news's own worker), not a
// job-queue consumer: there is one recurring thing to do, not a stream of
// discrete one-shot tasks.
type Worker struct {
	Service Service
	// Hold is how long an order may stay payment_status=pending before its
	// reservation is released. Defaults to 30 minutes, matching Service's
	// own CartTTL default.
	Hold time.Duration
}

func (w Worker) hold() time.Duration {
	if w.Hold <= 0 {
		return 30 * time.Minute
	}
	return w.Hold
}

// ExpireDue finds pending orders older than Hold and expires each one in
// its own transaction, returning how many were actually expired (a payment
// that won the race is not counted -- see Service.ExpireOrder).
func (w Worker) ExpireDue(ctx context.Context, now time.Time) (int, error) {
	cutoff := now.Add(-w.hold())
	var due []Order
	err := w.Service.Transactions.Within(ctx, func(tx database.Executor) error {
		var err error
		due, err = w.Service.Repository.ListPendingOrdersOlderThan(ctx, tx, cutoff, 100)
		return err
	})
	if err != nil {
		return 0, err
	}
	expired := 0
	for _, order := range due {
		ok, err := w.Service.ExpireOrder(ctx, order.ID)
		if err != nil {
			return expired, err
		}
		if ok {
			expired++
		}
	}
	return expired, nil
}

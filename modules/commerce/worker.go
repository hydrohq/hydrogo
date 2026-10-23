package commerce

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// Worker is the reservation-expiry sweep: an order whose payment never
// completed within Hold is expired, releasing its stock back to available.
// This is a periodic SQL sweep (mirrors modules/news's own worker, which
// scans for due scheduled items the same way), not a job-queue consumer --
// there is one thing to do (expire stale pending orders), not a stream of
// discrete one-shot tasks, so internal/jobs's lease/ack queue (built for
// the latter) does not fit here.
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

// ExpireDue finds pending orders older than Hold and expires each one,
// returning how many were actually expired (a concurrent payment
// confirmation racing the sweep is not counted -- see Service.ExpireOrder's
// own idempotent guard).
func (w Worker) ExpireDue(ctx context.Context, db database.Executor, now time.Time) (int, error) {
	cutoff := now.Add(-w.hold())
	due, err := w.Service.Repository.ListPendingOrdersOlderThan(ctx, db, cutoff, 100)
	if err != nil {
		return 0, err
	}
	expired := 0
	for _, order := range due {
		ok, err := w.Service.ExpireOrder(ctx, db, order.ID)
		if err != nil {
			return expired, err
		}
		if ok {
			expired++
		}
	}
	return expired, nil
}

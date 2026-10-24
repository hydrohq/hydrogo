package commerce

import (
	"context"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// ReconcileConfig controls Reconcile's one time-based threshold. Zero
// values use safe defaults; a project never has to configure this to get a
// working safety net.
type ReconcileConfig struct {
	// Hold should mirror the Worker's own Hold (the payment window after
	// which a pending order's reservation is released) -- not read from
	// Worker directly, since Reconcile may run in a separate process (the
	// `hydro commerce reconcile` CLI) that never constructs one. Defaults
	// to 30 minutes, the same default Worker.hold() uses.
	Hold time.Duration
	// StaleAfter is how much further past Hold a pending order must sit
	// before Reconcile flags it. A pending order just past Hold is not yet
	// an anomaly -- the sweep (Worker.ExpireDue, ticking every minute in
	// cmd/hydro/routes_commerce.go) simply hasn't reached it yet. One still
	// pending Hold+StaleAfter later means the sweep itself did not run.
	StaleAfter time.Duration
}

func (c ReconcileConfig) hold() time.Duration {
	if c.Hold <= 0 {
		return 30 * time.Minute
	}
	return c.Hold
}

func (c ReconcileConfig) staleAfter() time.Duration {
	if c.StaleAfter <= 0 {
		return 15 * time.Minute
	}
	return c.StaleAfter
}

// ReconcileReport is Reconcile's findings. Every field is an anomaly that
// never happens in healthy operation; a non-empty report is a signal for
// an operator (or an alerting job built on top of this) to look, not
// something Reconcile tries to fix itself -- it only reads, so running it
// can never make an inconsistency worse. See
// docs/decisions/0018-commerce-reconciliation.md for why this exists: no
// amount of test coverage proves a payment integration correct under real
// traffic, only catching real drift early and loudly does.
type ReconcileReport struct {
	GeneratedAt time.Time `json:"generated_at"`
	// StalePendingOrders: payment never completed within Hold, and the
	// worker's own expiry sweep -- which should already have failed these
	// -- has not caught up. Means the sweep is not running at all (crashed,
	// never started, wrong deployment), not that payment is merely slow;
	// every item here is still holding stock reserved from sale.
	StalePendingOrders []Order `json:"stale_pending_orders,omitempty"`
	// NotifiedButNotPaid: the configured payment provider's own webhook
	// was received and recorded (RecordPaymentNotification saw it), but
	// the order's payment_status never flipped to "paid". The one scenario
	// this exists specifically to catch: a payment the provider confirms
	// receiving, sitting on an order marked failed or still pending --
	// money potentially taken for an order the shopper may be told was
	// never completed. Empty when Reconcile is called with no provider
	// configured (nothing to compare against).
	NotifiedButNotPaid []Order `json:"notified_but_not_paid,omitempty"`
	// ReservationDrift: commerce_stock.reserved does not match what
	// currently-pending orders actually account for. A release that never
	// ran leaves stock reserved (and therefore unsellable) forever; one
	// that ran twice frees stock that is still actually spoken for.
	ReservationDrift []ReservationDrift `json:"reservation_drift,omitempty"`
}

// Clean reports whether Reconcile found nothing wrong -- the common case,
// and the only result that should ever be silent.
func (r ReconcileReport) Clean() bool {
	return len(r.StalePendingOrders) == 0 && len(r.NotifiedButNotPaid) == 0 && len(r.ReservationDrift) == 0
}

// Reconcile is a read-only consistency sweep over cart/checkout/payment
// state, meant to run on a schedule (cmd/hydro's own periodic tick,
// mirroring Worker.ExpireDue) or on demand (`hydro commerce reconcile`).
// provider is the payment provider to check NotifiedButNotPaid against
// ("paytr", matching what ConfirmPayment is called with in
// modules/commerce/providers/paytr/transport.go); empty skips that check,
// which is the correct, non-alarming result for a project with no payment
// provider configured at all.
func (s Service) Reconcile(ctx context.Context, db database.Executor, now time.Time, cfg ReconcileConfig, provider string, limit int) (ReconcileReport, error) {
	if limit <= 0 {
		limit = 100
	}
	cutoff := now.Add(-(cfg.hold() + cfg.staleAfter()))
	stale, err := s.Repository.ListPendingOrdersOlderThan(ctx, db, cutoff, limit)
	if err != nil {
		return ReconcileReport{}, err
	}
	var notified []Order
	if provider != "" {
		notified, err = s.Repository.ListNotifiedOrdersNotPaid(ctx, db, provider, limit)
		if err != nil {
			return ReconcileReport{}, err
		}
	}
	drift, err := s.Repository.StockReservationDrift(ctx, db, limit)
	if err != nil {
		return ReconcileReport{}, err
	}
	return ReconcileReport{
		GeneratedAt:        now.UTC(),
		StalePendingOrders: stale,
		NotifiedButNotPaid: notified,
		ReservationDrift:   drift,
	}, nil
}

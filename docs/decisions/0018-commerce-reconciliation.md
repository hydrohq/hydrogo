# 0018: Commerce reconciliation and drift alerting

- Status: Accepted
- Date: 2026-10-23

## Context

[0012](0012-commerce-catalog-as-content-types.md) through
[0017](0017-commerce-tax-shipping-discounts-currencies.md) built a complete
commerce path -- catalog, stock, cart, checkout, PayTR, tax, shipping,
discounts -- each phase verified against a real `hydro serve` instance
before merging. That verification proves the code is correct against the
scenarios it was run through; it cannot prove the integration is correct
under real traffic, because no amount of simulated load reproduces what
only a real payment gateway, real network conditions and real shopper
behaviour produce. The honest next step before a real launch is not more
simulated testing -- it is going live with real but low-volume traffic,
with a way to notice immediately if anything drifts, rather than finding
out from a customer.

Three places in this module can silently drift without ever returning an
error to anyone:

- A pending order whose payment never completes. `Worker.ExpireDue`
  already handles this ([0013](0013-commerce-cart-checkout-module.md)) --
  but if the worker crashes, is never started (a deployment config gap),
  or is quietly failing, pending orders accumulate with their stock
  reserved forever, and nothing surfaces that today.
- A payment provider's notification landing (`RecordPaymentNotification`
  returns true -- a genuinely new event) while the order's own
  `payment_status` transition fails to apply (`UpdatePaymentStatus`'s
  conditional `UPDATE` finds no matching row, for example because the
  order was already expired to `failed` by the worker moments before a
  delayed webhook arrived). The provider believes the payment succeeded;
  the order disagrees. This is the single highest-stakes failure mode in
  the whole module -- money potentially taken for an order a shopper may
  be told was never completed -- and nothing checks for it today.
- `commerce_stock.reserved` and what currently-pending orders actually
  account for silently diverging, if a release or a commit runs twice, or
  never runs, due to a bug not yet hit by any existing test.

## Decision

`Service.Reconcile` (`modules/commerce/reconcile.go`) is a **read-only**
sweep over existing data, deliberately not a repair tool -- it only
reports, so running it can never make a real inconsistency worse:

- **`StalePendingOrders`**: reuses the existing
  `ListPendingOrdersOlderThan` past `Hold + StaleAfter` (defaults 30m +
  15m) -- a pending order this old should already have been expired by
  `Worker.ExpireDue`; still being pending past that window means the sweep
  itself isn't running, not that payment is merely slow.
- **`NotifiedButNotPaid`**: a new repository query joining
  `commerce_payment_notifications` to `commerce_customer_orders` (for
  PayTR, `provider_event_id` IS the order id -- see
  `modules/commerce/providers/paytr/transport.go`) for orders whose
  `payment_status` isn't `paid` despite a recorded notification. Skipped
  entirely when no provider is passed in, rather than reported as an
  anomaly against a project with no payment provider configured.
- **`ReservationDrift`**: two separate queries (stock reserved per
  variant; pending-order quantities per variant) diffed in Go, not one SQL
  join -- a correct full-outer-join comparison needs syntax SQLite doesn't
  support, and this module runs across SQLite, PostgreSQL and MySQL alike.

Exposed two ways, both calling the same `Service.Reconcile`:
`hydro commerce reconcile [--provider paytr] [--hold] [--stale-after]
[--json]` (`cmd/hydro/commerce_cmd.go`) for an on-demand report -- before a
launch, after an incident, or from an operator's own cron feeding their
alerting -- using the same unscoped, cross-tenant database access
`hydro jobs dead-letter` already does, since this is operator tooling, not
a request-scoped API call; and `runCommerceReconciliationAlert`
(`cmd/hydro/routes_commerce.go`), a plain ticker alongside the existing
reservation-expiry sweeper, logging a structured `slog` warning every five
minutes whenever the report isn't clean, for an operator's existing log
pipeline to turn into a real alert.

What this phase explicitly does not add: automatic correction of anything
Reconcile finds (a human must look -- stale orders and reservation
mismatches found this way may need a refund decision, not just a database
UPDATE), and a charge-amount-vs-provider-reported-amount check
(`commerce_payment_charges` is a table with a Go model but nothing in this
module writes to it yet -- `ConfirmPayment` only touches
`commerce_payment_notifications` and the order's own `payment_status`;
building a check against data nothing populates would always report either
false positives or nothing at all, so this phase does not pretend
otherwise).

## Consequences

- Verified against a real `hydro serve` instance, not mocked: scaffolded a
  fresh `--preset commerce` project, enabled the `commerce` module,
  migrated, seeded a stale pending order, a stock row with `reserved=2`
  and no matching pending-order quantity, and a `paytr` notification for
  an order still `pending` -- `hydro commerce reconcile --provider paytr
  --json` correctly reported all three, and reported a clean, empty
  result against the same project before seeding anything.
- `modules/commerce/reconcile_test.go` covers all three checks plus the
  clean case against the real SQLite-backed service (the same pattern
  `service_test.go` already uses), not a hand-rolled fake repository.
- A future phase should wire `commerce_payment_charges` writes into
  `StartPayment`/`ConfirmPayment` so a real charge-amount reconciliation
  becomes possible; this phase deliberately leaves that gap named rather
  than building on top of it.

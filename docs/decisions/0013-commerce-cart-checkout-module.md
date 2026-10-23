# 0013: `modules/commerce` for stock, cart, checkout and orders

- Status: Accepted
- Date: 2026-10-23

## Context

[0012](0012-commerce-catalog-as-content-types.md) put the commerce catalog
(categories, products, variants) in the declarative content-type engine. What
it deliberately left out is anything that needs atomic, high-frequency SQL:
selling the last unit of a variant without overselling it, an idempotent
checkout, and a payment webhook that can be redelivered without
double-fulfilling an order. The generic collections store only has optimistic
version-locking, which cannot give a single conditional "decrement if enough
is available" write; that needs a dedicated table and a plain `UPDATE ...
WHERE available >= ?`, which is what commerce was removed from core for in
[0001](0001-cms-focus.md) in the first place, just scoped down this time to
exactly the four things that need it.

The old commerce module's cart was member-only (`commerce_carts.user_id
UNIQUE`); nothing in HydroGo gives an anonymous visitor a durable identity.
Guest checkout needs one.

## Decision

Add `modules/commerce`, mirroring `modules/news`'s layering
(`model.go` → `service.go` → `repository/sqlstore` → `transport`, plus a
`worker.go`), new tables and new migration IDs (never reusing the old
`commerce_*` names: the old baseline group is still in
`migrations/baseline/manifest.yml` and would collide on a fresh database that
enables the module):

- `commerce_stock(variant_id, available, reserved)`: `ReserveStock` is one
  conditional `UPDATE ... SET available = available - ?, reserved = reserved
  + ? WHERE variant_id = ? AND available >= ?`; `rows affected == 0` means
  sold out. No retry loop, no new concurrency primitive.
- `commerce_shopping_carts` / `commerce_shopping_cart_items`: a cart belongs
  to exactly one of `member_id` or `anonymous_token`
  (`CHECK ((member_id IS NULL) != (anonymous_token IS NULL))`). An anonymous
  visitor gets an `HttpOnly`, `SameSite=Lax` cookie on first cart write,
  minted the same way `modules/members/transport` mints its session cookie.
  A logged-in member's cart is resolved by `MemberID`, an optional hook on
  `transport.Handler` that `cmd/hydro` fills in only when `members` is also
  enabled — commerce does not import `modules/members`.
- `commerce_customer_orders` / `commerce_customer_order_items`: guest fields
  (`guest_name`, `guest_email`, ...) always present; `UNIQUE(tenant_id,
  idempotency_key)` makes `Checkout` idempotent regardless of whether the
  order has a member.
- `commerce_payment_charges` / `commerce_payment_notifications`: the latter's
  `PRIMARY KEY(provider, provider_event_id)` is the dedup for a payment
  provider's webhook (built in Phase 3); `ConfirmPayment`'s own transition is
  additionally guarded by a conditional `UPDATE ... WHERE payment_status =
  'pending'`, so a redelivered webhook can never double-commit stock or
  double-fire `order.paid`.

`Checkout` reserves stock for every cart item inside one transaction; if any
item is out of stock the whole attempt rolls back, including items already
reserved earlier in the same loop. `ExpireOrder` (the reservation-expiry
sweep in `worker.go`, a plain ticker like `runStaleMediaTempSweeper`, not
`internal/jobs`'s lease queue — this is one recurring thing to do, not a
stream of discrete tasks) releases a stale pending order's stock, guarded the
same way so it can never undo a payment that completed first.

`order.created` (already in the event catalog, unused since the old module
was removed) and the new, additive `order.paid` are emitted through the
existing outbox when a project has webhooks configured; `order.fulfilled` is
registered but not yet emitted — there is no fulfillment action in this
phase. `MemberOrdersReader` implements the pre-existing
`internal/contracts/customerorders.DetailReader` contract so
`modules/members`' `/member/orders` and `/api/v1/mobile/member/orders` work
against real orders again, without members importing commerce.

## Consequences

- `internal/config/validate.go`'s `commerce is deferred` guard is removed;
  `internal/security/cors.go`, `internal/tenant/backfill.go` and the two
  tenant/module-isolation contract tests are updated to the new table names
  (the catalog tables they used to reference, `commerce_products` and
  `commerce_product_variants`, no longer exist — catalog reads go through
  `/api/v1/content/products`, outside this allowlist).
- A new `config/presets/commerce.yaml` deployment preset. The scaffold
  `commerce` preset (`hydro new --preset commerce`, from 0012) is a separate
  concern (content types only) and is not changed here; wiring the
  `commerce` module into that preset's module list is a follow-up once 0012
  merges, so this PR does not depend on it landing first.
- Verified against a real `hydro serve` instance: two concurrent checkouts
  racing the last unit of a variant (one `201`, one `409`, stock ending at
  `available=0 reserved=1`, never negative or double-reserved), an anonymous
  cart cookie surviving across requests, a cart re-associating from an
  anonymous cookie to `member_id` on login, and `order.created` landing in
  `outbox_events`.

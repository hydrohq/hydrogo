# 0021: Admin order screen, order district, and parking payment/carrier verification

- Status: Accepted
- Date: 2026-10-24

## Context

After [0019](0019-commerce-order-notifications-and-member-login.md) and
[0020](0020-commerce-shipping-provider.md), two gaps were named rather than
fixed: there was no admin screen for commerce orders at all (an operator
could only reach an order by calling the API with its id), and
`commerce.Order` carried a city but no district (`ilçe`), which every
Turkish carrier's schema wants separately.

A third question was whether to keep pushing on the PayTR and Yurtiçi
Kargo integrations. Both are at the point where the only remaining work is
verification against the real services, and both require things this
template project does not have: a registered merchant with a Turkish
business entity for PayTR (its "test mode" runs on a real merchant
account, there is no credential-free sandbox), and a corporate shipping
contract plus an account representative for Yurtiçi Kargo (no public API
signup, no downloadable WSDL). **There is no store yet.** Chasing
credentials would mean inventing a business to register, which is not
this project's job, and building more against an unverified schema
would just pile speculation on speculation.

## Decision

**Payment and carrier integrations stay exactly as they are.** PayTR
([0014](0014-commerce-paytr-provider.md)) and Yurtiçi Kargo
([0020](0020-commerce-shipping-provider.md)) are structurally complete
and unit-tested, honestly labelled as unverified against the live
services, and will be verified by whichever project first adopts this
module with a real merchant/carrier account. No further work on them
until then. This is a deliberate scope decision, not a gap that was
missed.

**What was built instead — the parts that need no external account:**

- `/admin/commerce`, a Svelte admin screen
  (`admin/src/routes/commerce/CommerceOrdersView.svelte`) listing orders
  newest first with a payment-status filter, client-side search by order
  id/name/email, and a detail modal showing customer, shipping address,
  line items, the pricing breakdown, and — for a paid, unshipped order —
  a "hand to carrier" action that calls the existing
  `POST /api/v1/admin/commerce/orders/{id}/shipment`. Once shipped, the
  modal shows carrier, tracking number, shipped-at and a tracking link
  instead of the button. The action is shown only to users with
  `admin.commerce.write`; the server remains the authority. A 501 from the
  server (no carrier configured) is explained in plain words rather than
  surfaced as a raw error. Navigation entry and header title appear only
  when the `commerce` module is on, per AGENTS.md rule 5.
- Two protected endpoints behind it: `GET /api/v1/admin/commerce/orders`
  (`limit`/`offset`, optional `payment_status`/`fulfillment_status`,
  `limit+1` over-fetch for `has_more` with no COUNT query — the same
  convention `internal/contenttypes`' admin list uses) and
  `GET /api/v1/admin/commerce/orders/{id}`, which is the existing public
  `getOrder` handler registered a second time under a protected path: an
  admin session is simply a second, stronger way to pass the same "know
  the order id" access check, so there is nothing separate to implement.
- `guest_district` on orders (migration `202610241500`), threaded through
  `CheckoutRequest`, the checkout API, the storefront's checkout form
  (now required there), and the Yurtiçi Kargo provider, which sends it as
  `townName` and falls back to the city only for orders that predate the
  field.
- `repository/sqlstore/store.go` crossed the 800-line limit with the
  listing query and was split: order persistence now lives in
  `orders.go`, same package.

## Consequences

- `go run ./tools/check` passes; the storefront's `npm run check`,
  `build` and `smoke` pass.
- Looked at, not just type-checked: the list and the detail modal were
  screenshotted at desktop (1440) and phone (390) widths against a real
  `hydro serve` instance with a real paid order, via `e2e/screenshot.mjs`
  and a Playwright click-through for the modal. No JavaScript errors on
  either. On the phone width the table scrolls horizontally inside its
  own container, the same behaviour the Members screen has.
- Still open, and still named rather than hidden: no payment-retry or
  refund action in the admin (the public `POST .../orders/{id}/payment`
  retry exists for shoppers; an admin-initiated one was not needed yet);
  no `delivered`/`cancelled` transitions from the admin (nothing in the
  module sets them today either); and the two integrations above remain
  unverified until a real account exists.

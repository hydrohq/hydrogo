# 0017: Tax, shipping, discount codes and display currencies

- Status: Accepted
- Date: 2026-10-23

## Context

The commerce phases through [0016](0016-commerce-starter-catalog-sections.md)
covered the minimum viable checkout: an order's total was just the sum of
its line items. A real launch needs KDV/VAT, a shipping fee, promo codes,
and ideally prices a visitor can see in more than one currency. None of
these are generic CMS features — they are commerce-specific business rules
that belong in `modules/commerce`, configured by the operator rather than
hardcoded, since every one of them (the tax rate, whether prices are
tax-inclusive, the shipping fee, whether a currency switcher exists at
all) is a decision only the business running the store can make, and
differs store to store.

Multi-currency in particular was scoped deliberately narrow: a storefront
can **show** prices in USD/EUR using a static, operator-maintained
conversion rate, but checkout and PayTR always charge in one configured
currency. Real multi-currency settlement (charging a card in the
currency it shows) brings FX risk and settlement complexity a static-rate
display switcher does not, and was not asked for.

## Decision

Four additions to `modules/commerce`, all config-driven, all with a zero
value that reproduces the exact pre-existing behaviour (an order's total
is just its subtotal) so no project is affected until it opts in:

- **`PricingConfig`** (`modules/commerce/pricing.go`): a pure function,
  `Price(subtotalCents, discountCents) PricingBreakdown`, computing
  tax (inclusive -- backed out of the price for the receipt, not added on
  top -- or exclusive) and a flat shipping fee (optionally waived above a
  subtotal threshold). `Service.Pricing` holds it; `cmd/hydro` builds it
  from new `commerce.tax.{rate,inclusive}` and
  `commerce.shipping.{flat_rate_cents,free_above_cents}` config.
- **Discount codes** (`modules/commerce/discount.go` +
  `repository/sqlstore/discount.go`, new `commerce_discount_codes` table,
  migration `202610240001_commerce_checkout_pricing` since
  `commerce_customer_orders` -- gaining `subtotal_cents`, `discount_code`,
  `discount_cents`, `tax_cents`, `shipping_cents` -- already shipped and
  may not be edited): percentage or fixed, an optional minimum subtotal,
  date window and redemption limit. `RedeemDiscountCode` increments
  `redeemed_count` with one conditional `UPDATE ... WHERE max_redemptions
  = 0 OR redeemed_count < max_redemptions`, the same atomic-guard pattern
  `ReserveStock` uses for stock, for the same reason: two checkouts racing
  the last redemption must never both succeed. `POST
  /api/v1/admin/commerce/discount-codes` creates one; `checkout` accepts
  `discount_code`.
- **Checkout's own idempotency got a real fast path.** Validating a
  discount code on *every* call, retries included, was wrong: a retried
  request (the same idempotency key) would see its own first successful
  redemption as "exhausted" and fail a checkout that should just return
  the original order again. `Checkout` now checks
  `GetOrderByIdempotencyKey` first and returns the existing order
  immediately when found, before touching stock or discount validation at
  all -- the concurrent-race case (two first-time requests, same new key)
  still falls through to `CreateOrder`'s own unique-constraint dedup
  exactly as before.
- **Display currencies** (`config.Commerce.Currencies`): `default`,
  `display` and a static `rates` map. `GET /api/v1/commerce/currencies`
  and `GET /api/v1/commerce/checkout-config` are public, static config
  echoes (no DB read) so a storefront can render a currency switcher and
  show tax/shipping terms before the shopper commits to checkout; neither
  endpoint is authoritative for what checkout actually charges.

`examples/commerce-starter` gained `/para-birimi/[code]` (a plain GET
link, not a form action -- switching display currency charges nothing, so
it needs no CSRF token), category/product/search pages showing converted
prices via `convertDisplayPrice` (presentation only; the hidden
`unit_price_cents` sent to the cart is always the base currency), a
discount code field at checkout, and a full subtotal/discount/tax/
shipping/total receipt on the order confirmation page.

## Consequences

- Found live, not in review: `commerce.DiscountCode` had no JSON tags, so
  the admin create-discount-code endpoint returned `PascalCase` fields
  (`"ID"`, `"Code"`, ...) instead of this project's `snake_case`
  convention every other response follows. Fixed before this shipped.
- Found live: the checkout transport handler accepted `item_titles`/
  `item_skus` nowhere -- `Service.CheckoutRequest` had carried these
  fields since Phase 2, but nothing wired them from the public HTTP API,
  so every real order's line items had a blank title and SKU. The
  storefront already resolves this data for its own cart display
  (`cartLines.ts`); it now sends it at checkout too
  (`itemCatalogData`). This was a real, pre-existing, Phase-2-era gap that
  only surfaced once something displayed an order's item titles.
- Verified against a real `hydro serve` instance with
  `tax.rate=0.20/inclusive=true`, `shipping.flat_rate_cents=5000/
  free_above_cents=50000`, a `10%` discount code and a `USD`/`EUR`
  currency switcher: the checkout estimate and the real order receipt
  agree with hand-computed KDV-inclusive math at both sides of the
  free-shipping threshold, the discount reduces the taxable base before
  tax, the currency switcher converts a product's listed price without
  touching what is actually charged, and the order receipt's line items
  now show their real title and SKU instead of blank cells.

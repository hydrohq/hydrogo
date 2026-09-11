# ADR-0017: Commerce consistency and payment providers

- **Status:** Accepted
- **Date:** 2026-09-10
- **Related decisions:** [ADR-0006](0006-extensible-database-adapters.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0015](0015-versioned-http-api.md)
- **Implementation status:** Commerce domain/service, SQL store, HTTP checkout, provider-neutral payment contract, webhook replay migration, and Stripe/iyzico/PayTR signature verifiers have been implemented. Orders carry the customer/address data to be sent to the provider as an immutable checkout snapshot; with migration 202609100007, old orders remain readable backwards. iyzico/PayTR payment creation and refund flows are subject to a separate sandbox gate with provider account/API configuration.

## Decision

Commerce is a separate and optional module. When inactive, it does not load routes, migrations, workers, assets, or payment providers.

Order creation performs the following operations in a single DB transaction:

1. Finds a previously created order using the idempotency key.
2. Atomically reserves product stock.
3. Writes the price and currency to the order line as a product snapshot.
4. Creates the order.
5. Appends `order.created` and `email.send_invoice` outbox events.

If the idempotency key is reused by the same user with a different order body, an error is returned. The order transaction does not make network calls to the payment provider.

Money values are stored as minor-unit integers and ISO currency codes. Stock cannot be negative. The domain and application service do not know SQL dialects; the store adapter provides SQLite/PostgreSQL atomic update and locking behaviors.

## Payment

The payment provider adapter contract covers `CreatePaymentIntent`, signature-verified webhook, and `Refund` operations. Stripe intent/refund, iyzico Refund V2, and PayTR Refund API are implemented with real HTTP adapters. Buyer/address fields required for hosted intent are snapshotted during order creation; provider secrets like merchant callback/return URL and deployment settings come from config/env, and are not written to the order. If required data or provider configuration is missing, the adapter returns a configuration error instead of faking success. The webhook event ID and provider idempotency key are durably recorded. An unknown or invalidly signed event is rejected; the payment status only advances via a verified provider event.

The first commerce delivery can be tested without a payment provider. Stripe, iyzico, and PayTR are added with separate adapters and sandbox contract tests; provider SDK details do not leak into the domain.

The HTTP contract is versioned in [commerce-v1.yaml](../api/commerce-v1.yaml). The `/api` compatibility alias is provided by the router; new clients use `/api/v1`.

The cart uses a single active record per user and a single line per product. Adding to the cart snapshots the product price within a transaction; current product/price and stock are re-verified during checkout.

Checkout is preferably done via `cart_id`. The old direct `items` format is supported for transition period compatibility; both formats use the same idempotency and stock transaction.

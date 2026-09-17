# R26 — Stripe payment-intent idempotency boundary (2026-09-17)

The payment-intent route accepts an optional bounded `Idempotency-Key`. When
the selected provider implements the optional `IdempotentPaymentProvider`
contract, the key is forwarded to that provider. Stripe now implements this
contract and sends the key as the provider `Idempotency-Key` header. Existing
providers and callers without a key retain the previous behavior.

Verification: `go test ./modules/commerce/...` — PASS, including transport
tests for forwarding/rejecting keys and a Stripe adapter test that asserts the
outgoing header. This is provider-level at-most-one intent creation only when
Stripe honors the key; HydroGo does not claim exactly-once behavior for
iyzico/PayTR or arbitrary external providers.

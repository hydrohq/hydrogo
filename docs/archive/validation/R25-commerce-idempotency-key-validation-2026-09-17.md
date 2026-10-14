# R25 — Commerce idempotency-key validation (2026-09-17)

Commerce checkout now uses the same bounded key shape as the form idempotency
pilot: 16–128 visible ASCII characters, with no whitespace or control bytes.
Both the HTML checkout and JSON checkout route validate the key before business
logic runs, and the domain `Checkout.Validate` method enforces the same rule for
non-HTTP callers. Invalid keys fail with a stable client error and cannot reach
stock reservation or payment code.

Verification: `go test ./modules/commerce/...` — PASS. This change is
validation-only: it adds no tables or migrations, preserves the existing
SQLite/PostgreSQL `(user_id, idempotency_key)` order uniqueness contract, and
does not change payment-provider retry semantics. Provider calls remain
at-least-once until a provider-specific idempotency contract is enabled.

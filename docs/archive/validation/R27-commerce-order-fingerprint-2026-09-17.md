# R27 — Commerce order request fingerprint (2026-09-17)

Commerce checkout now stores a SHA-256 fingerprint of the logical request
(actor, cart or explicit items, and customer snapshot) in
`commerce_orders.idempotency_hash`. A retry with the same actor/key and the
same fingerprint replays the existing order. Reusing the key with a different
request returns `ErrIdempotencyConflict` before stock reservation, cart
mutation, outbox enqueue, or payment work.

Migration `202609170002_commerce_order_idempotency_hash` adds the bounded,
non-secret column on SQLite and PostgreSQL. Legacy rows remain readable with
an empty fingerprint; they retain key-only replay semantics until naturally
replaced or backfilled. The repository and service tests cover the new column
and changed-request conflict behavior.

Verification: `go test ./migrations ./modules/commerce/... ./tests/contract -count=1` — PASS.

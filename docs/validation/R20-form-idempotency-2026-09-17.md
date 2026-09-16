# R20 — Form idempotency pilot (2026-09-17)

The public site form route now supports bounded idempotency keys. Matching
retries replay `202 Accepted`; a different request fingerprint returns
`409 Conflict`. Only the key and SHA-256 fingerprint are stored. Empty legacy
keys do not collide, and SQLite/PostgreSQL share the same partial unique index.

Verification: `go test ./modules/site/... -count=1` — PASS. The repository
dialect contract covers SQLite and PostgreSQL when `HYDRO_TEST_POSTGRES_URL`
is configured. A uniqueness race is looked up only after the failed
transaction, preserving PostgreSQL transaction semantics.

This is a route-level pilot, not yet the universal HTTP mutation table in
ADR-0136. Further routes require replay metrics, retention review and explicit
response semantics.

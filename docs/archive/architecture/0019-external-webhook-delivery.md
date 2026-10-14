# ADR-0019: Signed external webhook delivery

- **Status:** Accepted
- **Date:** 2026-09-10
- **Related decisions:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0017](0017-commerce-consistency-and-payments.md)

Event delivery to external systems is performed by a separate dispatcher contract used by the outbox consumer that runs after the order transaction. The dispatcher enforces a topic allowlist, signs the body with HMAC-SHA256, sends the event ID as the `Idempotency-Key`, and uses an HTTP timeout.

Delivery is bounded by retry/backoff; external endpoint failure is left to the worker policy which removes the outbox event from its lease and schedules it for a later time. The dispatcher does not keep a permanent record and does not replace the outbox lease. The endpoint URL can only be HTTP/HTTPS; secrets come from the environment configuration and the response body is read up to a limit. `hydro serve` starts this worker if an endpoint is configured; if not configured, it remains disabled without consuming resources.

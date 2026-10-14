# ADR-0217: Consent-Gated, Bounded Content View Analytics

- **Status:** Accepted; unique visitor provision partially superseded by [ADR-0269](0269-optional-no-cookie-visitor-analytics.md)
- **Date:** 2026-09-22
- **Decision:** Content view telemetry is stored only as daily aggregates keyed
  by tenant, content type, content ID, locale and consent purpose version. The
  request path may enqueue a bounded job, but never waits on analytics storage.
  Events without a current granted consent are rejected by the store.

## Privacy and capacity boundary

- No IP, account ID, cookie, fingerprint, full URL or advertising ID is
  accepted by the contract.
- The queue has explicit capacity, worker count, dropped and failed counters.
- Aggregates are operator-retained and purged by an explicit retention action.
- Optional daily grouped unique visitor analytics is governed by ADR-0269;
  this ADR continues to govern consent-based content view events only.
- Tenant is a required dimension and must be bound by the Edition composition
  root before a report is exposed.

# ADR-0269: Optional No-Cookie Daily Visitor Analytics

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related:** [ADR-0125](0125-versioned-analytics-consent-contract.md), [ADR-0131](0131-durable-analytics-consent-store.md), [ADR-0217](0217-consent-gated-content-view-analytics.md), [ADR-0268](0268-seo-performance-and-discovery-runtime.md)

## Context

Issue #226 requests cookie-free daily visitor counts, referrers, pages, and
device summaries. A rotating hash of network address and user-agent is still
pseudonymous data; it is not anonymous merely because the source values are
hashed. Privacy rules and analytics consent exemptions vary by jurisdiction.
This decision adds a narrowly scoped optional capability without changing the
existing explicit-consent event contract.

## Decision

- No-cookie analytics is disabled by default. It runs only when the Analytics
  module is active and `analytics.no_cookie_enabled` is true.
- Enabling it requires an operator-managed `analytics.hash_key` of at least 32
  bytes, shared by instances that serve the same database. The key is secret
  configuration and must not be logged or committed.
- Successful public GET requests are observed after serving the response. DNT
  and Global Privacy Control opt-outs are honored. The middleware does not
  read or set cookies and does not persist raw IP addresses or user-agent text.
- The server reduces IPv4 addresses to /24 and IPv6 addresses to /56, then
  HMACs that network prefix, user-agent, tenant, and UTC day. It queues a
  bounded event. The database transaction inserts the daily key and updates
  page/referrer/device/browser/OS aggregates atomically; duplicate daily keys
  increment the unique count once, including across process restarts.
- Daily HMAC keys are pseudonymous identifiers and are retained only for
  deduplication. They are purged after the visitor day closes (at most about
  48 hours); grouped statistics are retained for 90 days. The feature must be
  disabled if this lifecycle cannot be maintained.
- Reports are tenant scoped and expose grouped dimensions only. The existing
  consent endpoints and consent-based reports remain governed by ADR-0125,
  ADR-0131, and ADR-0217.
- Enabling this feature does not itself establish a lawful basis, guarantee a
  consent exemption, or satisfy an operator's notice obligations. The Edition
  operator must determine local requirements and provide appropriate notice
  and controls. See the [KVKK cookies guide](https://www.kvkk.gov.tr/Icerik/7353/Cerez-Uygulamalari-Hakkinda-Rehber),
  [CNIL analytics guidance](https://www.cnil.fr/en/sheet-ndeg16-use-analytics-your-websites-and-applications),
  and [EDPB personal-data guidance](https://www.edpb.europa.eu/sme/be-compliant/secure-personal-data_en).

## Data and deployment compatibility

The optional Analytics module owns an additive SQLite/PostgreSQL migration for
daily grouped metrics and the temporary deduplication table. PostgreSQL applies
tenant row-level security to both tables. Deploy the migration before enabling
the feature. SQLite and PostgreSQL contract tests cover deduplication, tenant
dimensions, aggregate increments, and expiry. Rollback retains the tables and
data; dropping them loses reports and active deduplication keys.

The queue is bounded and adds one worker only while Analytics and this option
are enabled. A deployment must configure the same hash key on each instance;
rotation changes deduplication identity for the remainder of the current day,
so rotate at a UTC day boundary. The compact profile has no added worker when
the module or option is disabled. Hash-table size is bounded by incoming
requests and cleanup retains no more than two day partitions.

## ADR-0217 amendment

ADR-0217 remains in force for consent-gated content view events. Its statement
that unique visitor analytics is a separate consented capability is partially
superseded by this ADR only for the optional daily grouped capability above.
It does not authorize identifying, cross-day, or account-linked visitor
tracking.

# ADR-0022: Consent-based product analytics and manageable legal content

- **Status:** Proposed
- **Date:** 2026-09-10
- **Supersedes ADR:** None
- **Replaced by ADR:** None

## Context

Admin Studio is requested to offer reports for visits, session duration, country, and content interaction. Even if this data is not used to directly identify a user, it can carry personal data risk in combination with browser behavior and online identifiers. Reporting should not become a condition for user access; mandatory technical cookies and analytics consent should not be presented as the same thing.

The data controller must verify the identity, purpose, legal basis, transfer, retention, and application information in the privacy notice according to their own activities. Core should not fabricate this information or produce a text that guarantees legal compliance.

## Decision

- Analytics is a separate and optional `analytics` module. If inactive, it does not load migrations, routes, client assets, or workers.
- The module is off by default. Even when enabled, an analytical event is not accepted unless the visitor explicitly gives analytics consent.
- The consent record is kept with the purpose version, preference time, and revocation time; it is separate from the mandatory session/CSRF cookie. No new analytical event is written after rejection or revocation.
- The first version only produces daily aggregates: consented visit, total session duration, page view, and country code. IP address, full URL query string, user account, persistent device fingerprint, and advertising ID are not stored.
- The frontend connects to Core only via explicit contract consent and event endpoints. For a separate frontend deployment, the client SDK/adapter explicitly implements the consent flow; Core session cookies are not shared for tracking purposes.
- The Dashboard and Reports only show real metrics when the analytics module is active and there are enough consented aggregates; otherwise, it explicitly shows a "no data / analytics off" state. Demo numbers are not fabricated.
- Legal texts (privacy policy, cookie policy, privacy notice, and analytics consent text) are managed as versioned Pages records. Admin Studio shows the missing mandatory fields and publication status of the texts; it does not replace legal approval.
- The consent request is presented separately from the privacy notice; rejecting it does not block access to the service. Every new analytics purpose/text version requires a new preference.

## Alternatives

- **Opt-out by default, anonymous third-party analytics:** Blurs the consent boundary, weakens data control, and contradicts the self-hosted product goal.
- **Producing reports purely from server HTTP logs:** Carries the risk of turning into visitor behavior measurement without consent; conflates operational purpose with product analytics purpose.
- **Only legal text links:** Provides text publication but does not provide an auditable link between consent status, version, and report access.

## Consequences

### Gains

- Product metrics are produced at the boundary of explicit consent and data minimization.
- The Dashboard and Reports honestly show when there is no real data.
- Legal texts are managed with versions by the content team.

### Costs and risks

- Frontend starters need to integrate the consent banner and event contract.
- A trusted edge/header contract is required for country resolution; raw IP is not stored and if the header is not trusted, the country becomes "unknown".
- Text templates do not replace legal counsel.

## Scale impact

- **Compact:** Daily aggregate upserts are kept bounded; the raw event stream is not stored and the module allocates no resources if closed.
- **Standard:** If necessary, reports are refreshed with a bounded, durable aggregate job.
- **Distributed:** Consent/event idempotency key, trusted proxy country resolution, and aggregate partition/retention contract are certified.

## Database impact

- **SQLite:** Consent record and daily aggregate tables are written with short transactions; migrations are applied only if the analytics module is active.
- **PostgreSQL:** The same repository contract and day-based unique aggregate upsert behavior are verified.
- **Future adapters:** Must pass the upsert/retention contract test with date, country, and measurement dimension unique constraints.

## Cache impact

Report responses can be cached with a short TTL; when a new aggregate or consent purpose version changes, the relevant dashboard/report keys are invalidated. In case of cache failure, bounded aggregates are read from the database; an individual's consent record does not enter the shared cache.

## Deployment and compatibility

Migration is expand-only. The old binary does not use the analytics tables; the new binary does not open route/asset/workers if the module is closed. When the purpose/text version changes, the old consent is not considered valid for new analytical events.

## Validation

- Without consent, after rejection, and after revocation, the event endpoint writes no records.
- The aggregate increments only after consent; it is verified in tests that IP/query/user ID are not stored.
- SQLite and PostgreSQL pass aggregate upsert/retention contract tests.
- The Dashboard and Reports correctly explain the off/empty/active states.
- If the legal text area is unpublished or missing, the consent banner integration does not initiate analytics.

## References

- [KVKK — Guide on Cookie Applications](https://www.kvkk.gov.tr/Icerik/7353/Cerez-Uygulamalari-Hakkinda-Rehber)
- [KVKK — Obligation to Inform](https://www.kvkk.gov.tr/Icerik/2033/Aydinlatma-Yukumlulugu-)
- [KVKK — Procedures and principles of the obligation to inform](https://www.kvkk.gov.tr/Icerik/4132/aydinlatma-yukumlulugunun-yerine-getirilmesinde-uyulacak-usul-ve-esaslar-hakkinda-teblig)
- [ADR-0021: Headless browser session boundary](0021-headless-browser-session-boundary.md)

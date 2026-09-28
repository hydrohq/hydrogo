# ADR-0286: Bounded Dalga 4 platform capabilities

- **Status:** Accepted for the bounded implementations in this change
- **Date:** 2026-09-28
- **Related decisions:** ADR-0013, ADR-0014, ADR-0020, ADR-0033, ADR-0067, ADR-0076, ADR-0096, ADR-0128, ADR-0130, ADR-0133, ADR-0134, ADR-0142, ADR-0189

## Decision

- Realtime subscriptions are an optional `realtime` module. The in-process hub has bounded subscribers and queues, drops slow consumers, sends sanitized page/news identity and revision only, and checks tenant and role scope. Delivery is best effort after the application transaction commits. It is not durable and does not claim cross-instance fan-out certification.
- Standalone ACME TLS is opt-in and binds ports 80 and 443. Only the configured hostname is accepted; certificate cache lives under persistent data. It refuses a configured reverse-proxy topology. Deployments behind a proxy use their proxy's TLS termination.
- Scheduled cloud backups are disabled by default. When enabled, the runtime uses the configured S3-compatible media credentials, creates the existing verified local DB/media snapshot, streams a gzip archive with S3 server-side encryption, then prunes expired `.tar.gz` objects within the configured prefix. The schedule is UTC and the retention minimum is one day. Distributed scheduler coordination and restore-from-cloud certification remain deployment gates.
- Draft preview links are HMAC-signed, expire within 24 hours, bind page, tenant and current draft revision, and return private no-store output without editor controls. Saving a later draft revision invalidates the old link. Key rotation revokes all outstanding links. The key is operator-provided and never generated or persisted by the application.
- Search suggestions query only published page/news projections, cap results at five, and are enabled only with the optional `search` capability. The typed UI reuses the existing HTMX attribute contract.
- Session listing and “revoke other sessions” operate only on the authenticated admin user's DB-backed sessions; token hashes are never part of the response.
- Webhook tests send one signed request directly, bypassing the durable outbox and retry worker. Test email goes only to the active administrator's account address and uses the existing HydroGo email component.
- Countdown blocks accept an RFC3339 target and render a typed expired state. Working-hours evaluation is a bounded form contract helper; Client Editions own the submission/auto-response workflow as required by ADR-0134.
- Demo cleanup requires the explicit `clean --demo` target and refuses production unless the operator additionally supplies `--force-prod`.
- The visual schema-builder issue does not enable runtime DDL or dynamic Core routes. Admin Studio and generators may produce reviewable proposals; applying schema changes still uses generated, dialect-certified migrations and versioned module contracts. This follows ADR-0020 and the repository blueprint rules.

## Limits

The realtime hub is process-local; S3 scheduled backup is not a multi-instance lease; ACME has no proxy autodetection; preview-link revocation is revision-bound or by signing-key rotation; push-provider delivery and unused-media destructive cleanup are not enabled by this decision. These require separate persistence, provider, authorization and deployment contracts before production claims.

## Verification

Focused tests cover SSE tenant/topic isolation, bounded S3 archive creation and symlink rejection, signed preview token expiry/tampering, search result tenant/publication filters, webhook test dispatch, test-mail authorization, session listing, working-hours evaluation, countdown validation/rendering and demo-clean safeguards. Full repository verification remains the release gate for this change.

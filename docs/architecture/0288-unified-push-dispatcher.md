# ADR-0288: Optional unified push dispatcher

Status: Accepted

## Context

The platform needs Web Push and FCM delivery without requiring a background worker when push is inactive. Device registrations are privacy-sensitive and broadcasts require bounded work, tenant isolation and explicit provider configuration.

## Decision

- The opt-in `push` module owns a static, versioned subscription table. Inactive push creates no routes or service.
- Registrations are keyed by tenant, platform, authenticated owner and endpoint; web subscriptions preserve their browser keys. Device endpoints can be removed by their owner and dead provider registrations are disabled.
- Broadcasts are synchronous, capped to 10,000 active subscriptions per provider query, and report provider acceptance/failure counts. Provider acceptance is not proof a person saw a notification.
- VAPID and Firebase credentials come from environment configuration and are never stored in the database. Mobile app access uses the member bearer-token boundary.
- FCM HTTP v1 authentication uses service-account JWTs; Web Push uses the Go VAPID implementation.

## Compatibility and operations

The migration has SQLite, PostgreSQL and MySQL variants. No collection-defined schema, runtime DDL, mandatory worker, or inactive-module route is introduced. Configure `HYDRO_PUSH_VAPID_PUBLIC_KEY`, `HYDRO_PUSH_VAPID_PRIVATE_KEY`, `HYDRO_PUSH_VAPID_SUBJECT`, and optionally `HYDRO_PUSH_FCM_CREDENTIALS` (path to a service-account JSON file). Provider failures leave subscriptions available for retry except permanently gone web endpoints.

## Security and data lifecycle

Subscription create/delete requires the owning authenticated session or member bearer token. Broadcasts require an active admin. CSRF protections apply to browser mutations. Endpoint URLs are sent only to the relevant provider and are not included in responses or logs. Disable the module only after export/deletion according to the deployment retention policy.

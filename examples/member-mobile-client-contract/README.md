# HydroGo consumer mobile client fixture

This platform-neutral fixture defines the minimum iOS/Android behavior for a
consumer news-reader or storefront client. It is intentionally separate from
the staff mobile fixture and never uses browser cookies or staff credentials.

Keep access and refresh tokens only in platform secure storage. Replace the
pair atomically after refresh; on a rejected refresh, revoke response, or
secure-storage failure, clear local credentials and require sign-in again.

The wire contracts are [`member-mobile-v1.openapi.json`](../../docs/api/member-mobile-v1.openapi.json)
and [`member-commerce-v1.openapi.json`](../../docs/api/member-commerce-v1.openapi.json).

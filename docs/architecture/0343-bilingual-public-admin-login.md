# ADR-0343: Bilingual public Admin Studio login

- **Status:** Accepted
- **Date:** 2026-10-02
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** ADR-0197, ADR-0308, ADR-0313, ADR-0342

## Context

The public staff login mixed Turkish and English, while the adjacent developer
documentation now defaults to English and supports Turkish. The login must
give staff a coherent language choice before account preferences are available,
without changing the authentication and security contracts.

## Decision

1. The existing `GET /login` accepts `?lang=en` and `?lang=tr`. English is the
   default for a missing or unsupported value. `Content-Language` and the HTML
   `lang` attribute reflect the selected language. Language links preserve a
   validated `return_to` path.
2. All visible login copy, including the hero, labels, theme control, MFA
   challenge, countdown, success and failure notices, is provided in the
   selected language. Messages remain safe and do not reveal raw transport or
   internal error details. This public, pre-authentication choice is separate
   from the signed-in Account locale preference defined by ADR-0197.
3. The login hero follows the blue technical-site palette and provides local
   links to documentation and home. The existing Site Appearance background
   and overlay contract remains effective, and the form stays opaque and
   readable. Compact and narrow layouts retain the full sign-in flow.
4. Login POST, CSRF, MFA, sessions, security summary, and safe redirect
   behavior remain unchanged. No language cookie or new frontend runtime is
   introduced.

## Alternatives

- Use the signed-in Account locale: no preference can be read reliably before
  authentication, and it would couple a public page to a protected setting.
- Add a translation service or browser framework: unnecessary runtime and
  network cost for two bounded, versioned languages.

## Consequences

### Benefits

- Staff can use one language throughout the login and MFA flow.
- The login and documentation have a consistent visual identity.

### Costs and risks

- English and Turkish copy must be maintained together when login states
  change. Render tests cover the two versions and safe return-path links.

## Scale impact

- Compact: embedded markup and CSS only; no worker, service, asset request, or
  connection pool change.
- Standard: identical request-local rendering.
- Distributed: no shared language state or synchronization.

## Database impact

- SQLite: no queries, schema, or migration change.
- PostgreSQL: no queries, schema, or migration change.
- Future adapters: no database contract dependency.

Certified database behavior tests and migration metadata are not applicable
because this decision does not read or write a database. The existing login
handler's appearance lookup and authentication behavior are unchanged.

## Cache impact

The login response is request-local and carries a CSP nonce. It has no
dedicated cache key, TTL, or invalidation path. No stampede or cache failure
fallback is introduced.

## Deployment and compatibility

Old and new binaries can coexist during Blue/Green deployment. The new binary
understands the optional `lang` query; old binaries ignore it and continue to
serve the same route. No session, media, job, migration, or asset continuity is
affected. Rollback is a binary switch.

## Validation

- Render tests verify English and Turkish markup, MFA copy, safe messages,
  appearance behavior, and preserved return paths.
- Browser checks verify both languages, desktop and narrow layouts, and no
  horizontal overflow.
- Existing moduleguard, migrationguard, and importguard CI gates remain
  mandatory.

## References

- [Admin UI locale preference](0197-admin-ui-locale-preference-and-local-catalog.md)
- [Configurable login background](0313-login-background-appearance.md)
- [Bilingual documentation](0342-bilingual-tailwind-documentation-surface.md)

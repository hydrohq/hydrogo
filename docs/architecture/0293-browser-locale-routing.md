# ADR-0293: Browser locale routing

- Status: Accepted
- Date: 2026-09-29

## Context

The public root needs a predictable way to select a configured locale without
changing locale behavior for unrelated routes or making a Client Edition's
content policy implicit.

## Decision

When I18N is enabled, root requests may select a locale from the explicit
`?locale=` choice, the `hydro_locale` preference cookie, then the
`Accept-Language` header, in that order. Only configured locales are valid.
Browser preferences resolve by descending quality and may fall back from a
regional tag to its primary language. The configured default is used when no
header preference matches. A non-default selection redirects to `/{locale}/`;
an explicit choice also stores a one-year HttpOnly, SameSite=Lax cookie whose
Secure flag follows the site's secure-cookie setting. Redirects are private
and vary on Cookie and Accept-Language.

The locale root renders the existing small runtime landing surface and its
locale-specific demo link. Product-specific localized home content and
translation remain Edition responsibilities. A typed Design System language
picker submits an explicit locale choice to the root. No detection or redirect
occurs when I18N is disabled.

## Consequences

Locale routing adds no dependency, database state, or worker. The resolver is
bounded by the configured locale list and request header. Deployments should
keep the active locale list consistent across Blue/Green instances so a
preference cookie remains valid. Rollback consists of removing the root
resolver and locale root route; no data migration is needed.

SQLite and PostgreSQL behavior is identical because the feature does not use
the database. Contract tests cover quality ordering, regional fallback,
unsupported values, explicit selection, and cookie precedence.

# ADR-0280: Tenant-scoped static UI translation overrides

- **Status:** Accepted
- **Date:** 2026-09-28
- **Supersedes:** The catalog immutability and editor-boundary parts of
  [ADR-0197](0197-admin-ui-locale-preference-and-local-catalog.md); embedded
  catalogs and the Turkish/English fallback chain remain authoritative.
- **Related:** [ADR-0172](0172-tenant-scoped-site-data.md),
  [ADR-0276](0276-site-appearance-and-contrast.md)

## Context

Site editors need to adjust visible Admin Studio labels without editing source
catalog files. Core must remain offline-capable, preserve its embedded
translations, and prevent one Client Edition from changing another tenant's
labels.

## Decision

The Site module stores overrides in its existing tenant-scoped
`site_settings` table under `i18n.override.<locale>.<catalog-key>`. Only keys
present in the embedded Turkish and English catalogs can be changed. Empty
values clear an override. Lookup prefers the tenant override, then the
embedded requested locale, then the configured primary catalog. No external
translation service or new runtime dependency is introduced.

At startup, enabled Site tenants' public settings are loaded into a bounded
in-memory map. Successful editor changes update the database and that tenant's
map in one request, so subsequent renders in the process use the new value.
The Settings table remains the durable source of truth after restart. A
multi-process deployment must route edits and renders through a shared release
or restart peers to refresh their in-memory catalogs; Core does not add a
polling worker or distributed cache dependency.

## Compatibility and operations

- No migration is required. SQLite and PostgreSQL use the existing Settings
  upsert and tenant filter contracts.
- Old binaries ignore the namespaced rows. Rollback leaves those rows intact;
  restoring a newer binary restores their use.
- The inactive Site module performs no catalog query or cache allocation.
- Each tenant cache is bounded by the embedded catalog key set and each value
  is limited to 1,000 bytes.
- The Admin translation screen exposes only the embedded key set and supported
  locales; missing locale entries display the Turkish fallback and are marked
  for editors.

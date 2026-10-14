# ADR-0313: Configurable admin login background

## Status

Accepted

## Decision

The Site Appearance settings may specify a same-origin media background for
the admin login hero and a bounded dark overlay opacity. The appearance
settings page reuses the existing media picker and persists these values in
the existing site settings store. The login handler reads public appearance
settings when Site is enabled and falls back to the neutral gradient when
settings cannot be read or no image is configured. The login form pane keeps
its existing opaque, high-contrast surface.

URLs are restricted to validated same-origin media/theme paths and overlay
opacity is constrained to 0–0.8 before CSS generation. No schema migration,
worker, route, cache, dependency, or additional service is introduced. The
existing setting store retains its SQLite/PostgreSQL portability contract.

## Verification

Appearance validation and persistence tests cover the new settings. Login
rendering tests cover configured and rejected external backgrounds. The full
uncached Go suite and migration guard are required before release.

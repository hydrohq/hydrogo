# ADR-0197: Admin UI locale preference and local catalog

- **Status:** Accepted (catalog and preference foundation; full surface translation open)
- **Date:** 2026-09-20
- **Related decisions:** [ADR-0032](0032-admin-locale-selector.md), [ADR-0108](0108-locale-fallback-chain.md), [ADR-0025-locale](0025-locale-presentation-formatting.md)

## Context

HydroGo already supports locale-aware content and presentation formatting, but
Admin Studio contains product labels and messages embedded in Go/Templ output.
Administrators need to choose Turkish or English from their Account area
without introducing a runtime translation service, network dependency, or
non-deterministic AI output.

## Decision

Admin UI translations use small, embedded, version-controlled JSON catalogs.
The first catalog set is Turkish (`tr`) and English (`en`). Translation lookup
uses the bounded fallback chain and returns the key when a catalog entry is
missing. Domain content locale and Admin UI locale remain separate concepts.

The Admin Account surface is the canonical place for the user's language
selector. The selected value is bounded to the configured/catalog allowlist,
submitted with the existing CSRF contract, and becomes the locale passed to
Admin renderers. No translation model, external service, or runtime download
is permitted.

## Consequences

- Compact deployments add only embedded catalog bytes and no worker/service.
- New UI text must use a stable translation key rather than adding another
  hard-coded Turkish or English phrase.
- Full surface translation and browser visual QA are staged work; missing keys
  remain visible and do not fail closed into blank UI.

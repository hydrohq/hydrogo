# ADR-0083: Locale-derived document direction

## Status

Accepted

## Context

Public content can be addressed with locales beyond Turkish and English. A
document that always renders left-to-right makes Arabic, Persian, Hebrew, and
similar content harder to read and can break layout assumptions.

## Decision

- The canonical layer derives only the primary locale subtag and returns a
  bounded `ltr` or `rtl` value.
- Public page and news document roots emit that value as `dir`.
- Unknown locale tags default to `ltr`; user content never controls markup or
  CSS direction directly.

## Consequences

RTL-aware themes can use the document direction without duplicating locale
parsing. Full translation, bidirectional text review, and visual device QA
remain product/theme responsibilities.

## Verification

See `internal/canonical/canonical_test.go`, public transport tests, and
`docs/validation/R5-locale-direction.md`.

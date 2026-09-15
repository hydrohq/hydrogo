# ADR-0025: Locale-aware presentation formatting

- Status: Accepted
- Date: 2026-09-15

## Context

HydroGo stores domain dates and monetary values in locale-neutral forms, but
the generated/admin/public surfaces need consistent formatting for different
audiences and directions.

## Decision

`internal/i18n` provides bounded render-time helpers for date, number, and
ISO-4217 currency presentation. Invalid locale tags fall back to English;
invalid currency codes fail closed. Domain and persistence packages do not
depend on locale formatting.

## Consequences

Modules can present values consistently without duplicating formatting logic or
leaking locale concerns into Core contracts. Translation catalogs and complete
product locale review remain Client Edition/product responsibilities.

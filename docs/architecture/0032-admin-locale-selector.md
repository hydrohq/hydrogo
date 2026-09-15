# ADR-0032: Provider-neutral admin locale selector

## Status

Accepted

## Decision

Admin forms may receive a bounded locale list and current locale from the
application layer. The renderer emits an escaped selector; locale discovery,
fallback, translation storage, and permissions remain i18n/module concerns.

## Compatibility and capacity

This is additive and requires no schema migration. Locale values are bounded
to 16 characters at render time, behaves identically on SQLite/PostgreSQL, and
adds no persistent or background resource consumption.

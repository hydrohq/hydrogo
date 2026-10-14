# ADR-0029: Request-scoped relation field rendering

## Status

Accepted

## Decision

Admin metadata may declare `relation` fields, while the handler supplies
request-scoped `RelationOptions` (`id` and display label). The renderer emits a
safe select control and never performs database queries or exposes provider
details. Empty or stale choices fail closed at the application-service
validation boundary.

## Compatibility and capacity

The contract is SQLite/PostgreSQL neutral and adds no migration. Choices are
bounded by the caller's query policy; pagination/search remain adapter
concerns. Rendering is linear in supplied options and adds no worker or cache
overhead.

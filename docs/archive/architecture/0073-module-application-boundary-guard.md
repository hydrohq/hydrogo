# ADR-0073: Guard module application packages from concrete adapters

## Status

Accepted

## Decision

Continuously scan non-test module application/domain packages and reject direct
imports of SQL drivers, concrete SQL stores, Redis adapters, and durable queue
stores. Repository, transport, provider, and adapter packages remain the
allowed composition boundary.

## Consequences

New features must depend on stable interfaces instead of infrastructure
details, preserving replaceability and testability. Adapter tests remain free
to use real drivers, and the guard does not claim to replace code review or
runtime integration tests.

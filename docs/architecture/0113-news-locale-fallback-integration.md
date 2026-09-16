# ADR-0113: Configured locale fallback for public news

- **Status:** Accepted
- **Date:** 2026-09-16

## Context

Public news routes queried the repository directly, so the shared locale
fallback contract could not be used consistently with Pages.

## Decision

`news.Service.GetPublishedByRoute` owns the opt-in exact → primary → default
locale chain. Both JSON and HTML public routes call this service method. The
service retries only `sql.ErrNoRows`; all other errors remain visible to the
transport for correct status handling. Empty locale configuration keeps the
existing exact lookup behavior.

## Consequences

News and Pages now share the same application-layer fallback semantics while
keeping repository and SQL details behind module boundaries. Multilingual
edition acceptance and production translation fixtures remain required.

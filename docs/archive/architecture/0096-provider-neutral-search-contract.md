# ADR-0096: Provider-neutral search contract

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Search is useful for news, content, commerce, and portals, but forcing a
specific index or external engine into Core violates the optional-service and
data-sovereignty rules.

## Decision

Expose a bounded `internal/search` contract. Modules provide a `Provider`,
while Core-owned query validation limits input and pagination. Results are
stably ordered by score and ID for deterministic clients. Indexing, tokenizing,
ranking, persistence, and external engine selection remain adapter/module work.

## Consequences

Compact deployments add no search process or table unless selected. Providers
can target SQLite, PostgreSQL, or an external index without changing callers.
The contract does not claim relevance quality; a real dataset benchmark is
required before selecting a production provider.

## Verification

`go test ./internal/search -count=1` covers input bounds, locale validation,
deterministic ordering, and response limits.

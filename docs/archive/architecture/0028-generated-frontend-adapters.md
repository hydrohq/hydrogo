# ADR-0028: Optional generated frontend adapters

- Status: Accepted
- Date: 2026-09-15

## Context

Typed resource clients should be usable from React, HTMX and vanilla browser
frontends without forcing a frontend runtime or provider-specific package into
HydroGo Core or the Compact profile.

## Decision

The resource generator emits small `react.ts`, `htmx.ts` and `vanilla.ts`
adapter contracts beside the typed client. They are dependency-light examples:
React remains a peer dependency, HTMX attributes are plain data, and the
vanilla adapter accepts an injected fetch implementation. Client Editions may
replace them with richer adapters without changing the resource contract.

## Consequences

First-route prototyping needs less glue code while Core remains server-rendered
and provider-neutral. Adapter behavior must remain bounded, typed and covered
by the generated review checklist; no credentials or provider SDKs are emitted.

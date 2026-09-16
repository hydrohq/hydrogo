# ADR-0106: Opt-in bulk selection surface in Admin Studio lists

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Admin Studio had a bounded bulk request contract, but list rendering offered no
standard way for a module to expose selectable rows. Adding selection globally
would impose UI and mutation assumptions on collections that do not need bulk
operations.

## Decision

`Collection.BulkActions` is an optional presentation field. When present,
`RenderList` emits an accessible select-all checkbox, per-row checkboxes, and
escaped action metadata. Rows must provide a non-empty ID; action URLs remain
allowlisted. Collections without bulk actions keep the existing markup and
zero additional behavior.

## Consequences

Modules can progressively adopt bulk workflows while retaining ownership of
CSRF, authorization, transaction, audit, partial failure, and request handling.
The Core does not infer or execute a mutation from the rendered metadata.

## Verification

`go test ./internal/adminstudio ./internal/generator -count=1` covers opt-in
rendering, row-ID enforcement, escaping, and collection validation.

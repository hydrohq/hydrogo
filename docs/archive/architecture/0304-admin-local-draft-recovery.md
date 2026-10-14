# ADR-0304: Admin Local Draft Recovery

- Status: Accepted
- Date: 2026-09-29

## Context

An interrupted browser session can discard unsaved content in the page and
news editors. Recovery must stay in the user's browser and must not store
credentials or server security tokens.

## Decision

The page and news edit forms opt in to a small vanilla JavaScript recovery
helper. It saves named editable controls to local storage after a debounce,
using a key scoped by a one-way digest of the authenticated user identifier
and the edited resource. CSRF, version, password, file, and button controls
are excluded. Stored drafts are size and age bounded; users can restore or
discard a recovered draft. Successful updates redirect with a one-time query
marker so the matching local draft is cleared.

## Compatibility and operations

No schema, route, worker, or database changes are introduced. Storage failures
degrade to the regular server save flow. The helper runs only on opted-in
edit forms in authenticated Studio pages. Local storage is browser-local and
is not synchronized across devices. Content can remain in the browser until
the user saves or discards it, with stale entries removed after 30 days.

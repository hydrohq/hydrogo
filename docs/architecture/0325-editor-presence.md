# ADR-0325: Tenant-Scoped Editor Presence

Status: Accepted

## Context

Editors need a timely warning when another editor is on the same page or news
item. The content update handlers already use optimistic version checks, which
remain the final protection against overwriting a newer revision.

## Decision

Record editor heartbeats in a bounded process-local map, keyed by tenant,
resource, and user. Heartbeats refresh every 15 seconds and records expire after
60 seconds. Cap the map at 10,000 active editors and prune stale entries while
handling heartbeats and presence reads. Only users with content-write access
can publish or read presence. Responses contain a display name and last-active
age, never user IDs or document contents. Templ renders the warning through a
nonce-bearing script that updates text with DOM text nodes.

The feature adds routes only to active Pages and News modules. Existing
optimistic locking remains authoritative for writes.

## Consequences

No schema migration or background worker is required. Presence is best effort
and process-local; deployments with multiple independent HTTP workers may show
partial presence unless their routing keeps collaborators on the same worker.
The feature carries no durable data and does not change database adapter
contracts.

## Capacity and Recovery

Memory use is bounded by the active-entry cap and idle TTL. A process restart
clears transient presence without affecting stored content or sessions.

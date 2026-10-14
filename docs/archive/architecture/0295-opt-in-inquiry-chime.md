# ADR-0295: Opt-in form inquiry chime

- Status: Accepted
- Date: 2026-09-29

## Context

Admin staff may want an audible indication of new inquiries, while shared or
quiet workspaces need sound disabled unless a user explicitly opts in.

## Decision

When both Forms and Realtime are active, a newly persisted, non-replayed form
submission publishes the ephemeral tenant-scoped `forms:submission_created`
event. Its payload contains only the form key; it does not include the submitter,
email address, submitted fields, or submission identifier. Existing authorized
admin/editor SSE subscriptions deliver the event.

The Admin Studio account page exposes a typed Design System preference. It is
off by default and stored in that browser's local storage. With the preference
enabled, the Admin shell subscribes to the form event and generates a short
two-tone signal through the browser Web Audio API after user interaction has
unlocked audio. No audio asset or client runtime dependency is added.

## Consequences

No database migration, cache entry, worker, or durable event is added. SQLite
and PostgreSQL behavior is the same. Existing bounded SSE subscriber buffers
and reconnection behavior govern backpressure; a slow subscriber is dropped
and may reconnect. The setting is per browser. During Blue/Green deployments,
subscriptions are process-local and reconnect to the active instance. Tests
cover single publication for idempotent submissions, event payload privacy,
the default-off accessible control, and client subscription markers.

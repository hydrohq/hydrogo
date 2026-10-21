# 0009: Core-owned event outbox and a migration baseline for new databases

- Status: Proposed
- Date: 2026-10-21

## Context

Webhooks and other event delivery use the `outbox_events` table. Its migrations (`202609100006_outbox_delivery` and later) belong to the `pages` module, and the outbox worker only starts when `pages`, `news` or `commerce` is enabled. A project without those modules (the `crm` preset: core, auth, media, collections) has no outbox, so content-type events (`content.created`, `content.updated`, `content.deleted`) cannot be delivered there. Published migrations may not be edited or removed (rule 3), and 117 migration directories already make a fresh install slow to read and verify.

Making `collections` create the outbox with its own migration would collide with the `pages` migrations on databases that enable both (duplicate table or columns).

## Decision (proposed)

1. Add a **baseline** for new databases: one generated schema snapshot per dialect, recorded as having applied every existing migration, so a fresh install runs the snapshot instead of the chain. Existing databases keep running the chain; both paths must end in the same schema, and a test compares them (`internal/database/schema_diff.go` already exists).
2. Move ownership of the outbox, audit and jobs tables to `core` in the baseline, so any preset can use events without `pages`.
3. Start the outbox worker when any module or content type emits events and a webhook endpoint is configured; emit nothing (and write no rows) when no endpoint exists, so the table cannot grow unread.
4. Add `content.created`, `content.updated` and `content.deleted` to the event catalog (additive, payload version 1, `aggregate_id` = `<type>/<record id>`), written in the same transaction as the record.

## Consequences

- Fresh installs get a short, verifiable schema; upgrades are unchanged.
- Until this lands, content-type events are not implemented; phase A.4 of the roadmap waits for this decision.
- The baseline generator and the equivalence test are the main work; the event emission itself is small.
- Risk: a baseline that drifts from the chain. The equivalence test runs in `tools/check` for all dialects that are available.

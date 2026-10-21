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

## First slice (done)

Core migration `202610210002_core_audit_events` creates `audit_events` in its final shape (columns, indexes, immutability triggers and, on PostgreSQL, the tenant policy) when it does not exist, so a project without `pages` (the `crm` preset) can sign in and keep an audit trail. Every statement is a no-op where the pages migrations already built the table, so existing databases are unchanged. The released `pages` migrations were not touched: moving the first one to `core` would also create an empty `pages` table without `tenant_id`, which breaks the PostgreSQL tenant policy migration.

Known limit: enabling `pages` later on a database that was created without it fails at the first pages migration (the audit table already exists). The baseline below removes that. Until then the migration runner detects this before running any SQL and stops with `ErrPagesNeedsFreshDatabase` (so MySQL, which has no transactional DDL, is never left half-migrated); the way out is to create the project with `pages` from the start.

## Second slice (done)

Core migration `202610210004_core_outbox_events` creates `outbox_events` in its final shape (with the PostgreSQL tenant policy) when it does not exist, the same way as the audit table. Content events (`content.created`, `content.updated`, `content.deleted`, payload version 1, `aggregate_id` = `<type>/<id>`) are written in the record's transaction by the collections store, only while a configured webhook subscribes to one, and the outbox worker starts for `collections` in that case. The baseline (decision 1) is still open; the known limit above now also covers `outbox_events`.

## Consequences

- Fresh installs get a short, verifiable schema; upgrades are unchanged.
- The baseline generator and the equivalence test are the main work; the event emission itself is small.
- Risk: a baseline that drifts from the chain. The equivalence test runs in `tools/check` for all dialects that are available.

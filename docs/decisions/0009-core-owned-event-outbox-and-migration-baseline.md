# 0009: Core-owned event outbox and a migration baseline for new databases

- Status: Accepted
- Date: 2026-10-21 (baseline accepted and delivered 2026-10-22)

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

Known limit: enabling `pages` later on a database that was created without it fails at the first pages migration (the audit table already exists). The baseline (third slice) removes that. Until the baseline landed the migration runner detected this before running any SQL and stopped with `ErrPagesNeedsFreshDatabase` (so MySQL, which has no transactional DDL, was never left half-migrated).

## Second slice (done)

Core migration `202610210004_core_outbox_events` creates `outbox_events` in its final shape (with the PostgreSQL tenant policy) when it does not exist, the same way as the audit table. Content events (`content.created`, `content.updated`, `content.deleted`, payload version 1, `aggregate_id` = `<type>/<id>`) are written in the record's transaction by the collections store, only while a configured webhook subscribes to one, and the outbox worker starts for `collections` in that case. The known limit above now also covers `outbox_events`; the baseline (third slice) lifts it.

## Third slice (done): the baseline

`migrations/baseline/` holds one generated script per group and dialect, and `tools/baselinegen` regenerates them from the chain. The runner uses them in place of the chain; everything else (ledger, checksums, later migrations) behaves as before.

**Groups.** Migrations are partitioned by the modules they need: the owning module (unless `core`) plus `requires_modules`. `core` holds every migration that always runs, a module group holds that module's migrations (including core migrations that require only that module, such as the audit columns that need `pages`), and a combination group holds migrations that need several modules together. There are 26 groups. `forms` extends the `form_inbox` table that `site` creates without declaring it, so `baselinegen.ImpliedModules` makes the group `combo_forms_site`; the chain's own selection is unchanged.

**What a script is.** The difference between the schema the chain builds for the group's module set and the schemas of the groups below it, plus the seed rows (today one: `job_queue_capacity_lock`). It adds objects and also alters tables a lower group created (`form_inbox` gains columns and swaps an index). The generator refuses anything it cannot render faithfully (a dropped column, a changed type, a changed table constraint) and `-chain-only` can leave such a group in the chain on purpose.

**When it runs.** For each enabled group, in manifest order (core, then smaller module sets), if every migration of the group is still pending and the ledger is group-consistent (every group is either fully applied or fully pending). The script runs in one transaction (PostgreSQL, SQLite; MySQL commits DDL itself, so its scripts are written to be repeatable) and the group's migrations are recorded with the checksums the chain would have written. A database that ran only part of a group, for instance an older release in the middle of an upgrade, runs the chain for everything that is left. A bundle without a script for the dialect falls back to the chain.

**What it fixes.** A database created without a module can enable it later: the pages group no longer collides with the audit and outbox tables core created, for databases the baseline built and for existing ones created by the chain without `pages` (ADR 0009's known limit above). The interim guard (`ErrPagesNeedsFreshDatabase`) remains for chain-only runs. It also closes a gap in the chain: the PostgreSQL tenant RLS migration applies policies only to tables that exist when it runs, so a module enabled afterwards had none; the baseline gives every group its policies.

**Verification.** `internal/database/baselinegen` builds the same module set with the chain alone and with the baseline and compares schema (tables, constraints, indexes, functions, triggers, row security, policies), seed rows and the ledger: all presets, every module alone, every module pair in both orders on SQLite (a sample on PostgreSQL and MySQL), and "core first, module later" starting from a baseline-built and from a chain-built database. It runs on SQLite always and on PostgreSQL and MySQL when `HYDRO_TEST_POSTGRES_URL` / `HYDRO_TEST_MYSQL_URL` are set (certified against PostgreSQL 18 and MySQL 8.4). `migrationguard` checks the manifest against the chain and the script checksums on every run. The schema diff used by `hydro db diff` still builds its reference from the chain alone, so it stays independent of the baseline.

**Rules.**
- The baseline is immutable once released, like a migration. A migration added later is not in it and runs from the chain on top. Replacing the baseline means a new manifest version.
- Column order is not part of equivalence: a baseline puts a module's columns in one place, the chain appends them as migrations ran, and no code addresses columns by position (nothing uses `SELECT *` or `INSERT` without a column list).
- Generating needs a PostgreSQL and a MySQL 8 account that may create databases (see `tools/baselinegen`).

**Capacity and deployment.** Fresh installs are faster (best of three, local servers): SQLite 158 ms to 38 ms for all modules and 31 ms to 15 ms for the `crm` preset; MySQL 8.4 1.3 s to 0.54 s; PostgreSQL 18 197 ms to 162 ms. The binary grows by the embedded scripts (about 265 KB of SQL). Blue/Green is unaffected: both versions read the same ledger and the same schema. Upgrades of existing databases run the chain exactly as before unless a whole group is new to them (a module being enabled), in which case its script runs. No new route, worker, asset or process; nothing runs for a module that is off.

## MySQL 8 finding

Running the chain on a real MySQL 8.4 (until now only MariaDB was available) showed that `202610210001_collection_sequences` failed: `last_value` is a reserved word in MySQL 8. The migration had not been released, so its MySQL script now quotes the column (the checksum in its metadata changed with it) and the sequence queries quote it for MySQL.

## Consequences

- Fresh installs get a short, verifiable schema; upgrades are unchanged.
- The baseline generator and the equivalence test are the main work; the event emission itself is small.
- Risk: a baseline that drifts from the chain. The equivalence test runs in `tools/check` for all dialects that are available.

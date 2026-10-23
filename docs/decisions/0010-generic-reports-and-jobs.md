# 0010: Generic reports and job visibility in core, not per project

- Status: Accepted (Slice A); Proposed (Slices B and C)
- Date: 2026-10-23

## Context

A project that wants pipeline-style reporting (counts/sums/breakdowns over its
own content types) or scheduled/bulk background work has had to hand-build it
per project — `examples/crm-starter`'s dashboard does exactly that, calling
the admin content API directly from Svelte. There is no reusable "declare a
report" or "declare a scheduled job" capability in core, the way content
types already give a project CRUD and an admin screen for free from one YAML
file (`internal/contenttypes` → `collections.Schema()` /
`adminstudio.Collection()` → `SyncSchemas` → `/api/v1/admin/content/<id>` →
`GenericListView.svelte`).

Two gaps block reusing that shape directly:

1. No aggregate query capability exists anywhere. `internal/collections/sqlstore/query.go`
   only filters/sorts/paginates per record. The existing `rollup` field
   (`internal/collections/rollup.go`) is a different, push-based mechanism
   (one incrementally-maintained total on one parent record) and is not a
   pull-based ad-hoc aggregation engine.
2. `internal/jobs` + `internal/jobs/sqlstore` is real, core-owned, durable
   queue infra (lease/ack/nack/dead-letter, ownership established by ADR
   0009) with exactly one consumer (`modules/news/worker.go`) and no admin
   API — only a CLI (`hydro jobs dead-letter list/requeue/purge`) and
   Prometheus metrics. `internal/cron` is a separate, already-generic
   scheduling primitive, used today only to call external HTTP hooks
   (`hooks_schedule.go`), not wired to the jobs queue.

## Decision

Ship this in three slices, each landed as its own PR:

1. **Jobs admin visibility** (this PR): wrap the existing `internal/jobs/sqlstore`
   operator methods in an admin HTTP API and an admin UI tab, so an operator
   can see and manage the job queue for any project without the CLI.
2. **Generic reports engine** (proposed): a `ReportDefinition` declared like a
   content type — one YAML file, no migration — plus the first aggregate
   query capability (`GROUP BY` + `COUNT/SUM/AVG/MIN/MAX`) in
   `internal/collections/sqlstore`, registered into the same admin schema so
   it shows up with zero per-project code.
3. **Declarative bulk jobs** (proposed): content types can declare "run this
   filter + action over my records on a schedule," executed through the
   existing jobs queue (reusing lease/ack/retry/dead-letter) rather than a
   new execution model, scoped to the one safe, generically-interpretable
   case — a bulk field update or an existing hook call, not arbitrary code.

## Slice A (done): Jobs admin visibility

- `internal/jobs/sqlstore.Store` gained `List` (claimable/leased jobs, never
  dead-lettered) and `DeleteDeadLetter` (remove one by id), alongside the
  existing `ListDeadLetters`/`RequeueDeadLetter`/`PurgeDeadLetters`. No schema
  change: the `jobs` table already carries everything needed.
- `cmd/hydro/admin_jobs.go` exposes `GET /api/v1/admin/jobs`,
  `GET /api/v1/admin/jobs/dead-letters`,
  `POST /api/v1/admin/jobs/dead-letters/{id}/requeue`,
  `DELETE /api/v1/admin/jobs/dead-letters/{id}`, gated by the existing
  `admin.operations.manage` permission (the same one `system/services` and
  backups use) — no new auth parsing, per the shared admin middleware rule.
- Always registered in `cmd/hydro/routes_admin.go`, independent of which
  modules are enabled: jobs are core-owned infrastructure, not tied to any
  one module.
- Admin UI: a new "Jobs" section inside the existing System → Workers tab
  (`admin/src/routes/system/tabs/JobsTab.svelte` + `jobs.ts` loader), not a
  new top-level nav item — the Workers tab already shows aggregate queue
  counts; this adds the actual job list and dead-letter actions next to
  them, following the same self-fetching-tab pattern as
  `BackupsSecurityTab.svelte`.

## Slices B and C (proposed, not yet built)

Deferred to their own PRs. The risk worth naming now: Slice B's aggregate
query path is new SQL generation against the JSON-blob `hydro_collection_records`
table — it must reuse `query.go`'s existing field-path allowlisting exactly,
not introduce a second way to extract a field, or it reopens the
SQL-injection surface that file was written to close. Slice C's "declarative
action" must stay limited to interpretable actions (a field `Set`, an
existing hook call) — not a path to running arbitrary code from YAML.

## Consequences

- A project gets job visibility for free immediately (Slice A); reports and
  declarative scheduling remain project-specific until B and C land.
- No migration, no new module gate: Slice A is pure wiring of infrastructure
  ADR 0009 already made core-owned.
- Risk: Slice A's `List`/`DeleteDeadLetter` additions to `internal/jobs/sqlstore`
  are new read/delete paths on a table other workers mutate concurrently;
  covered by extending the existing cross-dialect lease contract test
  (`internal/jobs/sqlstore/store_test.go`) rather than a separate suite.

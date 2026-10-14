# 0002: One way to define a content type

- Status: Accepted
- Date: 2026-10-14

## Context

There were four overlapping ways to add a content type: hand-written modules,
blueprint code generation, `internal/collections` (runtime schemas) and
`internal/adminstudio` (admin metadata plus `GeneratedAdminHandler`, a full
runtime admin that nothing in the server used).

## Decision

A content type is a YAML file in `content-types/` (config `content_types_dir`,
env `HYDRO_CONTENT_TYPES_DIR`).

- `internal/contenttypes` loads the files and derives a `collections.Schema`
  (storage, validation) and an `adminstudio.Collection` (admin metadata).
- At startup the schemas are synced into collection storage. Fields may be
  added, removed or relabelled; changing an existing field's kind or relation
  target is refused because stored records would stop validating.
- `GeneratedAdminHandler` renders the admin at `/admin/resources/{type}`. It
  talks to `contenttypes.AdminAPI`, an in-process adapter over
  `collections.Service`; that API is not routed publicly.
- Record updates use optimistic locking (`version`), so concurrent edits get
  a conflict instead of overwriting each other.
- Types with `public: true` are readable at `GET /api/v1/content/{type}` and
  `GET /api/v1/content/{type}/{id}`; rich text is returned as HTML.
- The `collections` module is on by default.

`modules/pages` and `modules/news` stay hand-written for now because they
have drafts, scheduling, revisions and SEO behaviour that content types do
not have yet.

## Consequences

- New simple content needs no Go code. The 11 sectoral modules removed in
  0001 come back as YAML files when needed (FAQ, team and testimonials ship
  as examples).
- Not yet supported for content types: localisation, separate draft copies
  of published records, scheduling, array and block fields. They will be added to collection storage
  behind the callbacks `GeneratedAdminHandler` already defines.
- Lists support search (`searchable: true` fields), filters
  (`filterable: true`) and sorting by list columns. Collection storage runs
  them as dialect-specific JSON queries (SQLite `json_extract`, PostgreSQL
  `->>`, MySQL `JSON_EXTRACT`) with field paths as bound parameters, after
  checking every field ID against the schema. The public API accepts
  `?search=`, `?filter[field]=` and `?sort=[-]field`.
- `drafts: true` gives records a `status` field (`draft` or `published`)
  driven by `GeneratedAdminHandler`'s workflow mode (`CanTransition` /
  `Transition`): new records are drafts, Publish and Move to draft change the
  status with version checks, saves keep it, and the public API filters on
  it so drafts are never served. Editing a published record changes the live
  record; the handler's separate draft-copy mode (`Drafts`, with draft
  storage, publish and discard) was not used because it needs a second
  storage model for working copies. A workflow therefore no longer requires
  `Revisions` in `adminstudio`, only versioned records.
- `revisions: true` stores a snapshot of every saved version in
  `hydro_collection_revisions` (migration
  `202610140001_collection_revisions`, module `collections`), written in the
  record's transaction with the acting user, capped at 50 per record and
  deleted with the record. The admin lists, compares and restores them
  through `GeneratedAdminHandler`'s revision callbacks; a restore saves a new
  version with optimistic locking and keeps the record's current status.
- The blueprint generator and the browser-based schema builder at
  `/admin/collections` (with its `/api/v1/collections` API) overlapped with
  this and have been removed.

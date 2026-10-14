# ADR-0224: Global resource contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0223](0223-generated-admin-ui-and-publishing-preview.md)

## Context

Not every domain resource is a collection. Site settings, SEO defaults and
edition-level configuration are singleton resources, but they still need the
same typed validation, authorization, migration and API boundaries.

## Decision

Blueprints may set `type: global`; omitted type remains `collection` for
backward compatibility. The type is carried through the versioned Spec,
generated module manifest and Admin Studio metadata. A global resource uses the
same reviewed migration and service/policy layers, but its transport and admin
composition must expose a singleton form/API and enforce the one-record
invariant transactionally.

The generator does not create a hidden runtime DDL path or infer authorization.
Edition composition owns the singleton route, tenant scope, validation and
policy binding. Generated global transports resolve an empty resource path to
the stable `global` identifier and never expose collection-list semantics; a
future migration that changes a collection into a global (or the reverse) is a
breaking schema diff and requires an explicit migration plan.

## Consequences

- Resource metadata can describe both collections and globals without a second
  generator pipeline.
- Existing Blueprints remain valid and deterministic.
- Singleton semantics cannot be accidentally obtained by merely changing an
  admin label; the service/repository contract must enforce it.

## Verification

`internal/generator` tests cover `type: global`, default collection behavior,
strict Blueprint decoding, generated Spec metadata and singleton transport
resolution. Edition-level global persistence and PostgreSQL/SQLite singleton
race certification remain a follow-up implementation gate. Generated global
services normalize the ID to a stable `global` key before validation and
persistence; this is defense in depth, not a replacement for the
transaction/database invariant.

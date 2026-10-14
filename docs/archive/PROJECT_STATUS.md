# HydroGo Project Status

This document records what the repository currently provides and what the
recent reliability and security work was intended to achieve. It is a status
summary, not a claim that every deployment profile has production
certification.

## What HydroGo solves

HydroGo is a customer-owned Go CMS and JSON API platform for teams that need
content, identity, media, backups, and release operations without forking a
central Core or installing services they do not use. Its central promise is
portable control of customer data and deployments: the same application
services and authorization rules serve the HTML admin experience and the JSON
API, while SQLite and PostgreSQL implement the same behavioral contracts.

The design favors explicit boundaries over a large all-in-one CMS. A module is
enabled only when it is needed; inactive modules do not create routes,
migrations, workers, assets, or mandatory infrastructure.

## Current capabilities

- Pages and News support drafts, revisions, previews, optimistic locking,
  scheduling, publishing, withdrawal, and background work.
- Site services cover navigation, settings, SEO metadata, redirects, and form
  inbox flows.
- Authentication includes sessions, revocation, MFA foundations, password
  reset, invitations, CSRF protection, default-deny authorization, audit
  events, transactional outbox, job leasing, idempotency, and dead-letter
  operations.
- The versioned staff mobile API and the separate consumer/member mobile API
  have opaque short-lived bearer tokens, transactional refresh rotation,
  device-family revocation, disabled-identity invalidation, throttling, audit
  coverage, OpenAPI compatibility baselines, and platform-neutral client
  fixtures. Member-private cart, checkout, order history and ownership-scoped
  payment status are available under the member boundary; neither audience is
  production-certified until its external threat/device/provider drills pass.
- Media has explicit pending/ready and public/private lifecycles, bounded
  upload handling, safe metadata, and manifest verification.
- SQLite is the compact default. PostgreSQL has shared database contracts,
  migrations, compatibility checks, transaction behavior, and erasure replay
  coverage.
- Backup and restore are staging-first. Snapshot manifests are hashed, restore
  activation is quarantined and operator-controlled, retention is explicit,
  and destructive cleanup is bounded and revalidated.
- Portable export/import uses an allowlist and excludes secrets and session
  state. Imports are bounded, dry-run capable, confirmation-gated, and
  transactional.
- Release tooling verifies signed manifests against an operator-supplied
  trusted Ed25519 public key, verifies retained assets, records provenance,
  and supports deterministic generation and Client Edition boundaries.
- Project generator (`hydro new`) produces decoupled Client Edition project scaffolds
  for news, editorial, commerce, compact-site, corporate, portal, and headless
  presets without copying HydroGo Core code.
- Cryptographic licensing (`internal/licensing` and `hydro license`) provides offline
  Ed25519 license verification, grace period management, lockfile consistency (`hydrogo.lock`),
  and fail-closed runtime enforcement.
- Commerce is deferred and has no runtime, API, provider, worker, or configuration
  surface in this release. Its historical migrations are retained; a future return
  requires a new activation decision under ADR-0199.
- Cache behavior includes bounded L1 caching, optional TwoTier caching,
  generation invalidation, stale-refill rejection, single-flight misses, and
  invalidation fanout/reconciliation tests.
- The operational surface includes health/readiness checks, bounded metrics,
  doctor output, migration and upgrade commands, local smoke drills, and
  release/backup/restore command flows.

## How the system works

```text
hydro serve / migrate / doctor / upgrade check
                         |
                         v
                 HTTP handler / JSON API
                         |
                         v
                 application service + policy
                         |
                         v
                  module repository contract
                         |
                    SQLite or PostgreSQL
```

Domain and application services do not select SQL dialects, cache drivers, or
deployment-proxy behavior. Those concerns stay behind adapters and operational
boundaries. A release or restore is therefore validated as a sequence of
explicit contracts rather than as an implicit collection of scripts.

## Reliability and security hardening delivered

The recent hardening series added or verified:

- shared path validation that rejects empty roots, symlink components, unsafe
  parents, and path traversal before reads, writes, activation, or pruning;
- revalidation immediately before destructive or externally visible file
  operations, including erasure logs and generated output;
- bounded manifest, marker, JSON, multipart, upload, portable, and HTTP body
  reads, including protection against integer-boundary overflow;
- streaming hashing and verification for large release artifacts;
- trusted-key checking for release signatures rather than trusting a key
  embedded in the manifest;
- nil-context normalization at database, server, portable, erasure, GC, and
  mailer boundaries;
- cache-generation and blue/green concurrency contracts;
- PostgreSQL erasure replay idempotency and SQLite/PostgreSQL pool and dialect
  validation;
- response-writer capability preservation in HTTP middleware and safer hook
  dispatch with retry, coalescing, and re-entrancy behavior.

## Evidence available today

The repository's local suite, vet checks, action guard, import guard, and
format checks have been run successfully on the current Windows workspace.
The exact commands and evidence are recorded in
[`docs/validation/R7-reliability-security-hardening-2026-09-10.md`](validation/R7-reliability-security-hardening-2026-09-10.md).

That evidence does not replace the remaining environment-dependent gates:

- a fresh Linux race run;
- PostgreSQL contract tests against a clean, correctly migrated database;
- restore into clean SQLite and PostgreSQL targets;
- two-process blue/green restart and rollback behind a real reverse proxy;
- external backup/key recovery, signing-key ceremony, and representative
  RPO/RTO and capacity measurements.

These are intentionally kept visible in
[`docs/validation/PRODUCTION-GATES.md`](validation/PRODUCTION-GATES.md) and
the [roadmap](ROADMAP.md). HydroGo should be called production-ready for a
profile only after that profile has evidence for its own deployment and data
recovery assumptions.

The staff and consumer/member mobile API evidence is recorded in
[`docs/api/MOBILE_API_SPECIFICATION.md`](api/MOBILE_API_SPECIFICATION.md), the
machine-readable staff/member auth and commerce contracts, and the SaaS/CMS
readiness plan. It still requires ADR-0139/0150 real-environment abuse,
device-loss and payment/provider drills before public enablement.

## Where to continue

Start with the roadmap and the affected ADR before adding a feature. Prefer
the smallest change that closes a measurable acceptance criterion. When a
feature changes a contract, update the implementation, its certified tests,
the migration or compatibility metadata, and the deployment/capacity notes in
the same change.

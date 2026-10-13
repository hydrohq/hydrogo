# Real-World Production Readiness Roadmap

**Date:** 2026-09-18  
**Purpose:** Transition HydroGo from controlled pilot runs to scalable, measurable, and supportable production deployments.

This document is not a wishlist of features; it is a prioritized framework of required evidence and operational decisions. A phase is considered complete not merely when code is merged, but when defined acceptance criteria are measured and documented against real environments.

---

## Baseline

Today, HydroGo maintains a strong technical foundation: certified SQLite/PostgreSQL contracts, auth with TOTP MFA, CSRF protection, audit logging, idempotent webhooks and jobs, automated backup/restore, module boundary isolation, project scaffolding, binary health probes, and automated verification guards.

The experimental/pilot status of components and open gates are documented in [`PROJECT_STATUS.md`](PROJECT_STATUS.md) and [`OPEN_GATES_AND_OPERATOR_ACTIONS.md`](OPEN_GATES_AND_OPERATOR_ACTIONS.md).

---

## Phase 0 — Scope & Release Governance

**Goal:** Establish reproducible commitments for supported profiles and platforms.

- Pin supported Go, OS, database, Redis, and storage provider versions across Compact, Standard, and Distributed profiles.
- Define Pages/News, Commerce, and future modules as explicit capability tiers; keep experimental features isolated.
- Ensure `hydrogo.lock`, release manifests, edition tiers, and feature flags form a singular, verifiable release identity.
- Publish versioning, deprecation, security patching, and LTS support policies.

**Exit Criteria:** Approved [Support Matrix](SUPPORT_MATRIX.md), release checklists, and explicit statements of supported/unsupported configurations per profile.

---

## Phase 1 — Secure Packaging & Supply Chain

**Goal:** Deliver reproducible, verifiable, secret-free release artifacts to users.

- Produce reproducible binaries with pinned toolchains.
- Keep signing keys within isolated key-management systems; separate issuer tooling from runtime binaries.
- Publish Software Bill of Materials (SPDX SBOM), vulnerability scan logs, CodeQL analysis, and artifact provenance attestations.
- Automate checks verifying that release packages contain no internal core source files, private keys, or build secrets.
- Publish signed artifacts and SHA-256 checksums for Linux, macOS, and Windows.

**Exit Criteria:** Clean-machine package install, license/manifest verification, and checksum validation pass via `hydro version --json`.

---

## Phase 2 — Production Security & Identity Hardening

**Goal:** Enforce hardened defaults across internet-facing surfaces.

- Verify HTTPS termination, secure cookies, HSTS, and canonical origin routing behind real reverse proxies.
- Use secret managers for session signing keys, MFA master keys, database passwords, and API credentials.
- Formalize runbooks for initial admin creation, MFA enforcement, password resets, invitations, and session revocations.
- Validate email delivery, bounces, retries, and failure alerts with live SMTP relays.
- Run automated security test suites for authorization, CSRF, rate-limiting, file upload sanitization, and error scrubbing prior to penetration testing.

**Exit Criteria:** All P0/P1 threat model items verified; real SMTP and HTTPS smoke test reports published.

---

## Phase 3 — Data Durability & Upgrades

**Goal:** Guarantee verifiable zero-data-loss recovery and seamless upgrades.

- Restore customer-scale datasets to clean SQLite and PostgreSQL target instances.
- Rehearse backup, migration, rollback, and media activation procedures, measuring duration, disk consumption, and data integrity.
- Execute two-process blue/green transitions behind real reverse proxies, verifying zero downtime, graceful draining, and rollback safety.
- Test access-loss and key-recovery drills when using cloud S3 storage and external key escrow.
- Validate data retention, cryptographic erasure, and session invalidation against audit logs.

**Exit Criteria:** Profile-specific RPO and RTO metrics measured; clean restore, rollback, and data integrity reports signed off.

---

## Phase 4 — Observability & Operations

**Goal:** Detect, diagnose, and resolve anomalies without leaking sensitive user data.

- Route health probes, Prometheus metrics, structured logs, and request correlation IDs to centralized observability platforms.
- Test alert delivery across critical conditions: backup failures, disk saturation, failed migrations, SMTP delivery failures, and high error rates.
- Enforce strict PII and credential scrubbing policies across logs, metrics, and traces.
- Validate operational runbooks through live incident response drills.

**Exit Criteria:** Measurable alert-to-resolution drills; defined on-call owners, escalation paths, and error budgets.

---

## Phase 5 — Performance & Capacity Baselines

**Goal:** Provide concrete hardware-bound throughput and capacity baselines rather than vague claims.

- Model representative datasets and traffic workloads for Compact, Standard, and Distributed profiles.
- Measure process RSS, CPU utilization, DB connection pool utilization, queue latency, cache hit ratios, and p95/p99 latencies.
- Execute soak tests, traffic bursts, concurrent editor collisions, media transcode pipelines, and Redis network partitions.
- Document autoscaling thresholds, resource recommendations, and known capacity limits.

**Exit Criteria:** Documented capacity tables with explicit hardware profiles, traffic ceilings, and measured p95/p99 response curves.

---

## Phase 6 — Module & Integration Certification

**Goal:** Validate and certify turn-key domain workflows under realistic conditions.

- Complete responsive public and admin browser test journeys for Pages and News across draft, revision, schedule, and deletion states.
- Verify payment provider webhooks, signatures, idempotency, refunds, timeouts, and inventory race handling with sandbox accounts.
- Verify S3 and CDN integrations: IAM scoping, bucket retention, CDN cache invalidation, and signed private URLs.
- Isolate experimental capabilities behind feature flags and ADRs.

**Exit Criteria:** Published capability support matrices, E2E test reports, and explicit documentation of known constraints.

---

## Minimum Gate for Declaring Production Readiness

A configuration profile may only be designated "Production Ready" once all the following gates are cleared:

1. Release binary is reproducible, verifiable, and free of embedded secrets.
2. HTTPS, secret management, MFA, SMTP relay, and alerts are verified in live environments.
3. Clean-target restore, rollback, and data integrity verification have been documented.
4. Profile-specific capacity baselines, soak test metrics, and RPO/RTO measurements are published.
5. Vulnerability audits and threat models have resolved all blocking findings.
6. Runbooks, support matrices, deprecation policies, and operational responsibilities are current.

---

## Verification Methodology

When an architectural change is made, update the relevant ADR, dialect migration metadata, test fixtures, and capacity notes within the same pull request. Validation reports are recorded in `docs/validation/` with dates, commits, toolchains, commands, and results. Technical sequencing is tracked in [`ROADMAP.md`](ROADMAP.md); operational actions reside in [`OPEN_GATES_AND_OPERATOR_ACTIONS.md`](OPEN_GATES_AND_OPERATOR_ACTIONS.md).

# ADR-0014: Publication routes and safe recovery with erasure history

- **Status:** Accepted
- **Date:** 2026-09-05
- **Partially supersedes ADR:** [ADR-0012](0012-content-media-and-portability.md) for slug/301 activation time; [ADR-0013](0013-identity-and-security-boundaries.md) for the source of deletion records during restore.
- **Related ADR:** [ADR-0011](0011-consistency-upgrade-and-recovery.md)
- **Implementation status:** Partially implemented — route/canonicalization, idempotent erasure replay, and SQLite/PostgreSQL SQL adapter contract evidence exist. Actual post-backup deletion, external storage provider, retention/key recovery, and clean environment restore evidence are open as an R3/S07 gate.

## Context

The draft body was separated from the live revision; it was unclear when slug and SEO changes become visible. It was desired to reconcile deletion records after reverting to an old backup; the DB backup itself does not contain deletions that occurred after the backup.

## Publication decision

- Draft slug, canonical/SEO fields, and the body belong to the revision. Saving a draft does not alter the live route, redirect, sitemap, or public API projection.
- The publish transaction updates the revision pointer, active route, 301 record from the previous active route if needed, audit/outbox, and publication generations all together. If there is no previous active route in the initial publication, no redirect is created.
- Slug normalization, reserved path, uniqueness, and redirect loop are re-checked during publish. If a path that seemed available in the draft was subsequently taken, publish yields a conflict; the previous live state is preserved.
- There is no draft path reservation in the first slice. Two drafts can request the same path; the first successful publish wins. The error is visible to the user.
- Preview uses a revision-bound custom URL; it does not change the public route or shared cache. Scheduled publication runs the same publish policy with a static revision and schedule generation.
- Public cache propagation is subject to the stale budget of ADR-0011. Redirect and sitemap invalidation are also included in the same publication event.

## Erasure history and restore decision

This protocol is not for ordinary unpublish/soft-delete, but for permanent deletion or anonymization requests that must be enforced even after a restore.

- The deletion intent is first recorded in a durable log maintained off-server, independent of the production DB's restore process, and an acknowledgment of persistence is received. The audit/outbox in the same DB alone is not sufficient.
- The record carries the edition ID, unique transaction ID, sequential checkpoint, timestamp, operation type, and static target IDs sufficient for re-application. It does not carry deleted content, form body, or secrets; the IDs are also kept restricted and encrypted.
- Then, the deletion/anonymization and audit transaction are applied in the DB. An atomic transaction between the external log and DB is not assumed. If the intent is recorded but the DB operation fails, the operation remains pending and is replayed idempotently; it is not reported as completed to the user. If the log is unreachable, permanent deletion is not completed.
- The intent in the log is not an ordinary reversible edit. Authorization and target checks are performed before the intent is written. It must be able to safely continue content, revision, relation, media, and search cleanup using the same transaction ID.
- The backup manifest contains the log identity and the checkpoint it observed. Restore fetches the current log independent of the backup and applies applicable deletion intents idempotently to the staging target. Post-backup records are specifically covered.
- Public traffic, export, worker, or external egress are not opened until the log's integrity, sequence continuity, and current checkpoint are validated. Recovery with a missing log stays in isolation; the operator must recover the missing source. The old backup's checkpoint alone is not proof of currency.
- Log retention is coordinated with the oldest backup allowed to be restored and all its external copies. Before a required deletion record is dropped, the relevant old backups are removed from the supported restore set. The log and key recovery drill is a pilot gate.
- A separate continuously running service is not mandatory for Compact: a versioned log adapter over the existing secure external backup target can be used. The definitive provider, concurrent append/checkpoint, and durability protocol will be chosen in the R3 experiment. No permanent deletion/restore guarantee is provided without this evidence.

## Alternatives

Activating the slug change upon draft save breaks the draft/live separation. Path reservation for every draft adds unnecessary lifecycle in the first slice. Keeping the deletion history only in the DB backup loses post-backup requests. Instead of assuming general DB+external log atomicity, durable intent and idempotent completion are chosen first.

## Impacts and compatibility

- SQLite/PostgreSQL: route uniqueness, publish conflict, and content/route/audit/outbox atomicity pass the same contract kit. SQL and locking remain within the adapter; a new DB must provide the same semantics.
- Migration: the required metadata for route/revision relations and deletion transaction ID is prepared per dialect during implementation. No SQL migration is published yet. If the old reader/writer cannot preserve this semantics, an online release is not declared.
- Compact: route cache remains bounded; external log calls carry a deadline/concurrency budget. There is no unbounded retry or memory queue.
- Standard/Distributed: concurrent intent records must not be lost; global sequence/checkpoint and replay behavior must pass the provider test. All web/worker caches observe the publication generation.
- Blue/green: the two slots support the same publish and deletion protocol. Restore shuts down the old deployment writes; session revocation, retained assets, and job quarantine are subject to ADR-0011.

## Mandatory evidence

1. A draft slug/SEO change on a live page does not affect the public URL, canonical, and sitemap; publish moves them all to the correct revision.
2. When two contents publish to the same path, only one transaction succeeds; the loser preserves its old live state. A redirect loop and a failure mid-publish result in a rollback in both DBs.
3. Monday backup → Tuesday permanent deletion → Wednesday restore to Monday: data does not reappear from public, export, revision, media, and search.
4. DB crash after intent acknowledgment, log outage, missing/old checkpoint, and duplicate deletion operations yield safe results.
5. In case of log retention/key loss, restore does not open traffic; the external log and backup are recovered together in a clean environment.

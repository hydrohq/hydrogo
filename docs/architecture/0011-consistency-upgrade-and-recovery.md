# ADR-0011: Cache consistency, transaction boundary, upgrade and recovery

- **Status:** Accepted
- **Date:** 2026-09-05
- **Partially supersedes:** [ADR-0003](0003-cache-architecture.md) invalidation/failure guarantees; [ADR-0005](0005-zero-downtime-deployment.md) warm-up/transition validation.
- **Scope:** L0–L3 layers of previous decisions, profile selection, session/asset rules, proxy and expand–migrate–contract approach are preserved. The items below replace conflicting general statements.
- **Implementation status:** A significant portion of migration, cache, restore, blue/green and job/lease provisions have been implemented and verified with contract tests. Actual proxy/process, multi-instance and customer restore drills are open; current status is in [roadmap](../ROADMAP.md).

## Context and findings

If the worker is a separate process, clearing its own L1 does not clear web L1s. A cache key containing the release version does not keep the published data updated if it changes during the same release. An old binary that rejects any newer schema cannot restart after an expand migration. If the network response of an email sent after a transaction is lost, the idempotency key alone does not prevent double sending. PostgreSQL restore cannot be reduced to a file directory rename operation.

## Transaction and job contract

- Content, revision, publication pointer, audit, and outbox are written in the same DB transaction. Module repository and Core audit/outbox ports are instantiated from the same transaction scope; they cannot escape to a separate pool connection. If commit fails, none of them become visible.
- Transaction retry is only performed on DB operations safe to retry, with limited attempts and a deadline. The HTTP response is not considered successful until the commit is complete.
- The job queue is initially in the same DB: it carries `pending/leased/succeeded/dead/parked`, attempt, next-attempt, lease-owner, lease-until, fencing/claim token, and payload-version. Exact Go field names are determined in the prototype.
- Claim is atomic. Completion/renewal is accepted only from the current claim token owner. Time on the DB side and adapter tests make the lease calculation consistent. An expired worker can still reach an external service; fencing protects the DB update, it does not protect the external side effect alone.
- There is a delivery record and idempotency scope per consumer. Aggregate revision order prevents an old event from undoing a new state. An unknown payload version is quarantined; it is not executed by an incompatible worker.
- Delivery is **at least once**. If the provider has an idempotency key, it is used. If not, for example if an email's response is lost after being accepted, the possibility of resending explicitly remains; reconciliation/operator decision is required. No general exactly-once promise is made.
- Compact uses limited workers with polling/backoff. PostgreSQL claim/locking optimizations are inside the adapter. A new DB passes the same crash/lease/retry contract kit.

## Cache and publication visibility

1. Immutable revision body can be cached. Which revision is current is a separate mutable lookup; just writing the revision to the key does not solve this lookup going stale.
2. The publication transaction increments content and collection generations; the outbox event carries these versions. The L1 entry is tagged with the generation it was loaded under. A late-completing old read cannot be written under a new generation.
3. Local invalidation after commit is performed in the single `all` process; outbox retry path remains. When web/worker are separated, a notification/fan-out to **every web process** is required. A single worker taking a single queue message is not a fan-out.
4. During Compact blue/green transition, the mutable public response/lookup cache is temporarily bypassed. Warm-up can be done after green readiness; generation is re-validated before traffic is switched, or the warmed mutable cache is cleared. A publication changing during slot transition does not lock in the old warmed response.
5. In the Distributed profile, alongside L2 invalidation, there is disconnection detection and generation reconciliation for each instance. Pub/sub alone is not a lossless correctness record. During a disconnect, mutable L1 is bypassed/flushed; it is not reused upon reconnection until synchronization is complete.
6. Bounded eventual consistency is accepted for normal public content. The policy defines the `max_stale` duration; L1, L2, edge, and browser ages cannot exceed this total budget. The TTL cannot restart at a layer transition and infinitely extend the old data's life. The numeric budget is assigned before the pilot based on traffic measurement and content needs.
7. Admin save result shows the new revision from DB. Completion of public publication is a separate state: `committed`, `propagating`, `propagated`, or `failed`. CDN purge confirmation does not mean the copy in the user's browser is deleted; instant validation is provided via private/no-store preview.
8. Drafts, private media, sessions, and responses personalized by authorization do not enter the shared cache. No guarantee is given to recall all user copies of content that has become public. Routes requiring rapid recall are separately certified with their cache policy and purge time.

## Cache failure and overload

Cache loss does not cause loss of the actual data; it does not mean the service will remain uninterrupted in every failure. Bounded DB fallback, singleflight, concurrency admission, circuit breaker, and retry budget exist for L2 failure. When DB capacity is full, a controlled 429/503 is preferred over an unlimited DB queue. Only public, permitted data within the `max_stale` limit can be served stale; this path does not exist for hidden content.

The cache key allowlist covers query normalization, edition/site, locale, representation (HTML/fragment/JSON), allowed host, publication generation, and serialization version. Personalized responses carrying Authorization/Cookie are bypassed. Host and `X-Forwarded-*` are accepted only from a validated domain and trusted proxy. `Vary` alone does not guarantee the cache provider will behave correctly; an edge contract test is required. An inactive module does not generate a key, invalidation worker, or tag inventory.

## Release compatibility metadata

The binary does not rely on a single "schema version" number. Core and active modules declare the readable/writable schema epoch range, required migration IDs, config/content/job/event formats, and the compatible release set. Ledger checksums and required migration existence are verified together; a wide number range alone is not sufficient.

Migration metadata includes the following for each dialect:

- Logical identity, module, dependencies, and published checksum
- `online/backfill/maintenance` class and transactional/nontransactional execution mode
- Prerequisites, lock/statement timeout, disk estimate, and smoke/postcondition checks
- Old/new binary read/write compatibility and job/content version impact
- Failure/restart procedure, backfill checkpoint, and rollback constraint

The online class is not chosen by the SQL statement's name. The same migration might be online in PostgreSQL and maintenance in SQLite. Application startup does not return readiness in case of missing/incompatible schema; the deploy orchestrator runs the migration separately. A "contract test" in production is not a test suite writing to customer data: contract tests run on an isolated DB copy/in CI, only safe readiness and smoke checks run in production.

## Blue/green and rollback

The ADR-0005 flow is preserved; preflight checks all format compatibilities and resource quota for the two slots. The old binary must also be able to restart with the current ledger after an online migration. Before the new worker starts, old leases are handed over with a drain/expiry policy; events produced by the old web must be readable by the new worker.

Rollback only reverts the binary; it does not undo data written by the new release. If new writes can be read by the old binary, returning is possible. Restore is a separate recovery operation that can lose post-backup writes; it is not presented as a silent automatic rollback.

If the new release only carries its own embedded assets, old HTML might break. Deploy extracts hashed files from the signed asset inventory to a common release asset directory or preserves the layer serving previous packages' assets. GC cannot delete these before the minimum rollback window and allowed HTML/CDN cache lifetime expire. The HTTP service application remains a single binary; retained assets are deployment data.

## Backup and restore

Backup (disaster recovery to the same DB), export (portable logical data), and binary rollback are separate commands/contracts.

- Compact's first implementation can halt writes/workers in a controlled manner to take a consistent DB snapshot and immutable media manifest. The expected pause/byte budget is calculated in preflight; if the time limit is exceeded, it is safely cancelled. No "short lock" promise is given during an entire large media copy.
- For Standard/Distributed, GC retention/pin is coordinated with the referenced immutable media inventory in the DB snapshot. Pin creation and deletion decisions use the same coordination boundary; it is not necessary to stop writes throughout the entire copy. Online large backups are not sold before this path is validated.
- Local/S3 object listing does not replace the DB snapshot. DB, media manifest, checksum, key references, versions, and backup state are tied to the same backup identity.
- SQLite can be restored to a closed target via a staging file/directory; platform-specific atomic activation and WAL/SHM cleanup are validated.
- PostgreSQL is restored to a separate empty DB/instance with its own restore method. After references/smoke checks are validated, application writes are stopped and the connection target is changed in a controlled manner. DB restore is not a "rename the media directory" operation.
- PITR support is offered only with a media retention and key recovery model compatible with the DB history. Having only DB PITR does not guarantee the same RPO for the entire product.
- Restored outbox/jobs can resend old emails/webhooks. A restore is initially started isolated, with worker and external egress closed; side effects are explicitly enabled after being reconciled.
- Restored sessions are revoked by default; signature/preview/API secret versions and the shutdown of the old active deployment are in the runbook. Secret recovery is on a separate secure channel, public keys are not kept next to the backup.

## Alternatives and validation

Synchronous DB+cache dual write is not chosen: it is not a single atomic transaction. Removing the cache entirely simplifies correctness but may not meet the load target; Noop/bypass is valid for unnecessary entries. An external broker for the queue is not a prerequisite. General down migration does not guarantee the safe reverse of data transformation.

Mandatory evidence: crash after commit, event replay/reordering, lease expiry, provider timeout ambiguity, two separate web L1s, lost invalidation, stale refill race, bounded DB load during L2 outage, old binary restart during migration, rollback after new writes, previous asset URLs, actual restore in two DBs, and post-restore side effect quarantine. Success metrics are tracked in the [pre-implementation review](../PRE_IMPLEMENTATION_REVIEW.md).

## References

- [SQLite isolation](https://www.sqlite.org/isolation.html): Single writer limit despite concurrent connections.
- [PostgreSQL CREATE INDEX](https://www.postgresql.org/docs/current/sql-createindex.html): Concurrent index and transaction/lock constraints; need for migration execution per dialect.
- [Transactional outbox](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html): Atomic DB+event recording and retry delivery possibility.

These references support the mechanism constraints; the flow and acceptance criteria preferred by HydroGo are product design decisions.

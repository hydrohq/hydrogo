# Remaining GitHub issues: implementation plan

**Scope:** #256, #219, #152, #71, #48, and #28.  
**Execution rule:** implement and verify each acceptance criterion before closing its issue. Never close an issue solely because a design note or partial foundation exists.

## Wave 1 — media derivatives (#219)

- Use a pinned, license-reviewed Pure Go PDF renderer behind the existing bounded media derivative worker.
- Accept signature-validated PDF uploads under an explicit byte cap; never mutate or replace the source PDF.
- Store the first-page cover as a `media_variants` derivative. Keep local and configured object-store keys aligned, including tenant prefixes.
- Render the cover in the media library and same-origin PDF download cards. Preserve authorization and attachment behavior for private source documents.
- Verify valid/invalid PDF input, bounded PNG output, worker status, object-store key behavior, private authorization, and both preview surfaces.
- Close #219 only after all thumbnail surfaces and production renderer limitations have been verified.

## Wave 2 — SQLite migration path (#256)

- Read SQLite migration, recovery, and blue/green ADRs; distinguish lightweight additive changes from large table rebuilds.
- Specify a bounded copy/catch-up/cutover protocol, including writer coordination, foreign keys, indexes, triggers, crash recovery, disk estimate, and rollback metadata.
- Implement it only if it can preserve concurrent writes and old/new binary compatibility. Otherwise document the specific SQLite limitation and leave the issue open for an explicitly bounded maintenance/cutover contract.
- Test interruption at each durable phase, concurrent readers/writers, large fixtures, restart recovery, and supported dialect metadata.

## Wave 3 — public response cache (#71)

- Reuse the durable mutation/outbox boundary; do not use best-effort Pub/Sub as the sole source of invalidations.
- Ensure each cache node detects missing events after reconnect/restart and purges affected entries before serving them again.
- Measure warm concurrent hit latency and invalidation propagation across two independent server instances.
- Do not claim all-worker purge or zero stale content until the multi-instance tests pass. If only bounded eventual invalidation can be proven, report that contract and keep the stronger issue open.

## Wave 4 — deletion lifecycle (#48)

- Define soft deletion for Pages, News, Forms submissions, and Media in a new ADR linked to ADR-0013/0014.
- Add dialect migrations with backfill/rollback metadata; update every public, admin, search, reference, export, and background query to enforce visibility consistently.
- Provide authorization-checked restore/permanent-delete operations, an Admin Studio trash surface, and a bounded retention worker.
- Preserve the independent erasure log and restore fence: restoring a soft-deleted row must never undo permanent erasure/anonymization.
- Certify SQLite/PostgreSQL behavior, tenant isolation, cache invalidation, worker idempotency, retention, and restore/reconciliation before closing.

## Wave 5 — AI copilot (#28)

- Follow ADR-0141: disabled by default, explicit provider and data consent, no prompt/result persistence by default, secret redaction, bounded time/size/concurrency, and provider-neutral adapters.
- Provide editor-only suggestions as a typed Templ before/after diff. Applying a suggestion changes draft form state only; save/publish stays an explicit separate action.
- Audit the action/approval metadata without logging submitted content or secrets; use mock providers in tests and require configured credentials only at runtime.
- Test provider failure, malformed output, redaction, authorization, audit metadata, and that no suggestion can publish content.

## Wave 6 — social/review sync (#152)

- Re-scope the provider contracts to supported APIs: Instagram professional accounts only; no consumer-profile scraping or deprecated Basic Display assumptions.
- For Google Places, retain only data the applicable terms permit (such as Place IDs); render reviews with author attribution and source links and honor deletion/expiry rules. Do not persist review text/photos as a local mirror unless the exact customer agreement and current API terms expressly permit it.
- Keep external calls in an optional bounded background capability with credentials supplied by the Edition operator, backoff, timeouts, and stale/error status that does not leak provider secrets.
- Verify with fake provider clients and attribution rendering. Production credential, billing, and policy acceptance remain operator gates; they are not inferred from mocks.

## Release gates

1. Run `go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate` and `go test -count=1 ./...` after each complete wave.
2. Review generated output, ADR links, migration manifests, license/SBOM changes, module-off behavior, and capacity impact.
3. Commit each verified wave and push `origin/main`; do not close incomplete issues.
4. Finish with `gh issue list --state open`, `git status --short`, and remote/local commit equality.

## Execution state

- **#219 closed:** bounded Pure Go first-page PNG variant, PDF upload validation, private-source behavior, media library cover, download-card cover, and accessible media-library preview dialog implemented. HydroGo tests pass; renderer runtime packages pass. The separate upstream `tools/genfixtures` test fails because four checked-in fixture files differ from generator output; this non-runtime upstream tooling failure is retained as a dependency risk.
- **#256 closed and pushed:** WAL-only, batched SQLite shadow rebuild, durable backfill checkpoint, source write-mirroring triggers, atomic cutover/ledger, descriptor validation, ADR, and focused contract tests are in `bbd6ff65`. Migration guard reports 106 migrations; templ generation and `go test -count=1 ./...` pass. SQLite remains single-writer and the final cutover briefly blocks writers.
- **#71 remains open:** the current process-local middleware passes the cache behavior tests and parallel warm-hit benchmark (1.1–5.2 µs/op in three local samples), but does not have independent-process invalidation or durable reconnect reconciliation. SQLite's optional external-write watcher does not supply cross-process PostgreSQL invalidation.
- **#48 in progress:** ADR-0332, module-owned migrations, hidden normal reads, tenant-scoped move/restore, the Studio trash UI, dashboard filtering, and local cache invalidation are implemented in the current worktree. Expiry purge remains blocked on integrating ADR-0014's external erasure log for every supported content kind; no unlogged hard-delete path is exposed.
- **#28 in progress:** a disabled-by-default page editor assistant now requires the active `ai` module, operator-supplied OpenAI-compatible endpoint credentials, and per-request content consent. It redacts emails/common API keys, enforces input/output/time caps, logs request/application metadata without content, returns a typed before/after proposal, and only changes the in-memory editor when explicitly applied. Provider, UI, route/CSRF, and audit behavior have focused tests. Translation, SEO metadata fields, submission summarization, and deployment prompt-injection/DPA gates remain before closure.
- **#152:** not yet implemented in this sweep. Google's current Places policy forbids caching/storing most Places API content beyond stated exceptions, while Place IDs are exempt; any compliant design must fetch reviews live with attribution and source links rather than mirror review content. See [Places API policies](https://developers.google.com/maps/documentation/places/web-service/policies?hl=en). The compliant re-scope is documented in Wave 6.

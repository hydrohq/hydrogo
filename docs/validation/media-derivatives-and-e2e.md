# Media Derivatives and Automated E2E Status Report

Date: 2026-09-12  
Status: Local code and test validation completed; actual encoder installation is an open gate.

## Media Optimization

The media upload pipeline was expanded to preserve the original file as an immutable source. Image and video derivatives are stored in separate metadata records:

- `media_assets` represents the original file record and retains existing serving behavior.
- `media_variants` stores ID, status, MIME type, size, and error details for WebP/WebM derivatives.
- `?variant=card` and `?variant=video` requests serve the ready derivative; if the derivative is not ready, they safely fall back to the original.
- In addition to JPEG, PNG, and WebP, MP4, WebM, and QuickTime uploads are accepted.
- The Admin media form supports image and video uploads; video preview is displayed once metadata is ready.

Transcoding does not run within the upload request. `internal/media.Optimizer` runs in the background with a fixed-capacity queue and bounded workers. A 2-minute timeout is enforced per job; FFmpeg is invoked with a single codec thread. When the queue is full, uploads and original media remain unaffected.

The encoder path can be configured via `HYDRO_MEDIA_FFMPEG`. If empty, `ffmpeg` in `PATH` is searched. Because FFmpeg was not present on this machine, actual WebP/WebM generation has not yet been executed; in this scenario, the original file remains usable and the derivative is recorded in `failed` status with a bounded error message.

## RAM and Capacity Policy

- Original uploads are streamed and protected by the existing `MediaMaxBytes` limit.
- The derivative queue holds at most 8 pending assets.
- Worker count is derived from the profile and capped at 2 for media.
- In the Compact profile, a single transcoding job runs under normal usage.
- FFmpeg jobs use a single codec thread and process timeouts.
- Because original files are not deleted, encoder failures do not result in data loss.

These limits prevent RAM from scaling unboundedly with the number of uploads. An OS-level RAM quota for FFmpeg has not yet been applied; for high-volume deployments, container/job-object limits and a durable media job worker decision must be made separately.

## Architectural Decision

The decision is recorded in `docs/architecture/0026-bounded-media-derivatives.md`. New migration:

- `migrations/202609120001_media_variants`

RabbitMQ/BullMQ was not added. In accordance with HydroGo's current boundaries and Compact philosophy, the initial implementation was retained as a bounded in-process worker. If the process restarts, pending derivative jobs may be lost; original media remains unaffected. When multi-process durability is required, a durable media job type should be added on top of the existing SQL job queue.

## Automated Playwright E2E

The metadata-driven E2E generator was expanded. For each frontend route in `e2e/manifest.json`, the generator now produces:

- Desktop route smoke test
- 390x844 mobile viewport test
- Root test ID visibility test
- Horizontal overflow (`scrollWidth`) check

On the Admin side, the media upload journey was also added to the manifest. The test uploads a small PNG and verifies the redirect and the filename on the media card. Thus, the media upload surface is included in generatable E2E coverage.

Generated file:

- `e2e/generated/hydro.spec.ts`

`scripts/run-e2e.ps1` was executed against an isolated SQLite server using a test-only admin password:
30/30 tests passed on Chromium in the latest run. The run covers the published content API client, admin login,
five admin sections, media upload, Pages/News forms, public home/shop routes, and
two mobile horizontal-overflow checks. The same run now executes in Windows CI via the mandatory
`browser-e2e` job in `.github/workflows/ci.yml`. The runner temporarily unsets any leftover
`HYDRO_REDIS_URL` from the development environment to preserve Compact SQLite fixture isolation.

## Test Evidence

Successful checks:

- `go test ./tools/e2egen -p 1 -timeout=5m`
- `go test ./migrations ./internal/media/... ./cmd/hydro/... -p 1 -timeout=5m`
- `go vet ./...`
- `git diff --check`
- `scripts/run-e2e.ps1` — 30/30 PASS on Chromium

## Open Items

- `[!]` Installing FFmpeg in the test environment and running actual WebP/WebM transcoding smoke tests.
- `[?]` Migrating media jobs to an SQL durable queue for multi-process production deployments.
- `[?]` Enforcing strict container/job-object level RAM/CPU quotas for FFmpeg.
- `[?]` Adding module-specific cart, news detail, member auth, empty/error, and accessibility E2E flows to the manifest for each frontend.

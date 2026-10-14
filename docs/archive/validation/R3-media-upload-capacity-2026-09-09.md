# R3 media upload capacity validation

## Scope

`media.Uploader` now executes per-upload temporary file writing, SHA-256 calculation, and image size validation via a bounded semaphore. The runtime bootstrap binds this limit to the `WorkerConcurrency` budget of the selected capacity profile. Full payload decoding is also performed after the image size header; thus, truncated or corrupted images are not admitted to immutable storage. If the context of a request waiting on the semaphore is canceled, the upload stops before it begins. If the uploader or asset repository is wired incorrectly, the media service returns a fail-closed error instead of panicking; the HTTP upload/serve boundary also safely rejects these missing dependencies with a 503.

## Evidence

- Commit: `674df9b`
- Environment: Windows development environment, compatible with the Go toolchain repository CI version
- Commands:
  - `go test ./internal/media ./cmd/hydro`
  - `go vet ./internal/media ./cmd/hydro`
  - `go test ./... -p 1`
  - `go vet ./...`
  - `git diff --check`
- Result: all tests and static checks succeeded.
- Contract: `TestUploaderSaveContextHonorsCancellationWhileBusy` validates that a second pending upload exits boundedly with `context.Canceled` while the single slot is full.

## Limits

This evidence demonstrates process-local upload backpressure behavior. Throughput on actual customer hardware, disk I/O saturation, multi-instance aggregate capacity, and object-storage provider behavior are not certified by this report.

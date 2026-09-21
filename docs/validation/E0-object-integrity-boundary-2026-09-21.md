# E0 object integrity boundary validation

- **Date:** 2026-09-21
- **Scope:** Provider-neutral object read verification before a workflow
  transitions a pending object to `ready`.
- **Not evidence for:** direct browser-to-object-store upload, multipart
  provider certification, malware scanning, or a completed quarantine worker.

## Implemented boundary

`media.VerifyObject` consumes only the `BlobStore.Get` abstraction. It reads no
more than the expected byte count plus one byte, validates the exact byte count
and SHA-256 digest, and fails closed for an unavailable store, read/open error,
overlong object or digest mismatch. This catches lazy read failures from object
store SDKs instead of treating a successful `Get` handle as evidence that the
object exists and is intact.

The function deliberately does not mark metadata `ready`; the owning workflow
must run it before its transactional status transition. A failed verification
leaves the pending/quarantine object outside public readiness.

## Command run

```text
go test ./internal/media -count=1
```

Result: pass.

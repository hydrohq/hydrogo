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

`S3BlobStore.PresignedQuarantineUploadURL` additionally exposes a direct PUT
capability only for `quarantine/...` or `tenants/{tenant}/quarantine/...`
keys, with a 1-second to 15-minute expiry. It rejects final object keys. The
URL does not imply readiness, does not set public visibility, and still needs
an authorized completion workflow plus `VerifyObject` before promotion.

`media.Service.PrepareQuarantineUpload` selects an opaque random key on the
server and prefixes it with the validated tenant namespace when present. The
browser never selects a final key. The method deliberately creates no media
asset row and cannot mark a row ready; the later completion endpoint must bind
the pending record, checksum expectation, actor authorization and promotion in
one reviewed workflow.

## Command run

```text
go test ./internal/media -count=1
```

Result: pass.

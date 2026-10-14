# E0 Media Quarantine Completion — 2026-09-21

`internal/media.Service.CompleteQuarantineUpload` now verifies the provider
object size and SHA-256 before promotion. It rejects non-quarantine keys and,
when the SQL repository supports it, atomically requires the pending asset's
stored key to equal the server-issued quarantine key before marking it ready.

Checks:

```text
go test ./internal/media/... -count=1
```

The remaining production gate is a live S3-compatible provider drill and
private-media authorization through the deployed HTTP surface.

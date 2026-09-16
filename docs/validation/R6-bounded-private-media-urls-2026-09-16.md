# R6 bounded private media URLs — 2026-09-16

S3-compatible media storage now exposes an optional short-lived presigned URL
capability with 1 second–24 hour bounds and shared object-key validation. The
repository test suite verifies fail-closed behavior without requiring a real
provider account.

Verification:

```text
go test ./internal/media -count=1
PASS
```

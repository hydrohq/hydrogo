# R6 S3/CDN public URL boundary — 2026-09-16

S3-compatible storage now honors path-style addressing and exposes an optional,
explicit public/CDN URL provider. URLs are HTTPS by default, object keys are
bounded and traversal-safe, and no provider configuration is required for the
local filesystem path.

Verification:

```text
go test ./internal/media -count=1
PASS
```

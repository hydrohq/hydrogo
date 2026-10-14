# R6 protected media URL endpoint — 2026-09-16

Added protected `GET /api/media/{id}/url` for optional S3-compatible presigned
URLs. Only active editors/admins can request URLs for ready assets; expiry is
bounded to five minutes by default and 24 hours maximum.

Verification:

```text
go test ./internal/media -count=1
PASS
```

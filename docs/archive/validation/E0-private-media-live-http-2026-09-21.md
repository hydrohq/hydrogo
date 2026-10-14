# E0 Private Media Live HTTP Authorization — 2026-09-21

**Commits:** `67dd2298`, `e6f77de3`

`internal/media` now includes a real `httptest.Server` network-path contract
for `GET /media/{id}`. The test verifies both sides of the authorization
boundary:

- anonymous request to a ready private asset returns `401` and does not expose
  the object;
- an authenticated editor context, supplied through the test authentication
  middleware boundary, returns `200` and the object bytes.

```text
go test ./internal/media -run TestPrivateMediaLiveHTTPRejectsAnonymous -count=1
ok  github.com/ardaozyildirim/hydrogo/internal/media
```

This certifies the live HTTP handler path and authorization decision. It does
not claim certification of an external S3/MinIO provider, CDN policy, or
production identity/session middleware; those remain deployment gates.

# R22 — Site module zero-construction guard (2026-09-17)

The server bootstrap now constructs the Site service, form registry binding,
rate limiter and Site HTTP handler only when the `site` module is enabled.
Previously these objects were created unconditionally and only their routes
were omitted. The route guard remains unchanged, while disabled Site profiles
avoid the extra mutation dependencies and limiter allocation.

Verification: `go test ./cmd/hydro -count=1` and `git diff --check` — PASS.
The existing preset migration and route helper contracts remain required; a
real customer deployment still needs the full per-preset route/worker/browser
acceptance gate.

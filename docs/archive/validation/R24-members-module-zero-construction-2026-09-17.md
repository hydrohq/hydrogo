# R24 — Members module zero-construction guard (2026-09-17)

The bootstrap now creates the Members HTTP handler and its rate limiter only
when the `members` module is enabled. Commerce profiles may still create the
Members application service because Commerce uses it for authenticated customer
session resolution, but they do not register the Members HTTP surface or its
dedicated limiter.

Verification: `go test ./cmd/hydro ./modules/members/... ./modules/commerce/... -count=1`
and `git diff --check` — PASS. Full verification remains required by the push
hook; preset route/migration acceptance remains a separate deployment gate.

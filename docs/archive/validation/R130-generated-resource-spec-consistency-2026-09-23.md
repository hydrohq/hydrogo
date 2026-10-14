# R130 — Generated resource spec/source consistency — 2026-09-23

## Scope

Require generated-resource compatibility checks to validate the embedded spec
version and compare its normalized resource contract to the requested source,
separate from per-artifact digest verification.

## Verification

- `go test ./... -p 1 -timeout=5m` — passed.
- With `HYDRO_REDIS_URL` and `HYDRO_REDIS_KEY_PREFIX` unset,
  `go test -tags hydro_no_redis ./internal/generator ./cmd/hydro -count=1` — passed.
- `TestCheckCompatibilityRejectsEmbeddedSpecDriftWithUpdatedArtifactDigest`
  changed the generated resource's field type and updated the `spec.json`
  artifact digest; source-driven compatibility still rejected the drift.
- `git diff --check` — passed.

## Limits

The check establishes consistency between generator source, embedded spec,
version marker, and recorded artifact digests. It does not certify generated
business policy, PostgreSQL RLS, external Editions, or production migrations.

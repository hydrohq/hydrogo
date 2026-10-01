# ADR-0336: Dev-mode license bypass for local-only verification

- **Status:** Accepted
- **Date:** 2026-10-01
- **Supersedes:** None
- **Superseded by:** None

## Context

A scaffolded Client Edition project (`hydro new` + `hydro generate scaffold`)
has no official, vendor-signed `hydro.lic` available outside a real
commercial license purchase, and `hydro-issuer`-issued self-signed licenses
are explicitly rejected by `validateRuntimeLicense` (trusted vendor public
keys are never read from the license file itself, see
`internal/licensing/license.go`). As a result, nobody — not CI, not a new
contributor, not an agent verifying a generator change end to end — can run
`hydro serve` against a freshly generated project and confirm even a basic
`/healthz` response without a real commercial license. This blocks a cheap,
important verification step (`hydro new` → `hydro generate scaffold` → "does
it actually boot?") that has no legitimate reason to require a paid license.

## Decision

Add a `--dev` flag and `HYDRO_DEV_MODE=1` environment variable to `hydro
serve` that bypasses `validateRuntimeLicense` entirely. The bypass is only
permitted when **both** of the following hold, checked unconditionally and
in order whenever `--dev`/`HYDRO_DEV_MODE` is set, regardless of other flags,
proxies, or deployment topology:

1. **The binary is not an official build.** `internal/licensing.TrustedPublicKeyBase64`
   must be empty (`enforceDevModeOfficialBuild`). This variable is injected
   via `-ldflags` only by the official release pipeline; a customer-shipped
   binary always has it populated. If it is non-empty, `--dev`/`HYDRO_DEV_MODE`
   is refused with an explicit error ("dev mode is not available in official
   builds"), never silently ignored.
2. **The server only binds to loopback.** `--address` (and therefore
   `cfg.HTTP.Address`) must resolve to `127.0.0.1`, `localhost`, or `::1`
   (`enforceDevModeLocalBind`). Any other host is rejected.

Both checks must pass before the server starts; if either fails, `serve`
returns an error and never listens. When both pass, a large, impossible-to-
miss "UNLICENSED DEV MODE — NOT FOR PRODUCTION" banner is printed to stderr
(`printDevModeWarning`) before the server starts, and `validateRuntimeLicense`
is skipped for that run only. No change is made to `validateRuntimeLicense`
itself or to any file under `internal/licensing/`.

## Alternatives

- **Document the limitation only, add no bypass.** Rejected: it leaves the
  "can a scaffolded project be verified locally at all" gap open indefinitely
  and keeps blocking cheap CI/local verification work with no benefit to
  license integrity (official binaries are unaffected either way).
- **Loopback bind check only, no official-build check.** Rejected: a loopback
  bind is not a security boundary by itself — it is trivially defeated by a
  reverse proxy or port-forward sitting in front of the process on the same
  host, and nothing would stop `--dev` from working if accidentally left
  enabled in a vendor-built customer binary. The official-build check
  (`TrustedPublicKeyBase64 == ""`) is the real boundary: it guarantees the
  bypass code path is structurally unreachable in anything built by the
  official release pipeline, independent of network topology.
- **Official-build check only, no loopback requirement.** Rejected: even for
  a non-official binary, there is no reason to let an unlicensed server
  accept traffic from outside the local machine; loopback-only keeps the
  blast radius of an accidentally-shared dev machine minimal.

## Consequences

### Benefits

- Scaffolded Client Edition projects can be verified end to end
  (`/healthz`, basic smoke tests) locally and in CI without any commercial
  license, using an ordinary `go build`/`go run` binary.
- Official customer binaries are structurally unaffected: the bypass code
  path is unreachable whenever `TrustedPublicKeyBase64` is set, regardless
  of `--dev`, `--address`, or any reverse proxy configuration.
- `internal/licensing/*.go` and its existing test suite are untouched.

### Costs and risks

- A non-official binary built from source can still start unlicensed if
  deliberately run with `--dev`/`HYDRO_DEV_MODE=1` and bound to loopback —
  this is the intended, bounded scope (local development/verification), not
  a production deployment path.
- Adds one more flag/env var and two small validation functions to maintain.

## Scale impact

- Compact: no change; dev mode only affects local, single-process
  verification runs.
- Standard: not applicable — official builds always carry
  `TrustedPublicKeyBase64`, so dev mode never activates in a standard
  deployment.
- Distributed: not applicable, same reasoning as Standard.

## Database impact

- SQLite: none.
- PostgreSQL: none.
- Future adapters: none; this change touches only CLI flag parsing and an
  early-return gate in `serve`, not persistence.

## Cache impact

None.

## Deployment and compatibility

No migration or route changes. Behavior for existing commands and official
builds is unchanged: `validateRuntimeLicense` runs exactly as before unless
`--dev`/`HYDRO_DEV_MODE=1` is explicitly set, and even then only activates
for non-official, loopback-bound processes.

## Validation

- `cmd/hydro/dev_mode_test.go`: unit tests for `enforceDevModeLocalBind`,
  `enforceDevModeOfficialBuild`, `printDevModeWarning`; in-process rejection
  tests for official-build binaries, non-loopback addresses, and the
  unlicensed-serve regression path (no `--dev`); a subprocess-based end-to-
  end test that boots `hydro serve --dev` (and `HYDRO_DEV_MODE=1`) with no
  `hydro.lic` and confirms `/healthz` responds 200 with the warning banner on
  stderr.
- `go test ./cmd/hydro/... ./internal/licensing/...` — all green;
  `internal/licensing` package and its tests are unmodified by this change.

## References

- `cmd/hydro/main.go` (`serve`, `enforceDevModeLocalBind`,
  `enforceDevModeOfficialBuild`, `printDevModeWarning`)
- `internal/licensing/license.go` (`TrustedPublicKeyBase64`)
- `cmd/hydro/license_cmd.go` (`validateRuntimeLicense`, unchanged)
- `projeler/hydrogo/TASKS.md` — "Scaffold edilen Client Edition projeleri
  lisanssız yerel e2e doğrulamaya kapalı" (brain repo)

# R7 reliability and security hardening validation

**Date:** 2026-09-10  
**Repository commit:** `56ee349`  
**Validation baseline:** `56ee349`  
**Scope:** automated reliability, observability, supply-chain and contributor
verification improvements after the R3/R5 contract work.

## Verified changes

- `/readyz` exposes a bounded `hydro_readiness_state` gauge and a separate
  `hydro_readiness_failures_total` counter. The alarm evaluator uses current
  readiness state, so a recovered instance clears `readiness_failed`.
- CI, local verification and production-gate scripts use bounded Go test
  timeouts. The CI race job runs all `internal/...`, `modules/...`, CLI and
  `tests/contract` packages with `-count=1`.
- External GitHub Actions are pinned to full commit SHAs and `tools/actionguard`
  rejects mutable action or reusable-workflow references.
- Dependabot, CodeQL, `govulncheck`, the dependency review, and the private
  vulnerability reporting policy are present.
- Backup snapshot, restore and retention roots reject empty paths, symlink
  roots and symlinked parent components; retention revalidates the root before
  destructive pruning.
- Release asset retention and generator output roots apply the same fail-closed
  path policy. Distributed rate-limit keys are hashed before reaching a shared
  store, avoiding raw IP/email persistence.
- Backup, erasure and release workflows share the tested
  `internal/pathguard` symlink-component contract; empty path input is rejected
  by the shared helper itself.
- Hook dispatch releases its mutex while handlers run, coalesces in-flight
  duplicate events, and safely handles nested/reentrant dispatch. HTTP error
  buffering preserves the underlying `ResponseWriter` capabilities through
  `Unwrap`.
- Strict JSON request decoding fails closed for nil requests/bodies and invalid
  limits, while retaining the unknown-field, single-value and byte-bound
  contract.
- Release verification bounds operator-provided signed manifests to 1 MiB and
  trusted public-key files to 4 KiB, with regression coverage for oversized
  inputs.
- Restore activation bounds the quarantine marker to 64 KiB, with regression
  coverage for oversized marker input.
- Release asset retention and verification stream artifact bytes through a
  bounded reader and hash instead of loading the full artifact into memory;
  invalid declared sizes are rejected.
- Release manifest generation also streams artifact hashing and rejects
  symlink artifacts instead of following paths outside the release root.
- Generator output paths reject symlinked parents and targets before writing,
  protecting generated files and published migrations from path redirection.
- Generator output directories are checked before `MkdirAll`, preventing a
  symlinked parent from causing external directory creation as a side effect.
- Media upload limits reject the integer-overflow boundary for bounded reads
  and cap concurrent upload slots at 1024, with regression coverage.
- Portable CLI import/export paths reject symlinked source files and destination
  parents before reading or atomically publishing documents.
- Erasure log operations revalidate the log file and parent path before each
  read/write, reject post-construction symlink replacement, and normalize nil
  replay contexts.
- Erasure GC and SMTP/retry senders normalize nil contexts, avoiding runtime
  panics at public cancellation-aware boundaries.

## Evidence commands

The following checks passed on the validation machine:

```text
go test ./... -p 1 -count=1 -timeout=5m
go test ./... -shuffle=on -count=1 -p 1 -timeout=5m
go test ./tools/actionguard -count=1
go run ./tools/actionguard
go test ./tools/importguard -count=1
go vet ./...
git diff --check
```

`cmd/hydro` was additionally run repeatedly with:

```text
go test ./cmd/hydro -count=10 -shuffle=on -timeout=120s
```

The Windows development environment cannot run `go test -race` without a C
compiler. The authoritative race evidence is the Linux CI job, which runs the
complete package scope including `tests/contract`. PostgreSQL contract and
restore evidence still requires the configured PostgreSQL service and
credentials.

## Limitations

This report does not certify a production profile or SLA. Real reverse-proxy
rollback, clean-target restore, external backup/key recovery, production key
ceremony, customer capacity, and RPO/RTO drills remain open in
`docs/validation/PRODUCTION-GATES.md`.

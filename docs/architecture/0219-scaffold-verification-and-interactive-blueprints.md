# ADR-0219: Scaffold Verification and Interactive Blueprint Workflow

## Status

Accepted — 2026-09-22

## Decision

`hydro generate scaffold` formats generated Go files and, if a valid `go.mod` exists, runs a bounded `go test ./...` verification. Commands that mutate external dependency trees (such as `go mod tidy` or automatic tool downloads) are not invoked implicitly. `hydro new resource` produces identical Blueprint declarations through interactive CLI prompts; no secondary code-generation pathway exists.

## Consequences

- Generated scaffolding remains reviewable, deterministic, and compilable.
- Generation hooks never silently alter client module dependencies, schemas, or migrations.

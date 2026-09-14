# R5 configured public home locale validation — 2026-09-14

Related decision: [ADR-0090](../architecture/0090-configured-public-home-locale.md)

- `cmd/hydro/main_test.go` verifies configured `tr-TR`, empty-locale `en`
  fallback, and path escaping.
- The root redirect now uses `cfg.I18N.DefaultLocale` instead of a literal
  `tr` segment.
- `go test ./cmd/hydro -p 1 -count=1` passed.

This validates locale selection only; replacing the demo home slug with a
customer-configured page remains an open product/content gate.

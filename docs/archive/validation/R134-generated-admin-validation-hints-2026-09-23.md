# R134 — Generated Admin validation hints — 2026-09-23

## Scope

Generated Admin forms now project supported string format metadata to
appropriate email/date/URL controls and length metadata to typed input and
textarea components. Server-side `Validate()` remains authoritative; no
client-side validation is treated as a security boundary. The new component
attributes are numeric and emitted only when explicitly configured.

## Verification

- `go test ./internal/generator ./internal/ui/components -count=1` — passed.
- `go test ./... -p 1 -timeout=5m` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.

## Limits

This does not define `datetime-local` timezone conversion, HTML patterns for
Go regular expressions, uniqueness, immutability, or create-time defaults.

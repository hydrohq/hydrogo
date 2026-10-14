# R3 — Referer redirect boundary validation

**Status:** Local contract succeeded; deployment/proxy behavior remains part of
the existing production gate checklist.

The member admin active-state mutation now validates the attacker-controlled
`Referer` header with `security.SafeReturnPath`. External, scheme-relative,
backslash, encoded, and control-character targets fall back to the canonical
member detail path. A transport test exercises a forged external Referer and
asserts that the `Location` header remains local.

Evidence: `go test ./modules/members/transport ./internal/security` passed on
2026-09-14. Full repository pre-push verification is required before delivery.

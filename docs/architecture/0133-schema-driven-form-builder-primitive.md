# ADR-0133: Schema-driven Form Builder primitive

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0121](0121-versioned-forms-contract.md), [ADR-0016](0016-structured-page-blocks-and-theme-slots.md), [ADR-0086](0086-user-facing-error-redaction.md)

## Decision

Admin Studio exposes a reusable `RenderFormBuilder` primitive that edits the
bounded provider-neutral form schema. It supports field type, label, required
flag and select options, emits a live schema payload, and uses a CSP nonce for
its minimal client behavior. All values are HTML/JS escaped and the owning
module remains responsible for authorization, persistence and final contract
validation.

The primitive does not introduce a runtime JavaScript dependency or guess
business rules. Persistent form registry/CRUD wiring and arbitrary visual form
design remain outside this phase.

## Evidence

`internal/adminstudio/form_builder_test.go` covers valid rendering, escaping,
CSP nonce presence and invalid-schema rejection. The component reuses the
existing `internal/forms` bounds, so generated and hand-authored forms share
the same validation contract.

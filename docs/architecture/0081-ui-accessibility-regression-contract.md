# ADR-0081: UI accessibility regression contract

## Status

Accepted

## Context

HydroGo renders the Studio shell and several module surfaces as server-side
HTML. Browser-only audits are valuable but can miss regressions before a
browser is available in local or CI environments.

## Decision

- Every interactive anchor and button emitted by the shared Admin shell must
  have a non-empty accessible name from visible text, `aria-label`, or `title`.
- The shell stylesheet must retain visible `:focus-visible` treatment and a
  reduced-motion media query.
- These checks run as Go regressions against the rendered HTML and CSS. They
  complement, but do not replace, keyboard, contrast, screen-reader, and
  visual browser audits.

## Consequences

New shared controls fail locally when they are unnamed or when accessibility
CSS is accidentally removed. Browser/device-specific behavior remains an
explicit release check.

## Verification

See [`docs/validation/R5-ui-accessibility.md`](../validation/R5-ui-accessibility.md)
and `internal/ui/accessibility_test.go`.

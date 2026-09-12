# Implementation status — 2026-09-12

This document records the current completion boundary for the HydroGo hardening
work. It deliberately separates repository-verifiable changes from gates that
require an external deployment, provider account, or operator-controlled key.

## Completed and locally verified

- Public/member/commerce/admin copy no longer claims a specific TLS/SSL bit
  length, PCI-DSS compliance, invented ratings, or guaranteed delivery time.
- Theme toast messages are inserted as text nodes rather than `innerHTML`.
- Public and admin image fallbacks use delegated, CSP-compatible handlers;
  inline `onerror`/`onclick` handlers were removed from rendered surfaces.
- Deployment presets (`compact-site`, `editorial`, `commerce`, `headless`) are
  explicit and tested for config/module declarations; page routes now respect
  the selected module set.
- Admin Studio search renders stored/API-provided labels with safe DOM text
  nodes instead of interpolating them into `innerHTML`.
- Impeccable detector is clean for the corporate theme; gradient text,
  decorative gradient surfaces, and backdrop blur were removed in favor of
  solid token-based surfaces with lower rendering cost.
- Native forms disable submit controls after the first submit event to reduce
  duplicate mutations.
- Reduced-motion support disables smooth scrolling and long transitions.
- `git diff --check` passes for the edited files.
- `node --check themes/corporate/static/theme.js` passes.
- `go test ./... -p 1 -timeout=5m` passes.
- `go vet ./...` passes.
- Playwright generated frontend/Admin E2E passes: 20/20 tests.

## Not certified by local execution

The following remain operator/environment gates and must not be marked complete
from local tests alone:

- Race tests on Windows require a working C compiler (`CGO_ENABLED=1`); the
  current workstation has no `gcc` in `PATH`.
- Real PostgreSQL restore, clean-target restore, and two-process blue/green
  rollback.
- External backup storage, key recovery, signing-key ceremony, and RPO/RTO.
- SMTP delivery, bounce handling, and provider alerting.
- Live Stripe/iyzico/PayTR sandbox callbacks, refunds, and timeout ambiguity.
- Real Redis/Valkey multi-instance failure and partition drills.
- Customer-capacity measurements for Compact/Standard/Distributed profiles.

These gates are tracked in `docs/ROADMAP.md`,
`docs/validation/PRODUCTION-GATES.md`, and
`docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`.

## Worktree safety

The repository contains pre-existing uncommitted frontend and transport
changes. They were preserved and not reset or folded into an unrelated commit.

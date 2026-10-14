# R5 validation: frontend state conventions

Date: 2026-09-14

- `go test ./internal/ui` passes.
- Shared state validation accepts idle/loading/success/empty/error snapshots
  and rejects unknown states, missing error messages, misplaced error details,
  and invalid optimistic flags.
- Technical error details are excluded from the user-facing contract.

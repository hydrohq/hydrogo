# R61 — Commercial license issuer boundary (2026-09-17)

## Scope

Verified separation between vendor-only license generation and customer runtime verification. This record serves as a technical release gate; it does not substitute for legal counsel review of licensing agreements.

## Executed checks

```text
gofmt -w cmd/hydro/license_cmd.go cmd/hydro/main_test.go cmd/hydro-issuer/main.go
go test ./cmd/hydro ./cmd/hydro-issuer ./internal/licensing
```

Result: PASS.

## Acceptance criteria

- `hydro license issue` and `hydro license keygen` fail in customer runtime: met.
- `hydro-issuer issue` and `hydro-issuer keygen` reside in vendor-only binary: met.
- `hydro license verify/status` continues to verify existing licenses: met.
- Signed `key_id` matching and invalid slot rejection tested: met.
- Dual verification of old/new trusted public keys during short rotation window tested: met.
- Private keys are not used in runtime sources or customer commands in release documentation: met.

## Open operational tasks

- Separating the issuer artifact from customer packages in the release pipeline.
- Rotating official trusted public keys via release secret/ldflags flow.
- Authorized legal counsel review of commercial license and SLA documents.

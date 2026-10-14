# R5 validation: secret rotation and emergency revoke

Date: 2026-09-14

- `go test ./internal/auth` passes.
- Contract tests verify emergency all-session revoke and that only token hashes
  are persisted.
- The runbook explicitly treats MFA master-key rotation as key-bound
  re-enrollment, avoiding an unsafe in-place rotation claim.
- Real secret-manager/key ceremony and deployment reload verification remain
  external gates.

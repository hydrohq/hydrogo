# ADR-0076: Make secret rotation and emergency revoke explicit contracts

## Status

Accepted

## Decision

Session tokens are persisted only as one-way hashes and the auth service exposes
an explicit all-session revoke operation for emergency response. MFA master-key
rotation is treated as a planned re-enrollment ceremony because existing
ciphertext is key-bound; the application must not silently claim in-place
rotation support.

## Consequences

Operators have a repeatable revoke path and a clear safe boundary around key
rotation. Real secret-manager ceremonies, key retention, and deployment
reload behavior remain external gates.

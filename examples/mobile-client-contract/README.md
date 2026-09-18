# HydroGo staff mobile client fixture

This platform-neutral fixture defines the minimum behavior for an iOS or
Android client of the staff API. It is intentionally not a member/customer
authentication contract.

Use [`contract.json`](contract.json) as a test input in the client repository.
Clients must keep both tokens only in platform secure storage, replace the pair
atomically after a successful refresh, and clear all local credentials on a
refresh rejection or token-storage failure. Automatic refresh retry is unsafe:
an uncertain request may already have consumed the single-use refresh token.

The authoritative wire schema is
[`docs/api/mobile-v1.openapi.json`](../../docs/api/mobile-v1.openapi.json).

# R5 — SDK broker grant client

`internal/sdkdist.RequestGrant` provides the client side of the broker contract: bounded JSON request/response, HTTPS/localhost test policy, Authorization header, redirect rejection, and non-200 error mapping. The access token is never written to the response or disk; the caller additionally checks lock context match via `ValidateGrant`.
`RequestValidatedGrant` unifies transport and lock/grant validation into a single fail-closed call, rejecting incorrect project/Core/SDK pairings.
Request coordinates (project/Core/SDK/target/edition) are mandatory before network calls, and feature lists are validated boundedly against duplicates/path content.

Windows:

```text
go test ./internal/sdkdist
PASS
```

WSL Ubuntu (`testuser`):

```text
go test ./internal/sdkdist ./internal/release ./cmd/hydro
exit code 0
```

Tests run successful grant, 403, and oversized response scenarios using `httptest`. Real OAuth client registration and broker deployment are out of scope.
The WSL execution also covers the `RequestValidatedGrant` lock mismatch regression test.
Coordinate/feature preflight tests also passed on WSL with exit code 0.

Current race verification:

```text
go test -race ./internal/sdkdist ./internal/release ./cmd/hydro
exit code 0
```

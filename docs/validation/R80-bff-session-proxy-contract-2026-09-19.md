# R80 — BFF session-to-proxy mutation contract (2026-09-19)

The integration test `TestHandlerAndSessionStoreMutationFlow` exercises the
full local reference chain:

1. create a BFF-only opaque cookie while retaining the Core cookie server-side;
2. submit a browser mutation with the BFF cookie and CSRF header;
3. validate CSRF and resolve the Core session;
4. forward only the versioned API request to Core;
5. verify Core receives its own cookie and no browser `Origin` or
   `Authorization` header;
6. return the upstream status without leaking `Set-Cookie`.

```text
go test ./internal/security/bffproxy ./internal/security/bffsession
```

The test is a local contract, not evidence of a shared production session
backend, TLS/reverse-proxy deployment, login exchange or staging browser E2E.

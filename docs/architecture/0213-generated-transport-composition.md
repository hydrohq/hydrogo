# ADR-0213: Generated transport composition boundary

- **Status:** Accepted
- **Date:** 2026-09-21

## Decision

Blueprint scaffolding emits a dependency-injected JSON CRUD handler alongside
the model, repository, service, and policy contracts. The handler owns only
bounded HTTP decoding, method dispatch, status mapping, and response encoding.
It receives the generated service and policy from the Client Edition
composition root; it does not open databases, execute DDL, infer tenant scope,
or grant authorization.

The generated policy interface is deny-by-default. Edition code must supply
authentication, tenant/member authorization, CSRF or BFF boundary controls,
audit, idempotency, validation, and transaction policy before publishing the
route. The checked-in examples use an explicitly named `demoPolicy` only in
local tests.

## Consequences

- A generated resource can be wired to a real HTTP server without hand-writing
  repetitive transport plumbing.
- Security responsibility remains visible at the Edition boundary instead of
  being guessed by the generator.
- Generated transport tests prove denied access and bounded JSON behavior, while
  production policy tests remain Edition-owned.


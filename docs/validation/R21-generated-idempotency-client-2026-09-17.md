# R21 — Generated client idempotency helper (2026-09-17)

The headless TypeScript starter now emits a provider-neutral mutation helper.
`HydroClient.mutation` supports POST/PUT/PATCH/DELETE, optional abort signals,
JSON bodies, empty 202/204 success responses and the bounded `Idempotency-Key` header. The generated validator
rejects keys outside 16–128 visible ASCII characters before a request is sent.
Caller cancellation is combined with the configured timeout, so supplying an
`AbortSignal` cannot accidentally disable the client's bounded wait.

This is an ergonomic client contract: the server route must opt into durable
idempotency (the Site form pilot does). It does not claim exactly-once external
provider effects and does not store credentials in the scaffold.

Verification: `go test ./internal/projectgen -count=1` — PASS; generated-client
content assertions cover the helper and header contract.

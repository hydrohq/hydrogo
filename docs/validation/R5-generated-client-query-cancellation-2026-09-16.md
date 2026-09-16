# R5 validation: generated client query and cancellation

Date: 2026-09-16

- `go test ./internal/generator` passes.
- Generator tests verify that `client.ts` imports the query helper, accepts an
  optional `AbortSignal`, and attaches encoded query parameters.
- Existing no-argument `list()` calls remain valid.
- Generated clients now expose a typed `HydroAPIError` and parse the stable
  JSON error shape with a 2 MiB body bound; malformed/oversized bodies remain
  safe and do not enter the thrown message.

Browser build and provider authentication remain Client Edition acceptance
gates.

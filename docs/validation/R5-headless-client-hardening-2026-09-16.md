# Headless starter client hardening evidence (2026-09-16)

`client/hydrogo.ts` produced by the generator now applies the same baseline boundaries as generated collection clients:

- 100 ms–30 s range, default 10 s `AbortController` timeout;
- 2 MiB response limit via `Content-Length` and streaming reader;
- bounded locale/slug inputs (256 Unicode characters);
- typed `HydroAPIError` and safe fallback message in HTTP error bodies.

`go test ./internal/projectgen ./internal/generator -count=1` verifies these contract markers in generator output. The generated client file is maintained under `client/` and introduces no dependencies to Core runtime.
